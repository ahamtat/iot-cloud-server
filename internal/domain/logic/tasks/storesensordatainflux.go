package tasks

import (
	"fmt"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/viper"
	"strings"
	"time"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
)

type StoreSensorDataInfluxTask struct {
	connUrl  string
	username string
	password string
}

func NewStoreSensorDataInfluxTask() interfaces.Task {
	return &StoreSensorDataInfluxTask{
		connUrl: fmt.Sprintf("http://%s:%d",
			viper.GetString("db.sensor.host"),
			viper.GetInt("db.sensor.port")),
		username: viper.GetString("db.sensor.username"),
		password: viper.GetString("db.sensor.password"),
	}
}

func (t *StoreSensorDataInfluxTask) Run(message *entities.IotMessage) {
	go func() {
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
			Addr:     t.connUrl,
			Username: t.username,
			Password: t.password,
		})
		if err != nil {
			logger.Error("error creating InfluxDB client", "error", err,
				"caller", "StoreSensorDataInfluxTask")
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
		}

		// Create a point and add to batch
		name := "device_" + strings.ReplaceAll(message.DeviceId, "-", "_")
		tags := map[string]string{
			"class": strings.ReplaceAll(message.SensorType, "-", "_"),
			"label": strings.ReplaceAll(message.Label, "-", "_")}
		if len(message.Units) != 0 {
			tags["units"] = message.Units
		}
		fields := map[string]interface{}{}
		if strings.Contains(message.SensorData, ".") {
			fields["value_float"] = message.SensorData
		} else {
			fields["value"] = message.SensorData
		}
		pt, err := client.NewPoint(name, tags, fields, time.Now())
		if err != nil {
			logger.Error("error creating new point", "error", err,
				"caller", "StoreSensorDataInfluxTask")
		}
		bp.AddPoint(pt)

		// Write the batch
		err = c.Write(bp)
		if err != nil {
			logger.Error("error writing point to InfluxDB", "error", err,
				"caller", "StoreSensorDataInfluxTask")
		}
	}()
}
