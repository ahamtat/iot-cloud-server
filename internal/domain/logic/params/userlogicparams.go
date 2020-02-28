package params

import (
	"context"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
)

const UserTarifOnline = 1

// Parameters for user business logic
type UserLogicParams struct {
	ctx         context.Context
	gatewayId   string
	UserId      uint64
	TarifId     uint64
	Money       uint64
	Vip         bool
	LegalEntity bool
	Blocked     bool
	Push        bool
}

func NewUserLogicParams(ctx context.Context, gatewayId string) interfaces.LogicParams {
	p := &UserLogicParams{ctx: ctx, gatewayId: gatewayId}
	return p
}

func (p *UserLogicParams) Load() error {
	//
	return nil
}

func (p *UserLogicParams) CanBeRecorded() bool {
	return (p.TarifId > UserTarifOnline && p.Money > 0) || p.Vip || p.LegalEntity
}
