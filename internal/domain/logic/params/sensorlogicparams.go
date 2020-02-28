package params

import (
	"context"
	"time"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
)

type InnerParams struct {
	Influx    bool
	Notify    bool
	Desc      string
	Value     string
	Timestamp time.Time
}

type InnerParamsMap map[string]InnerParams

// Parameters for sensor business logic
type SensorLogicParams struct {
	DeviceLogicParams
	ParamsMap InnerParamsMap
}

// Vocabulary for gateway cameras
type SensorLogicParamsMap map[string]SensorLogicParams

func NewSensorLogicParams(ctx context.Context, gatewayId string) interfaces.LogicParams {
	p := &SensorLogicParams{
		DeviceLogicParams: DeviceLogicParams{ctx: ctx, gatewayId: gatewayId},
	}
	return p
}

func (p *SensorLogicParams) Load() error {
	//
	return nil
}
