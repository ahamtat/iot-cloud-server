package entities

import (
	"strconv"
	"strings"
	"time"
)

// IoT-gateway message representation
type IotMessage struct {
	Timestamp       string `json:"timestampMs,omitempty"`
	Vendor          string `json:"vendor,omitempty"`
	Version         string `json:"version,omitempty"`
	GatewayId       string `json:"gatewayId,omitempty"`
	ClientType      string `json:"clientType,omitempty"`
	DeviceId        string `json:"deviceId,omitempty"`
	DeviceType      string `json:"deviceType,omitempty"`
	DeviceState     string `json:"deviceState,omitempty"`
	DeviceTableId   uint64 `json:"deviceState,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
	MessageType     string `json:"messageType,omitempty"`
	SensorType      string `json:"sensorType,omitempty"`
	SensorData      string `json:"sensorData,omitempty"`
	Preview         string `json:"preview,omitempty"`
	Label           string `json:"label,omitempty"`
	Value           string `json:"value,omitempty"`
	Units           string `json:"units,omitempty"`
	MediaserverIp   string `json:"mediaserverIp,omitempty"`
	ApplicationName string `json:"applicationName,omitempty"`
	Recording       string `json:"recording,omitempty"`
	Command         string `json:"command,omitempty"`
	Attribute       string `json:"attribute,omitempty"`
	TariffId        uint64 `json:"tariffId,omitempty"`
	Money           uint64 `json:"money,omitempty"`
	Vip             bool   `json:"vip,omitempty"`
	LegalEntity     bool   `json:"isLegalEntity,omitempty"`
	UserId          uint64 `json:"userId,omitempty"`
	Title           string `json:"title,omitempty"`
	Content         string `json:"content,omitempty"`
	Status          string `json:"status,omitempty"`
}

func (m IotMessage) GetSensorType() string {
	return strings.ReplaceAll(m.Label, " ", "_")
}

func CreateTimestampMs(t time.Time) string {
	return strconv.Itoa(int(t.UnixNano() / int64(time.Millisecond)))
}
