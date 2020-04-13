package tasks

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/viper"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
)

// StoreSensorDataInfluxTask structure
type StoreSensorDataInfluxTask struct {
	connURL  string
	username string
	password string
}

// NewStoreSensorDataInfluxTask constructs StoreSensorDataInfluxTask
// and returns task interface
func NewStoreSensorDataInfluxTask() interfaces.Task {
	return &StoreSensorDataInfluxTask{
		connURL: fmt.Sprintf("http://%s:%d",
			viper.GetString("db.sensor.host"),
			viper.GetInt("db.sensor.port")),
		username: viper.GetString("db.sensor.username"),
		password: viper.GetString("db.sensor.password"),
	}
}

// Run extracts data from incoming message and send it via HTTP to InfluxDB database
func (t *StoreSensorDataInfluxTask) Run(message *entities.IotMessage) {
	if len(message.GatewayId) == 0 || len(message.DeviceId) == 0 {
		logger.Error("no sender defined", "caller", "StoreSensorDataInfluxTask")
		return
	}
	if message.DeviceType != "sensor" && message.DeviceType != "camera" {
		logger.Error("wrong device type", "deviceType", message.DeviceType,
			"caller", "StoreSensorDataInfluxTask")
		return
	}

	// Create InfluxDB client
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     t.connURL,
		Username: t.username,
		Password: t.password,
	})
	if err != nil {
		logger.Error("error creating InfluxDB client", "error", err,
			"caller", "StoreSensorDataInfluxTask")
		return
	}
	defer c.Close()

	// Create a new point batch
	dbName :=
		"gateway_" + strings.ReplaceAll(message.GatewayId, "-", "_") + "_" +
			message.DeviceType + "s"
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  dbName,
		Precision: "ms",
	})
	if err != nil {
		logger.Error("error creating batch point", "error", err,
			"caller", "StoreSensorDataInfluxTask")
		return
	}

	// Create a point and add it to a batch
	name := "device_" + strings.ReplaceAll(message.DeviceId, "-", "_")
	tags := map[string]string{
		"class": message.GetSensorType(),
		"label": message.GetLabel()}
	if len(message.Units) != 0 {
		tags["units"] = message.Units
	}
	fields := map[string]interface{}{}
	if floatValue, err := strconv.ParseFloat(message.SensorData, 64); err != nil {
		fields["value"] = message.SensorData
	} else {
		fields["value_float"] = floatValue
	}
	//logger.Debug("Point fields", "fields", fields, "caller", "StoreSensorDataInfluxTask")
	pt, err := client.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		logger.Error("error creating new point", "error", err,
			"caller", "StoreSensorDataInfluxTask")
		return
	}
	logger.Debug("New point value", "value", pt,
		"caller", "StoreSensorDataInfluxTask")
	bp.AddPoint(pt)

	// Write the batch
	err = c.Write(bp)
	if err != nil {
		logger.Error("error writing point to InfluxDB", "error", err,
			"caller", "StoreSensorDataInfluxTask")
	}
}
