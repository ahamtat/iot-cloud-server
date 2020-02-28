package params

import (
	"context"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
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

// Vocabulary for gateway cameras
type CameraLogicParamsMap map[string]CameraLogicParams

func NewCameraLogicParams(ctx context.Context, gatewayId string) interfaces.LogicParams {
	p := &CameraLogicParams{
		DeviceLogicParams: DeviceLogicParams{ctx: ctx, gatewayId: gatewayId},
	}
	return p
}

func (p *CameraLogicParams) Load() error {
	//
	return nil
}

func (p *CameraLogicParams) ConvertRecordingMode(mode string) (m CloudCameraRecordingMode) {
	switch mode {
	case "continuous":
		m = RecordingModeContinuous
	case "motion":
		m = RecordingModeMotion
	case "schedule":
		m = RecordingModeSchedule
	}
	return
}
