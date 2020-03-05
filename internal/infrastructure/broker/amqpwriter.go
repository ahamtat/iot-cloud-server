package broker

import (
	"errors"
	"fmt"
	"io"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"

	"github.com/streadway/amqp"
)

type AmqpWriter struct {
	ch      *amqp.Channel
	routing string
}

func NewAmqpWriter(ch *amqp.Channel, gatewayId string) io.Writer {
	w := &AmqpWriter{}
	w.ch = ch
	w.routing = fmt.Sprintf("gateway.%s.in", gatewayId)
	return w
}

// Write message to RabbitMQ broker.
// Returns message length on success or error if any
func (w *AmqpWriter) Write(p []byte) (n int, err error) {
	if w.ch == nil {
		err = errors.New("no output channel defined")
		logger.Error("error writing message", "error", err)
		return 0, err
	}

	// Send message to gateway
	err = w.ch.Publish(
		"veedo.gateways", // exchange
		w.routing,        // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        p,
		})
	if err != nil {
		logger.Error("failed to publish a message", "error", err)
		return 0, err
	}

	return len(p), nil
}
