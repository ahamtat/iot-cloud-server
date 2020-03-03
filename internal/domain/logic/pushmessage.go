package logic

type PushMessage struct {
	DeviceType    string
	DeviceTableId uint64
	UserId        uint64
	Title         string
	Content       string
}

func NewPushMessage(deviceType, title, content string, deviceTableId, userId uint64) *PushMessage {
	return &PushMessage{
		DeviceType:    deviceType,
		DeviceTableId: deviceTableId,
		UserId:        userId,
		Title:         title,
		Content:       content,
	}
}
