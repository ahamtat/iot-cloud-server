package messages

import (
	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
)

// NewStatusMessage creates status message
func NewStatusMessage(gatewayID, status string) *entities.IotMessage {

	message := entities.CreateCloudIotMessage(gatewayID, "")
	message.Protocol = "amqp"
	message.Status = status

	return message
}
