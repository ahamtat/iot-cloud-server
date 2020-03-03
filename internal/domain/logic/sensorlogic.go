package logic

import (
	"errors"
	"strings"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic/params"
)

func (l *GatewayLogic) getSensorLogicParams(deviceId string) (*params.SensorLogicParams, error) {
	if l.SensorParams == nil {
		return nil, errors.New("no sensor logic params loaded")
	}

	p, ok := l.SensorParams[deviceId]
	if !ok {
		return nil, errors.New("no logic params for sensor " + deviceId)
	}
	return p, nil
}

func (l *GatewayLogic) processSensorData(message *entities.IotMessage) error {
	sensorLogicParams, err := l.getSensorLogicParams(message.DeviceId)
	if err != nil {
		return err
	}

	// Check sensor type existence
	sensorType := strings.ReplaceAll(message.Label, " ", "_")
	innerParams, ok := sensorLogicParams.ParamsMap[sensorType]
	if !ok {
		return errors.New("no params for sensor: " + sensorType)
	}

	// Store sensor data in MySQL
	message.DeviceTableId = sensorLogicParams.DeviceTableId
	// TODO: Run StoreSensorDataMySql task

	// Store sensor data in InfluxDB
	if innerParams.Influx {
		// TODO: Run StoreSensorDataInflux task
	}

	// Inform user about sensor event
	if message.SensorData == "on" && innerParams.Notify && l.UserParams.Push {
		/*pushMessage :=*/ _ = NewPushMessage(
			"device",
			sensorLogicParams.Title,
			innerParams.Desc,
			sensorLogicParams.DeviceTableId,
			sensorLogicParams.UserId)
		// TODO: Run SendPushNotification task
	}

	return nil
}
