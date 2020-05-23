package interfaces

import (
	"io"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
)

// Interface for application business logic
type Logic interface {
	LoadParams(writer io.Writer) error
	Process(message *entities.IotMessage) error
	SetPush(state bool)
}
