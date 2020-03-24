package messages

import (
	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
)

// NewStatusMessage creates status message
func NewStatusMessage(gatewayID, status string) *entities.IotMessage {

	message := entities.CreateCloudIotMessage(gatewayID, "")
	message.Protocol = "amqp"
	message.Status = status

	return message
}
