package entities

import "time"

// IoT-gateway message representation
type IotMessage struct {
	Timestamp   time.Time `json:"timestampMs"`
	GatewayId   string    `json:"gatewayId"`
	ClientType  string    `json:"clientType"`
	DeviceId    string    `json:"deviceId"`
	DeviceType  string    `json:"deviceType"`
	Protocol    string    `json:"protocol"`
	MessageType string    `json:"messageType"`
	SensorType  string    `json:"sensorType"`
	SensorData  string    `json:"sensorData"`
	Label       string    `json:"label,omitempty"`
	Units       string    `json:"units,omitempty"`
	Vendor      string    `json:"vendor"`
	Version     string    `json:"version"`
}
