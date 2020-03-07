package logic

import (
	"errors"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic/messages"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic/params"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic/tasks"
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
	sensorType := message.GetSensorType()
	innerParams, ok := sensorLogicParams.ParamsMap[sensorType]
	if !ok {
		return errors.New("no params for sensor: " + sensorType)
	}

	// Store sensor data in MySQL
	message.DeviceTableId = sensorLogicParams.DeviceTableId
	tasks.NewStoreSensorDataMySqlTask(l.conn).Run(message)

	// Store sensor data in InfluxDB
	if innerParams.Influx {
		tasks.NewStoreSensorDataInfluxTask().Run(message)
	}

	// Inform user about sensor event
	if message.SensorData == "on" && innerParams.Notify && l.UserParams.Push {
		pushMessage := messages.NewPushMessage(
			"device",
			sensorLogicParams.Title,
			innerParams.Desc,
			sensorLogicParams.DeviceTableId,
			sensorLogicParams.UserId)
		tasks.NewSendPushNotificationTask(l.conn).Run(pushMessage)
	}

	return nil
}
