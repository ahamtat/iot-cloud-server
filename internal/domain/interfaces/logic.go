package interfaces

import (
	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
)

// Interface for application business logic
type Logic interface {
	LoadParams() error
	Process(message entities.IotMessage) error
}
