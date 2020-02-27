package interfaces

import "github.com/AcroManiac/iot-cloud-server/internal/domain/entities"

// Interface for application business logic
type Logic interface {
	Process(message entities.IotMessage) error
}
