package messages

import "github.com/AcroManiac/iot-cloud-server/internal/domain/entities"

func NewPushMessage(deviceType, title, content string, deviceTableId, userId uint64) *entities.IotMessage {
	return &entities.IotMessage{
		DeviceType:    deviceType,
		DeviceTableId: deviceTableId,
		UserId:        userId,
		Title:         title,
		Content:       content,
	}
}
