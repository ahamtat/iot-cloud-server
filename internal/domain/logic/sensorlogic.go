package logic

import (
	"errors"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
	"github.com/ahamtat/iot-cloud-server/internal/domain/logic/messages"
	"github.com/ahamtat/iot-cloud-server/internal/domain/logic/params"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/tasks"
)

func (l *GatewayLogic) getSensorLogicParams(deviceId string) (*params.SensorLogicParams, error) {
	if l.SensorParams == nil {
		return nil, errors.New("no sensor logic params loaded")
	}

	p, ok := l.SensorParams.Get(deviceId)
	if !ok {
		return nil, errors.New("no logic params for sensor " + deviceId)
	}
	pc, ok := p.(*params.SensorLogicParams)
	if !ok {
		return nil, errors.New("error casting interface to SensorLogicParams")
	}
	return pc, nil
}

func (l *GatewayLogic) processSensorData(message *entities.IotMessage) error {
	sensorLogicParams, err := l.getSensorLogicParams(message.DeviceId)
	if err != nil {
		return err
	}

	// Check sensor existence
	label := message.GetLabel()
	something, ok := sensorLogicParams.Inner.Get(label)
	if !ok {
		return errors.New("no params for sensor: " + label)
	}
	innerParams, ok := something.(*params.InnerParams)
	if !ok {
		return errors.New("error casting interface to InnerParams")
	}

	// Store sensor data in MySQL
	message.DeviceTableId = sensorLogicParams.DeviceTableId
	go tasks.NewStoreSensorDataMySqlTask(l.conn).Run(message)

	// Store sensor data in InfluxDB
	if innerParams.Influx {
		go tasks.NewStoreSensorDataInfluxTask().Run(message)
	}

	// Inform user about sensor event
	if message.SensorData == "on" && innerParams.Notify && l.UserParams.Push {
		pushMessage := messages.NewPushMessage(
			"sensor",
			sensorLogicParams.Title,
			innerParams.Desc,
			sensorLogicParams.DeviceTableId,
			sensorLogicParams.UserId)
		go tasks.NewSendPushNotificationTask(l.conn).Run(pushMessage)
	}

	return nil
}
