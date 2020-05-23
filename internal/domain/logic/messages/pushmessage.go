package messages

import "github.com/ahamtat/iot-cloud-server/internal/domain/entities"

// NewPushMessage create message for Push Notification service
func NewPushMessage(deviceType, title, content string, deviceTableId, userId uint64) *entities.IotMessage {
	return &entities.IotMessage{
		DeviceType:    deviceType,
		DeviceTableId: deviceTableId,
		UserId:        userId,
		Title:         title,
		Content:       content,
	}
}
