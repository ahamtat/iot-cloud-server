package params

import (
	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
)

type CloudCameraRecordingMode uint

const (
	RecordingModeNoRecord CloudCameraRecordingMode = iota
	RecordingModeContinuous
	RecordingModeMotion
	RecordingModeSchedule
)

// Parameters for cloud camera business logic
type CameraLogicParams struct {
	DeviceLogicParams
	RecordingMode        CloudCameraRecordingMode
	Schedule             string
	MediaserverParamsSet bool
	MediaserverIp        string
	ApplicationName      string
	MotionInProcess      bool
}

func (p *CameraLogicParams) SetRecordingMode(mode string) {
	p.RecordingMode = p.ConvertRecordingMode(mode)
}

func (p CameraLogicParams) ConvertRecordingMode(mode string) CloudCameraRecordingMode {
	var recordingMode CloudCameraRecordingMode
	switch mode {
	case "continuous":
		recordingMode = RecordingModeContinuous
	case "motion":
		recordingMode = RecordingModeMotion
	case "schedule":
		recordingMode = RecordingModeSchedule
	}
	return recordingMode
}

func (p CameraLogicParams) ToMessage(recording bool) *entities.IotMessage {
	recMode := "off"
	if recording {
		recMode = "on"
	}

	message := entities.CreateCloudIotMessage("", p.DeviceId)
	message.MediaserverIp = p.MediaserverIp
	message.ApplicationName = p.ApplicationName
	message.Recording = recMode

	return message
}
