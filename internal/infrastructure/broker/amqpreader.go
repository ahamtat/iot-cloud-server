package broker

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"

	"github.com/streadway/amqp"
)

// AmqpReader structure for reading channel
type AmqpReader struct {
	ctx  context.Context
	cwq  *ChannelWithQueue
	msgs <-chan amqp.Delivery
}

// NewAmqpReader function for AmqpReader structure construction
func NewAmqpReader(ctx context.Context, conn *amqp.Connection, gatewayID string) io.ReadCloser {

	// Create amqp channel and queue
	queueName := fmt.Sprintf("%s.out", gatewayID)
	ch, err := NewChannelWithQueue(conn, &queueName)
	if err != nil {
		logger.Error("failed creating amqp channel and queue",
			"error", err, "queue", queueName, "gateway", gatewayID,
			"caller", "NewAmqpReader")
		return nil
	}

	// Create consuming channel
	msgs, err := ch.Ch.Consume(
		ch.Que.Name, // queue
		"",          // consumer
		true,        // auto ack
		true,        // exclusive
		false,       // no local
		false,       // no wait
		nil,         // args
	)
	if err != nil {
		logger.Error("failed to register a consumer",
			"error", err, "queue", queueName, "gateway", gatewayID,
			"caller", "NewAmqpReader")
		return nil
	}

	// Return reader object
	//logger.Info("Gateway output channel created", "gateway", gatewayID, "queue", ch.Que.Name)
	return &AmqpReader{
		ctx:  ctx,
		cwq:  ch,
		msgs: msgs,
	}
}

// Read one message from RabbitMQ queue. Returns message length in bytes
func (r *AmqpReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		logger.Debug("Context cancelled", "caller", "AmqpReader")
	case message, ok := <-r.msgs:
		if ok {
			n = copy(p, message.Body)
		}
	}
	return
}

// Close function releases RabbitMQ channel and corresponding queue
func (r *AmqpReader) Close() error {
	if err := r.cwq.Close(); err != nil {
		return errors.Wrap(err, "failed closing gateway output channel")
	}
	//logger.Info("Gateway output channel closed")
	return nil
}
