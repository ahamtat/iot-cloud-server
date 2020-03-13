package broker

import (
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"

	"github.com/streadway/amqp"
)

type AmqpWriter struct {
	cwq        *ChannelWithQueue
	routingKey string
}

func NewAmqpWriter(conn *amqp.Connection, gatewayId string) io.WriteCloser {

	// Create amqp channel and queue
	ch, err := NewChannelWithQueue(conn, nil)
	if err != nil {
		logger.Error("failed creating amqp channel and queue",
			"error", err, "gateway", gatewayId,
			"caller", "NewAmqpWriter")
		return nil
	}

	return &AmqpWriter{
		cwq:        ch,
		routingKey: fmt.Sprintf("gateway.%s.in", gatewayId),
	}
}

// Write message to RabbitMQ broker.
// Returns message length on success or error if any
func (w *AmqpWriter) Write(p []byte) (n int, err error) {
	if w.cwq.Ch == nil {
		return 0, errors.New("no output channel defined")
	}

	// Send message to gateway
	err = w.cwq.Ch.Publish(
		exchangeName, // exchange
		w.routingKey, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        p,
		})
	if err != nil {
		return 0, errors.Wrap(err, "failed to publish a message")
	}

	return len(p), nil
}

func (w *AmqpWriter) Close() error {
	if err := w.cwq.Close(); err != nil {
		return errors.Wrap(err, "failed closing gateway input channel")
	}
	//logger.Info("Gateway input channel closed")
	return nil
}
