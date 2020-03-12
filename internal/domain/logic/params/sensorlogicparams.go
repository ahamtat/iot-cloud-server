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

// Parameters for sensor business logic
type SensorLogicParams struct {
	DeviceLogicParams
	Inner *GuardedParamsMap
}
