package tasks

import (
	"context"
	"time"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
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
	go func() {
		if len(message.GatewayId) == 0 || len(message.DeviceId) == 0 {
			logger.Error("no sender defined", "caller", "StoreSensorDataMySqlTask")
			return
		}
		if message.DeviceType != "sensor" {
			logger.Error("wrong device type", "deviceType", message.DeviceType,
				"caller", "StoreSensorDataMySqlTask")
			return
		}

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		updateQueryText :=
			`update v3_sensors
			set value = ?, updated_at = now()
			where device_id = ? and sensor = ?`
		_, err := t.conn.Db.ExecContext(ctx, updateQueryText,
			message.SensorData, message.DeviceTableId, message.GetLabel())
		if err != nil {
			logger.Error("error updating sensors", "error", err, "caller", "StoreSensorDataMySqlTask")
		}
	}()
}
