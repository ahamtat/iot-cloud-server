package messages

import (
	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"time"
)

func NewStatusMessage(gatewayId, status string) *entities.IotMessage {
	return &entities.IotMessage{
		Timestamp:  entities.CreateTimestampMs(time.Now()),
		Vendor:     "Veedo",
		Version:    "3.1.0",
		GatewayId:  gatewayId,
		ClientType: "veedoCloud",
		Protocol:   "amqp",
		Status:     status,
	}
}
