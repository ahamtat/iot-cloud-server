package tasks

import (
	"context"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
	"time"
)

type StorePreviewTask struct {
	conn *database.Connection
}

func NewStorePreviewTask(conn *database.Connection) interfaces.Task {
	if conn == nil {
		logger.Error("database connection is nil")
	}
	return &StorePreviewTask{conn: conn}
}

func (t *StorePreviewTask) Run(message *entities.IotMessage) {
	go func() {
		if len(message.GatewayId) == 0 || len(message.DeviceId) == 0 {
			logger.Error("no sender defined", "caller", "StorePreviewTask")
			return
		}
		if len(message.Preview) == 0 {
			logger.Error("no preview in message", "caller", "StorePreviewTask")
			return
		}

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		updateQueryText := `update camers set preview = '?' where stream_id = '?'`
		_, err := t.conn.Db.ExecContext(ctx, updateQueryText, message.Preview, message.DeviceId)
		if err != nil {
			logger.Error("error updating preview", "error", err, "caller", "StorePreviewTask")
		}
	}()
}
