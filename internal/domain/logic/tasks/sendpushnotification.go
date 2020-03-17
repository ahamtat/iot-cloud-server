package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/viper"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
)

// SendPushNotificationTask structure
type SendPushNotificationTask struct {
	conn       *database.Connection
	host       string
	requestUri string
	appId      string
	restApiKey string
}

// RequestBodyHeadings nested structures to marshal valid JSON request body
type RequestBodyHeadings struct {
	En string `json:"en"`
	Ru string `json:"ru"`
}

type RequestBodyContents struct {
	En string `json:"en"`
	Ru string `json:"ru"`
}

type RequestBodyData struct {
	DeviceType string `json:"deviceType"`
	DeviceId   uint64 `json:"deviceId"`
}

type RequestBody struct {
	AppId            string              `json:"app_id"`
	IncludePlayerIds []string            `json:"include_player_ids"`
	Headings         RequestBodyHeadings `json:"headings"`
	Contents         RequestBodyContents `json:"contents"`
	Data             RequestBodyData     `json:"data"`
}

func NewSendPushNotificationTask(conn *database.Connection) interfaces.Task {
	if conn == nil {
		logger.Error("database connection is nil", "caller", "NewSendPushNotificationTask")
	}
	return &SendPushNotificationTask{
		conn:       conn,
		host:       viper.GetString("push.host"),
		requestUri: viper.GetString("push.requestUri"),
		appId:      viper.GetString("push.appId"),
		restApiKey: viper.GetString("push.restApiKey"),
	}
}

func (t *SendPushNotificationTask) Run(message *entities.IotMessage) {
	go func() {
		// Check input data
		if len(message.GatewayId) == 0 || len(message.DeviceId) == 0 {
			logger.Error("no sender defined", "caller", "SendPushNotificationTask")
			return
		}
		if message.DeviceType != "sensor" {
			logger.Error("wrong device type", "deviceType", message.DeviceType,
				"caller", "SendPushNotificationTask")
			return
		}

		// Get Player Ids for sending push notifications to user mobile device
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		queryText := `select player_id from v3_playerids where user_id = ?`
		rows, err := t.conn.Db.QueryContext(ctx, queryText, message.UserId)
		if err != nil {
			logger.Error("error selecting player ids",
				"error", err, "caller", "SendPushNotificationTask")
		}
		defer func() { _ = rows.Close() }()

		playerIds := make([]string, 8)
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				logger.Error("error reading player id",
					"error", err, "caller", "SendPushNotificationTask")
				continue
			}
			playerIds = append(playerIds, id)
		}

		// Check for user devices
		if len(playerIds) == 0 {
			logger.Debug("Skip sending message. No player Ids for user",
				"user", message.UserId, "caller", "SendPushNotificationTask")
			return
		}

		// Create message content
		content := time.Now().Format("15:04:05") + "   " + message.Content

		// Create Push notification request
		// https://documentation.onesignal.com/reference#create-notification

		// Create request body first
		requestBody, err := json.Marshal(&RequestBody{
			AppId:            t.appId,
			IncludePlayerIds: playerIds,
			Headings: RequestBodyHeadings{
				En: message.Title,
				Ru: message.Title,
			},
			Contents: RequestBodyContents{
				En: content,
				Ru: content,
			},
			Data: RequestBodyData{
				DeviceType: message.DeviceType,
				DeviceId:   message.DeviceTableId,
			},
		})
		if err != nil {
			logger.Error("could not marshal request body",
				"error", err, "caller", "SendPushNotificationTask")
			return
		}

		// Create request
		request, err := http.NewRequest("POST", t.host, bytes.NewBuffer(requestBody))
		if err != nil {
			logger.Error("failed to create http request",
				"error", err, "caller", "SendPushNotificationTask")
			return
		}
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
		request.Header.Set("authorization", "Basic "+t.restApiKey)

		// Create HTTP client
		client := http.Client{
			Timeout: time.Duration(5 * time.Second),
		}

		// Send request
		resp, err := client.Do(request)
		if err != nil || resp == nil {
			logger.Error("error while sending Push notification",
				"error", err,
				"caller", "SendPushNotificationTask")
		}
		defer func() {
			if resp != nil {
				_ = resp.Body.Close()
			}
		}()

		// Read response for debugging purposes
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Error("failed reading response", "error", err, "caller", "SendPushNotificationTask")
			return
		}
		logger.Debug("Received response status code", "response", string(respBody), "caller", "SendPushNotificationTask")
	}()
}
