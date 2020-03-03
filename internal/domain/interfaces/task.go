package interfaces

import "github.com/AcroManiac/iot-cloud-server/internal/domain/entities"

type Task interface {
	Run(message *entities.IotMessage)
}
