package logic

import (
	"context"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
)

type GatewayLogic struct {
	ctx context.Context
}

func NewGatewayLogic(ctx context.Context) interfaces.Logic {
	return &GatewayLogic{ctx: ctx}
}

func (l *GatewayLogic) Process(message entities.IotMessage) error {
	//
	return nil
}
