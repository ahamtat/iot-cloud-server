package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"

	"github.com/spf13/viper"

	"github.com/ahamtat/iot-cloud-server/internal/domain/logic/messages"

	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/tasks"

	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/logger"

	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/database"
	"github.com/pkg/errors"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
	"github.com/ahamtat/iot-cloud-server/internal/domain/interfaces"
	"github.com/ahamtat/iot-cloud-server/internal/domain/logic/params"
)

type GatewayLogic struct {
	ctx          context.Context
	conn         *database.Connection
	gatewayId    string
	CameraParams *params.GuardedParamsMap
	SensorParams *params.GuardedParamsMap
	UserParams   params.UserLogicParams
}

func NewGatewayLogic(ctx context.Context, conn *database.Connection, gatewayId string) interfaces.Logic {
	return &GatewayLogic{
		ctx:          ctx,
		conn:         conn,
		gatewayId:    gatewayId,
		CameraParams: params.NewGuardedParamsMap(),
		SensorParams: params.NewGuardedParamsMap(),
	}
}

func (l *GatewayLogic) LoadParams(writer io.Writer) error {
	var err error

	// Check input params
	if writer == nil {
		return errors.New("wrong input parameter")
	}

	// Wrap context with timeout value for database interactions
	ctx1, cancel1 := context.WithTimeout(l.ctx, viper.GetDuration("db.cloud.timeout"))
	defer cancel1()

	// Load user params
	userParamsQueryText :=
		`SELECT usr.id AS user_id, usr.tfid AS tarif_id, usr.amount AS money, 
			usr.vip, usr.isLegalEntity, usr.blocked, usr.push
		FROM v3_gateways AS gw
			INNER JOIN users AS usr
				ON gw.user_id = usr.id
		WHERE gw.gateway_id = ?;`
	if err = l.conn.Db.GetContext(ctx1, &l.UserParams, userParamsQueryText, l.gatewayId); err != nil {
		return errors.Wrap(err, "failed to query user params")
	}

	// Wrap context with timeout value for database interactions
	ctx2, cancel2 := context.WithTimeout(l.ctx, viper.GetDuration("db.cloud.timeout"))
	defer cancel2()

	// Load cameras params
	cameraParamsQueryText :=
		`SELECT cam.id AS device_table_id, cam.uid AS user_id, cam.stream_id,
			cam.recording, cam.schedule, cam.gateway_id, cam.title
		FROM camers AS cam
			INNER JOIN v3_gateways AS gw
				ON cam.gateway_id = gw.gateway_id
			INNER JOIN users AS usr
				ON cam.uid = usr.id
			WHERE gw.gateway_id = ?;`
	cameraRows, err := l.conn.Db.QueryContext(ctx2, cameraParamsQueryText, l.gatewayId)
	if err != nil {
		return errors.Wrap(err, "failed to query camera params")
	}
	for cameraRows.Next() {
		p := &params.CameraLogicParams{}
		var recMode sql.NullString
		var schedule sql.NullString
		if err = cameraRows.Scan(
			&p.DeviceTableId, &p.UserId, &p.DeviceId,
			&recMode, &schedule, &p.GatewayId, &p.Title); err != nil {
			return errors.Wrap(err, "could not read record data")
		}
		if recMode.Valid {
			p.SetRecordingMode(recMode.String)
		}
		if schedule.Valid {
			p.Schedule = schedule.String
		}
		l.CameraParams.Add(p.DeviceId, p)
	}
	_ = cameraRows.Close()

	// Wrap context with timeout value for database interactions
	ctx3, cancel3 := context.WithTimeout(l.ctx, viper.GetDuration("db.cloud.timeout"))
	defer cancel3()

	// Load sensors params
	sensorParamsQueryText :=
		`SELECT dev.id AS device_table_id, dev.device_id, dev.user_id, dev.title, dev.gateway_id
		FROM v3_devices AS dev
			INNER JOIN v3_gateways AS gw
				ON dev.gateway_id = gw.gateway_id
		WHERE gw.gateway_id = ?;`
	sensorRows, err := l.conn.Db.QueryContext(ctx3, sensorParamsQueryText, l.gatewayId)
	if err != nil {
		return errors.Wrap(err, "failed to query sensor device params")
	}
	for sensorRows.Next() {
		p := &params.SensorLogicParams{
			DeviceLogicParams: params.DeviceLogicParams{},
			Inner:             params.NewGuardedParamsMap(),
		}
		if err = sensorRows.Scan(&p.DeviceTableId, &p.DeviceId, &p.UserId, &p.Title, &p.GatewayId); err != nil {
			return errors.Wrap(err, "could not read record data")
		}

		// Load inner params
		innerParamsQueryText :=
			`SELECT sens.sensor, sens.influx, sens.notify, sens.desc
			FROM v3_sensors AS sens
				INNER JOIN v3_devices AS dev
					ON dev.id = sens.device_id
			WHERE dev.id = ?;`
		innerRows, err := l.conn.Db.QueryContext(ctx3, innerParamsQueryText, p.DeviceTableId)
		if err != nil {
			return errors.Wrap(err, "failed to query sensor inner params")
		}
		for innerRows.Next() {
			var (
				sensorType  string
				description string
			)
			ip := &params.InnerParams{}
			if err = innerRows.Scan(&sensorType, &ip.Influx, &ip.Notify, &description); err != nil {
				return errors.Wrap(err, "could not read record data")
			}
			ip.Desc = getDescription(description, "on")
			p.Inner.Add(sensorType, ip)
		}
		_ = innerRows.Close()

		l.SensorParams.Add(p.DeviceId, p)
	}
	_ = sensorRows.Close()

	logger.Debug("Params for business logic were loaded successfully",
		"gateway", l.gatewayId, "caller", "GatewayLogic")

	// Inform gateway that logic is loaded and it can operate
	statusMessage := messages.NewStatusMessage(l.gatewayId, "registered")
	jsonMessage, err := json.Marshal(statusMessage)
	if err != nil {
		return errors.Wrap(err, "error marshalling JSON")
	}
	if _, err = writer.Write(jsonMessage); err != nil {
		return errors.Wrap(err, "error sending message to gateway")
	}

	logger.Debug("Registered message were sent to gateway",
		"gateway", l.gatewayId, "caller", "GatewayLogic")

	return nil
}

// Extract one value from JSON string
func getDescription(full, value string) string {
	m := map[string]string{}
	if err := json.Unmarshal([]byte(full), &m); err != nil {
		return ""
	}
	out, ok := m[value]
	if !ok {
		return ""
	}
	return out
}

func (l *GatewayLogic) Process(message *entities.IotMessage) error {
	// Check input params
	if message == nil {
		return errors.New("wrong input parameter")
	}
	// Check if user is blocked
	if l.UserParams.Blocked {
		logger.Info("Gateway owner's account is blocked in cloud database")
		return nil
	}
	//// Check device id
	//if len(message.DeviceId) == 0 {
	//	return errors.New("no device defined in message")
	//}

	var err error
	switch message.MessageType {
	case "status":
		// Update gateway status in MySQL database
		go tasks.NewUpdateGatewayStatusTask(l.conn).Run(message)
	case "sensorData":
		switch message.DeviceType {
		case "camera":
			err = l.processCameraData(message)
		case "sensor":
			err = l.processSensorData(message)
		}
	case "preview":
		// Store camera image preview in database
		go tasks.NewStorePreviewTask(l.conn).Run(message)
	case "command":
		err = l.processCameraCommand(message)
	case "deviceState":
		if message.DeviceType == "camera" {
			err = l.processCameraState(message)
		}
	case "cloudStreaming":
		if message.DeviceType == "camera" {
			// Update camera streaming state in MySQL database
			go tasks.NewUpdateCameraStreamingStateTask(l.conn).Run(message)
		}
	case "configurationData":
		if message.DeviceType == "gateway" {
			// SetGatewayConfigure(message)
		}
	}
	return err
}

// SetPush changes push flag for user params
func (l *GatewayLogic) SetPush(state bool) {
	l.UserParams.Push = state
}
