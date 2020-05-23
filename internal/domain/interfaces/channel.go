package interfaces

import (
	"io"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"
)

// Channel interface for data exchange between IoT-gateway and cloud server
type Channel interface {
	io.ReadWriteCloser
	Start()
	Stop()
	DoRPC(request *entities.IotMessage) (*entities.IotMessage, error)
}
