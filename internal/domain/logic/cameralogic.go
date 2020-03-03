package logic

import (
	"errors"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic/params"
)

func (l *GatewayLogic) getCameraLogicParams(deviceId string) (*params.CameraLogicParams, error) {
	if l.CameraParams == nil {
		return nil, errors.New("no camera logic params loaded")
	}

	p, ok := l.CameraParams[deviceId]
	if !ok {
		return nil, errors.New("no logic params for camera " + deviceId)
	}
	return p, nil
}

func (l *GatewayLogic) processCameraState(message *entities.IotMessage) error {
	cameraLogicParams, err := l.getCameraLogicParams(message.DeviceId)
	if err != nil {
		return err
	}

	// TODO: Run UpdateCameraState task
	// Update camera state in database

	switch message.DeviceState {
	case "streamingOn":
		// Check tariff restrictions first
		if !l.UserParams.CanBeRecorded() {
			return nil
		}
		message.Recording = "on"
		// Store mediaserver params
		cameraLogicParams.MediaserverIp = message.MediaserverIp
		cameraLogicParams.ApplicationName = message.ApplicationName
		cameraLogicParams.MediaserverParamsSet = true
	case "streamingOff":
		message.Recording = "off"
		cameraLogicParams.MediaserverIp = ""
		cameraLogicParams.ApplicationName = ""
		cameraLogicParams.MediaserverParamsSet = false
	default:
		return errors.New("wrong deviceState: " + message.DeviceState)
	}

	// Check recording mode
	switch cameraLogicParams.RecordingMode {
	case params.RecordingModeContinuous:
		// TODO: Run RecordMediaStream task
	case params.RecordingModeMotion:
		if cameraLogicParams.MotionInProcess {
			// TODO: Run RecordMediaStream task
		}
	}

	return nil
}

func (l *GatewayLogic) processCameraData(message *entities.IotMessage) error {
	cameraLogicParams, err := l.getCameraLogicParams(message.DeviceId)
	if err != nil {
		return err
	}

	// If recording is motion and detector message then do recording
	if message.Label == "motionDetector" && cameraLogicParams.RecordingMode == params.RecordingModeMotion {
		cameraLogicParams.MotionInProcess = message.SensorData == "on"
		if cameraLogicParams.MediaserverParamsSet {
			message.Recording = message.SensorData
			// TODO: Run RecordMediaStream task
		}
	}

	// TODO: Run StoreSensorDataInflux task

	// Inform user about motion detection
	if message.Label == "motionDetector" && message.SensorData == "on" && l.UserParams.Push {
		/*pushMessage :=*/ _ = NewPushMessage(
			"camera",
			cameraLogicParams.Title,
			"Обнаружено движение",
			cameraLogicParams.DeviceTableId,
			cameraLogicParams.UserId)
		// TODO: Run SendPushNotification task
	}

	return nil
}

func (l *GatewayLogic) processCameraCommand(message *entities.IotMessage) error {
	cameraLogicParams, err := l.getCameraLogicParams(message.DeviceId)
	if err != nil {
		return err
	}

	if message.Command == "setRecording" {
		newRecordingMode := cameraLogicParams.ConvertRecordingMode(message.Attribute)
		currentRecordingMode := cameraLogicParams.RecordingMode

		prevTariffId := l.UserParams.TarifId

		// Update user params
		l.UserParams.TarifId = message.TariffId
		l.UserParams.Money = message.Money
		l.UserParams.Vip = message.Vip
		l.UserParams.LegalEntity = message.LegalEntity

		if currentRecordingMode == params.RecordingModeMotion &&
			newRecordingMode == params.RecordingModeContinuous &&
			l.UserParams.CanBeRecorded() {
			// Start recording on Wowza
			/*recordingCommand :=*/
			_ = cameraLogicParams.ToMessage(true)
			// TODO: Run RecordMediaStream task
		} else if currentRecordingMode == params.RecordingModeContinuous &&
			newRecordingMode == params.RecordingModeMotion {
			// Stop recording on Wowza
			/*recordingCommand :=*/
			_ = cameraLogicParams.ToMessage(false)
			// TODO: Run RecordMediaStream task
		} else if currentRecordingMode == newRecordingMode {
			// Process user with online tariff
			if prevTariffId == params.UserTarifOnline && l.UserParams.CanBeRecorded() {
				// Start recording on Wowza
				/*recordingCommand :=*/
				_ = cameraLogicParams.ToMessage(true)
				// TODO: Run RecordMediaStream task
			}
			if prevTariffId > params.UserTarifOnline && message.TariffId == params.UserTarifOnline {
				// Stop recording on Wowza
				/*recordingCommand :=*/
				_ = cameraLogicParams.ToMessage(false)
				// TODO: Run RecordMediaStream task
			}
		}
	}

	return nil
}
