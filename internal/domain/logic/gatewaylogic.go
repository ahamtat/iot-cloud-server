package logic

import (
	"context"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic/params"
)

type GatewayLogic struct {
	ctx          context.Context
	gatewayId    string
	CameraParams params.CameraLogicParamsMap
	SensorParams params.SensorLogicParamsMap
	UserParams   params.UserLogicParams
}

func NewGatewayLogic(ctx context.Context, gatewayId string) interfaces.Logic {
	return &GatewayLogic{ctx: ctx, gatewayId: gatewayId}
}

func (l *GatewayLogic) LoadParams() error {
	return nil
}

func (l *GatewayLogic) Process(message entities.IotMessage) error {
	//
	return nil
}
