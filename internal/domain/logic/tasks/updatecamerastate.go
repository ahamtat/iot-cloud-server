package tasks

import (
	"context"
	"strings"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/viper"
)

type UpdateCameraStateTask struct {
	conn *database.Connection
}

func NewUpdateCameraStateTask(conn *database.Connection) interfaces.Task {
	if conn == nil {
		logger.Error("database connection is nil", "caller", "UpdateCameraStateTask")
	}
	return &UpdateCameraStateTask{conn: conn}
}

func (t *UpdateCameraStateTask) Run(message *entities.IotMessage) {
	go func() {
		if len(message.GatewayId) == 0 || len(message.DeviceId) == 0 {
			logger.Error("no sender defined", "caller", "UpdateCameraStateTask")
			return
		}

		// Wrap context with timeout value for database interactions
		ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("db.cloud.timeout"))
		defer cancel()

		// Create update query
		var onair int
		if message.DeviceState == "on" || message.DeviceState == "streamingOn" {
			onair = 1
		}
		var updateQueryText string
		if strings.Contains(message.DeviceState, "streaming") {
			updateQueryText =
				`update camers
				set onair = ?, ip = ?, server_ip = ?, application = ?
				where stream_id = ?`
			_, err := t.conn.Db.ExecContext(ctx, updateQueryText,
				onair, message.MediaserverIp, message.MediaserverIp,
				message.ApplicationName, message.DeviceId)
			if err != nil {
				logger.Error("error updating cameras", "error", err, "caller", "UpdateCameraStateTask")
			}
		} else {
			updateQueryText = `update camers set onair = ? where stream_id = ?`
			_, err := t.conn.Db.ExecContext(ctx, updateQueryText,
				onair, message.DeviceId)
			if err != nil {
				logger.Error("error updating cameras", "error", err, "caller", "UpdateCameraStateTask")
			}
		}
	}()
}
