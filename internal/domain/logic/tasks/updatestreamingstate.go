package tasks

import (
	"context"
	"time"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
)

type UpdateCameraStreamingStateTask struct {
	conn *database.Connection
}

func NewUpdateCameraStreamingStateTask(conn *database.Connection) interfaces.Task {
	if conn == nil {
		logger.Error("database connection is nil", "caller", "UpdateCameraStreamingStateTask")
	}
	return &UpdateCameraStreamingStateTask{conn: conn}
}

func (t *UpdateCameraStreamingStateTask) Run(message *entities.IotMessage) {
	go func() {
		if len(message.GatewayId) == 0 || len(message.DeviceId) == 0 {
			logger.Error("no sender defined", "caller", "UpdateCameraStreamingStateTask")
			return
		}

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

		// Create update query
		var onair int
		if message.DeviceState == "streamingOn" {
			onair = 1
		}
		updateQueryText := `update camers set onair = ? where stream_id = ?`
		_, err := t.conn.Db.ExecContext(ctx, updateQueryText, onair, message.DeviceId)
		if err != nil {
			logger.Error("error updating camera streaming state",
				"error", err, "caller", "UpdateCameraStreamingStateTask")
		}
	}()
}
