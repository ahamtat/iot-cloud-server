package params

import (
	"time"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
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
	return &entities.IotMessage{
		Timestamp:       entities.CreateTimestampMs(time.Now()),
		Vendor:          "Veedo",
		Version:         entities.VeedoVersion,
		ClientType:      "veedoCloud",
		DeviceId:        p.DeviceId,
		MediaserverIp:   p.MediaserverIp,
		ApplicationName: p.ApplicationName,
		Recording:       recMode,
	}
}
