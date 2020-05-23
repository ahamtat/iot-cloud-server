package interfaces

import "github.com/ahamtat/iot-cloud-server/internal/domain/entities"

type Task interface {
	Run(message *entities.IotMessage)
}
