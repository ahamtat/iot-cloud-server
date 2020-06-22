package tasks

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
	"github.com/ahamtat/iot-cloud-server/internal/domain/interfaces"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/database"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/viper"
)

type StoreSensorDataMySqlTask struct {
	conn *database.Connection
}

func NewStoreSensorDataMySqlTask(conn *database.Connection) interfaces.Task {
	if conn == nil {
		logger.Error("database connection is nil", "caller", "StoreSensorDataMySqlTask")
	}
	return &StoreSensorDataMySqlTask{conn: conn}
}

func (t *StoreSensorDataMySqlTask) Run(message *entities.IotMessage) {
	if len(message.GatewayId) == 0 || len(message.DeviceId) == 0 {
		logger.Error("no sender defined", "caller", "StoreSensorDataMySqlTask")
		return
	}
	if message.DeviceType != "sensor" {
		logger.Error("wrong device type", "deviceType", message.DeviceType,
			"caller", "StoreSensorDataMySqlTask")
		return
	}

	// Round long float values to 2 decimal places
	value := message.SensorData
	if len(value) > 10 {
		if floatValue, err := strconv.ParseFloat(message.SensorData, 64); err == nil {
			value = fmt.Sprintf("%.2f", math.Round(floatValue*100)/100)
		}
	}

	// Wrap context with timeout value for database interactions
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("db.cloud.timeout"))
	defer cancel()

	updateQueryText :=
		`update v3_sensors
			set value = ?, updated_at = now()
			where device_id = ? and sensor = ?`
	_, err := t.conn.Db.ExecContext(ctx, updateQueryText,
		value, message.DeviceTableId, message.GetLabel())
	if err != nil {
		logger.Error("error updating sensors", "error", err, "caller", "StoreSensorDataMySqlTask")
	}
}
