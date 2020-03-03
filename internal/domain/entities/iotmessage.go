package entities

import "time"

// IoT-gateway message representation
type IotMessage struct {
	Timestamp       time.Time `json:"timestampMs"`
	Vendor          string    `json:"vendor"`
	Version         string    `json:"version"`
	GatewayId       string    `json:"gatewayId"`
	ClientType      string    `json:"clientType"`
	DeviceId        string    `json:"deviceId"`
	DeviceType      string    `json:"deviceType"`
	DeviceState     string    `json:"deviceState,omitempty"`
	DeviceTableId   uint64    `json:"deviceState,omitempty"`
	Protocol        string    `json:"protocol,omitempty"`
	MessageType     string    `json:"messageType"`
	SensorType      string    `json:"sensorType,omitempty"`
	SensorData      string    `json:"sensorData,omitempty"`
	Label           string    `json:"label,omitempty"`
	Units           string    `json:"units,omitempty"`
	MediaserverIp   string    `json:"mediaserverIp,omitempty"`
	ApplicationName string    `json:"applicationName,omitempty"`
	Recording       string    `json:"recording,omitempty"`
	Command         string    `json:"command,omitempty"`
	Attribute       string    `json:"attribute,omitempty"`
	TariffId        uint64    `json:"tariffId,omitempty"`
	Money           uint64    `json:"money,omitempty"`
	Vip             bool      `json:"vip,omitempty"`
	LegalEntity     bool      `json:"isLegalEntity,omitempty"`
}
