package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/viper"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
	"github.com/ahamtat/iot-cloud-server/internal/domain/interfaces"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/database"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/logger"
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

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
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
	// Check input data
	if message.DeviceTableId == 0 {
		logger.Error("no sender defined", "caller", "SendPushNotificationTask")
		return
	}
	if message.DeviceType != "camera" && message.DeviceType != "sensor" {
		logger.Error("wrong device type", "deviceType", message.DeviceType,
			"caller", "SendPushNotificationTask")
		return
	}

	// Wrap context with timeout value for database interactions
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("db.cloud.timeout"))
	defer cancel()

	// Get Player Ids for sending push notifications to user mobile device
	queryText := `select player_id, device_ids from v3_playerids where user_id = ?`
	rows, err := t.conn.Db.QueryContext(ctx, queryText, message.UserId)
	if err != nil {
		logger.Error("error selecting player ids",
			"error", err, "caller", "SendPushNotificationTask")
	}
	defer func() { _ = rows.Close() }()

	playerIds := make([]string, 0, 8)
	for rows.Next() {
		var id string
		var deviceIds string
		if err = rows.Scan(&id, &deviceIds); err != nil {
			logger.Error("error reading player id",
				"error", err, "caller", "SendPushNotificationTask")
			continue
		}
		if deviceIds == "" {
			playerIds = append(playerIds, id)
		} else {
			var result map[string]interface{}
			json.Unmarshal([]byte(deviceIds), &result)
			if contains(message.DeviceTableId, result[message.DeviceType].(map[string]interface{})) {
				playerIds = append(playerIds, id)
			}
		}
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
	logger.Debug("request payload", "request", string(requestBody))
	if err != nil {
		logger.Error("could not marshal request body",
			"error", err, "caller", "SendPushNotificationTask")
		return
	}

	// Create request
	request, err := http.NewRequest("POST", t.host+t.requestUri, bytes.NewBuffer(requestBody))
	if err != nil {
		logger.Error("failed to create http request",
			"error", err, "caller", "SendPushNotificationTask")
		return
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	request.Header.Set("Authorization", "Basic "+t.restApiKey)

	// Create HTTP client
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	// Send request
	resp, err := client.Do(request)
	if err != nil || resp == nil {
		logger.Error("error while sending Push notification",
			"error", err,
			"caller", "SendPushNotificationTask")
		return
	}
	defer func() {
		if resp != nil {
			_ = resp.Body.Close()
		}
	}()

	// Read response for debugging purposes
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("failed reading response",
			"error", err, "caller", "SendPushNotificationTask")
		return
	}
	logger.Debug("Received response status code",
		"response", string(respBody), "caller", "SendPushNotificationTask")
}
