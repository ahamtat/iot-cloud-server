package params

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
type CameraLogicParamsMap map[string]*CameraLogicParams

func (p *CameraLogicParams) SetRecordingMode(mode string) {
	switch mode {
	case "continuous":
		p.RecordingMode = RecordingModeContinuous
	case "motion":
		p.RecordingMode = RecordingModeMotion
	case "schedule":
		p.RecordingMode = RecordingModeSchedule
	}
	return
}
