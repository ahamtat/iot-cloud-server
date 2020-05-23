package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
	"github.com/ahamtat/iot-cloud-server/internal/domain/interfaces"
	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/viper"
)

// RecordMediaStreamTask structure
type RecordMediaStreamTask struct {
	username string
	password string
	port     int
}

// NewRecordMediaStreamTask constructs RecordMediaStreamTask
// and returns task interface
func NewRecordMediaStreamTask() interfaces.Task {
	return &RecordMediaStreamTask{
		username: viper.GetString("wowza.user"),
		password: viper.GetString("wowza.password"),
		port:     viper.GetInt("wowza.port"),
	}
}

// Run extracts data from incoming message and
// send RESTful Wowza recording commands
func (t *RecordMediaStreamTask) Run(message *entities.IotMessage) {
	if len(message.DeviceId) == 0 {
		logger.Error("no sender defined",
			"message", message, "caller", "RecordMediaStreamTask")
		return
	}

	recOn := message.Recording == "on"

	// Make Wowza RESTful API recording URI
	uri := fmt.Sprintf(
		"http://%s:%d/v2/servers/_defaultServer_/vhosts/_defaultVHost_/applications/%s/instances/_definst_/streamrecorders/%s",
		message.MediaserverIp, t.port, message.ApplicationName, message.DeviceId)
	if !recOn {
		uri += "/actions/stopRecording"
	}

	// Create request data body
	var requestBody []byte
	var err error
	if recOn {
		requestBody, err = json.Marshal(map[string]interface{}{
			"instanceName":            "_definst_",
			"fileVersionDelegateName": "ru.veedo.v3.VeedoFileVersionDelegate",
			"serverName":              "",
			"recorderName":            message.DeviceId,
			"segmentSchedule":         "",
			"outputPath":              "",
			"currentFile":             "",
			"applicationName":         message.ApplicationName,
			"fileTemplate":            "",
			"segmentationType":        "SegmentByDuration",
			"fileFormat":              "MP4",
			"recorderState":           "",
			"option":                  "",

			"currentSize":     0,
			"segmentSize":     0,
			"segmentDuration": 1800000, // 30 minutes
			"backBufferTime":  0,
			"currentDuration": 0,

			"startOnKeyFrame":           true,
			"recordData":                false,
			"moveFirstVideoFrameToZero": true,
			"defaultRecorder":           false,
			"splitOnTcDiscontinuity":    false,
		})
		if err != nil {
			logger.Error("could not marshal request body",
				"error", err, "caller", "RecordMediaStreamTask")
			return
		}
	}
	logger.Debug("Sending Wowza recording command", "uri", uri, "request", requestBody,
		"caller", "RecordMediaStreamTask")

	// Create request
	method := "POST"
	if !recOn {
		method = "PUT"
	}
	request, err := http.NewRequest(method, uri, bytes.NewBuffer(requestBody))
	if err != nil {
		logger.Error("failed to create http request",
			"error", err, "caller", "RecordMediaStreamTask")
		return
	}
	request.SetBasicAuth(t.username, t.password)
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	request.Header.Set("Accept", "application/json; charset=utf-8")

	// Create HTTP client
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	// Send request
	resp, err := client.Do(request)
	if err != nil || resp == nil {
		logger.Error("error while sending Wowza recording command",
			"error", err,
			"caller", "RecordMediaStreamTask")
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
		logger.Error("failed reading response", "error", err, "caller", "RecordMediaStreamTask")
		return
	}
	logger.Debug("Wowza response", "response", string(respBody), "caller", "RecordMediaStreamTask")
}
