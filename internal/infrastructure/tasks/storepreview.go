package tasks

import (
	"context"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/viper"
)

type StorePreviewTask struct {
	conn *database.Connection
}

func NewStorePreviewTask(conn *database.Connection) interfaces.Task {
	if conn == nil {
		logger.Error("database connection is nil", "caller", "StorePreviewTask")
	}
	return &StorePreviewTask{conn: conn}
}

func (t *StorePreviewTask) Run(message *entities.IotMessage) {
	if len(message.GatewayId) == 0 || len(message.DeviceId) == 0 {
		logger.Error("no sender defined", "caller", "StorePreviewTask")
		return
	}
	if len(message.Preview) == 0 {
		logger.Error("no preview in message", "caller", "StorePreviewTask")
		return
	}

	// Wrap context with timeout value for database interactions
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("db.cloud.timeout"))
	defer cancel()

	updateQueryText := `update camers set preview = ? where stream_id = ?`
	_, err := t.conn.Db.ExecContext(ctx, updateQueryText, message.Preview, message.DeviceId)
	if err != nil {
		logger.Error("error updating preview", "error", err, "caller", "StorePreviewTask")
	}
}
