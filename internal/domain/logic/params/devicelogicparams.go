package params

import (
	"context"
)

// Basic structure for device business logic parameters
type DeviceLogicParams struct {
	ctx           context.Context
	gatewayId     string
	DeviceTableId uint64
	UserId        uint64
	DeviceId      string
	Title         string
}
