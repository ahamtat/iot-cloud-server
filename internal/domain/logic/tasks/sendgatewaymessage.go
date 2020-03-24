package tasks

import (
	"encoding/json"
	"io"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
)

type SendGatewayMessageTask struct {
	gatewayChannel io.ReadWriteCloser
}

func NewSendGatewayMessageTask(gatewayChannel io.ReadWriteCloser) interfaces.Task {
	return &SendGatewayMessageTask{gatewayChannel: gatewayChannel}
}

func (t *SendGatewayMessageTask) Run(message *entities.IotMessage) {
	go func() {
		if message == nil {
			logger.Error("nothing to send", "caller", "SendGatewayMessageTask")
			return
		}
		if len(message.GatewayId) == 0 {
			logger.Error("no gateway defined", "caller", "SendGatewayMessageTask")
			return
		}
		if t.gatewayChannel == nil {
			logger.Error("no gateway channel to send a message", "caller", "SendGatewayMessageTask")
			return
		}

		// Marshal message to JSON
		buffer, err := json.Marshal(message)
		if err != nil {
			logger.Error("failed marshalling message to JSON",
				"error", err, "caller", "SendGatewayMessageTask")
			return
		}

		// Send JSON to RabbitMQ broker
		n, err := t.gatewayChannel.Write(buffer)
		if err != nil || n != len(buffer) {
			logger.Error("error sending message to broker",
				"error", err, "caller", "SendGatewayMessageTask")
		}
	}()
}
