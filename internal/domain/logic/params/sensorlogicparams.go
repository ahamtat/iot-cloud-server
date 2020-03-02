package params

import (
	"time"
)

type InnerParams struct {
	Influx    bool
	Notify    bool
	Desc      string
	Value     string
	Timestamp time.Time
}

type InnerParamsMap map[string]*InnerParams

// Parameters for sensor business logic
type SensorLogicParams struct {
	DeviceLogicParams
	ParamsMap InnerParamsMap
}

// Vocabulary for gateway cameras
type SensorLogicParamsMap map[string]*SensorLogicParams
