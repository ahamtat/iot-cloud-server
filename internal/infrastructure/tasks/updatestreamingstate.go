package tasks

import (
	"context"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
	"github.com/ahamtat/iot-cloud-server/internal/domain/interfaces"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/database"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/viper"
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
	if len(message.GatewayId) == 0 || len(message.DeviceId) == 0 {
		logger.Error("no sender defined", "caller", "UpdateCameraStreamingStateTask")
		return
	}

	// Wrap context with timeout value for database interactions
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("db.cloud.timeout"))
	defer cancel()

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
}
