package tasks

import (
	"context"

	"github.com/spf13/viper"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
	"github.com/ahamtat/iot-cloud-server/internal/domain/interfaces"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/database"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/logger"
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

	// Wrap context with timeout value for database interactions
	ctx1, cancel1 := context.WithTimeout(context.Background(), viper.GetDuration("db.cloud.timeout"))
	defer cancel1()

	// Update gateway status in database
	updateQueryText := `update v3_gateways set status = ? where gateway_id = ?`
	_, err := t.conn.Db.ExecContext(ctx1, updateQueryText, message.Status, message.GatewayId)
	if err != nil {
		logger.Error("error updating gateway status in database",
			"error", err, "gateway", message.GatewayId, "caller", "UpdateGatewayStatusTask")
	} else {
		logger.Debug("Gateway status updated in database", "status", message.Status,
			"gateway", message.GatewayId, "caller", "UpdateGatewayStatusTask")
	}

	// Wrap context with timeout value for database interactions
	ctx2, cancel2 := context.WithTimeout(context.Background(), viper.GetDuration("db.cloud.timeout"))
	defer cancel2()

	// Update devices statuses
	updateQueryText = `update v3_devices set state = ? where gateway_id = ?`
	_, err = t.conn.Db.ExecContext(ctx2, updateQueryText, statusInt, message.GatewayId)
	if err != nil {
		logger.Error("error updating devices statuses in database",
			"error", err, "caller", "UpdateGatewayStatusTask")
	} else {
		logger.Debug("Devices statuses updated in database", "status", statusInt,
			"gateway", message.GatewayId, "caller", "UpdateGatewayStatusTask")
	}

	// Update cameras off statuses
	if statusInt == 0 {
		// Wrap context with timeout value for database interactions
		ctx3, cancel3 := context.WithTimeout(context.Background(), viper.GetDuration("db.cloud.timeout"))
		defer cancel3()

		updateQueryText = `update camers set onair = ? where gateway_id = ?`
		_, err = t.conn.Db.ExecContext(ctx3, updateQueryText, statusInt, message.GatewayId)
		if err != nil {
			logger.Error("error updating cameras off statuses in database",
				"error", err, "caller", "UpdateGatewayStatusTask")
		} else {
			logger.Debug("Cameras statuses updated in database", "status", statusInt,
				"gateway", message.GatewayId, "caller", "UpdateGatewayStatusTask")
		}
	}
}
