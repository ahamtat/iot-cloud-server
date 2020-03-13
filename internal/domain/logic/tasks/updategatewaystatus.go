package tasks

import (
	"context"
	"time"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
	"github.com/google/uuid"
)

type UpdateGatewayStatusTask struct {
	conn *database.Connection
}

func NewUpdateGatewayStatusTask(conn *database.Connection) interfaces.Task {
	if conn == nil {
		logger.Error("database connection is nil", "caller", "UpdateGatewayStatusTask")
	}
	return &UpdateGatewayStatusTask{conn: conn}
}

func (t *UpdateGatewayStatusTask) Run(message *entities.IotMessage) {
	go func() {
		// Check input data
		if _, err := uuid.Parse(message.GatewayId); err != nil {
			logger.Error("wrong gateway id",
				"gateway", message.GatewayId,
				"caller", "UpdateGatewayStatusTask")
			return
		}

		var statusInt int
		switch message.Status {
		case "off":
			statusInt = 0
		case "on":
			statusInt = 1
		default:
			logger.Error("wrong gateway status",
				"status", message.Status, "caller", "UpdateGatewayStatusTask")
			return
		}

		// Update gateway status in database
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		updateQueryText := `update v3_gateways set status = ? where gateway_id = ?`
		_, err := t.conn.Db.ExecContext(ctx, updateQueryText, message.Status, message.GatewayId)
		if err != nil {
			logger.Error("error updating gateway status in database",
				"error", err, "gateway", message.GatewayId, "caller", "UpdateGatewayStatusTask")
		} else {
			logger.Debug("Gateway status updated in database", "status", message.Status,
				"gateway", message.GatewayId, "caller", "UpdateGatewayStatusTask")
		}

		// Update devices statuses
		ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
		updateQueryText = `update v3_devices set state = ? where gateway_id = ?`
		_, err = t.conn.Db.ExecContext(ctx, updateQueryText, statusInt, message.GatewayId)
		if err != nil {
			logger.Error("error updating devices statuses in database",
				"error", err, "caller", "UpdateGatewayStatusTask")
		} else {
			logger.Debug("Devices statuses updated in database", "status", statusInt,
				"gateway", message.GatewayId, "caller", "UpdateGatewayStatusTask")
		}

		// Update cameras off statuses
		if statusInt == 0 {
			ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
			updateQueryText = `update camers set onair = ? where gateway_id = ?`
			_, err = t.conn.Db.ExecContext(ctx, updateQueryText, statusInt, message.GatewayId)
			if err != nil {
				logger.Error("error updating cameras off statuses in database",
					"error", err, "caller", "UpdateGatewayStatusTask")
			} else {
				logger.Debug("Cameras statuses updated in database", "status", statusInt,
					"gateway", message.GatewayId, "caller", "UpdateGatewayStatusTask")
			}
		}
	}()
}
