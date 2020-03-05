package interfaces

import (
	"io"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
)

// Interface for application business logic
type Logic interface {
	LoadParams(writer io.Writer) error
	Process(message *entities.IotMessage) error
}
