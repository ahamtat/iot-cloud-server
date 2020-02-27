package broker

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"

	"github.com/streadway/amqp"
)

type AmqpReader struct {
	ch   *amqp.Channel
	q    amqp.Queue
	msgs <-chan amqp.Delivery
	ctx  context.Context
}

func NewAmqpReader(ctx context.Context, ch *amqp.Channel, gatewayId string) io.ReadCloser {
	var err error

	// Create reader object
	r := &AmqpReader{}
	r.ctx = ctx

	// Create queue
	r.ch = ch
	queueName := fmt.Sprintf("%s.out", gatewayId)
	r.q, err = r.ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		true,      // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		logger.Error("failed to declare a queue", "error", err)
		return nil
	}

	// Binding queue to exchange
	exchange := "veedo.gateways"
	routing := fmt.Sprintf("gateway.%s.out", gatewayId)
	logger.Debug(
		"Binding queue to exchange with routing key",
		"queue", r.q.Name,
		"exchange", exchange,
		"routing_key", routing)
	err = r.ch.QueueBind(
		r.q.Name, // queue name
		routing,  // routing key
		exchange, // exchange
		false,
		nil)
	if err != nil {
		logger.Error("Failed to bind a queue", "error", err)
		return nil
	}

	// Create consuming channel
	r.msgs, err = r.ch.Consume(
		r.q.Name, // queue
		"",       // consumer
		true,     // auto ack
		false,    // exclusive
		false,    // no local
		false,    // no wait
		nil,      // args
	)
	if err != nil {
		logger.Error("Failed to register a consumer", "error", err)
		return nil
	}

	logger.Info("Gateway output channel created", "gateway", gatewayId, "queue", r.q.Name)
	return r
}

// Read one message from RabbitMQ queue. Returns message length in bytes
func (r *AmqpReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		err = errors.New("context cancelled")
	case message, ok := <-r.msgs:
		if ok {
			n = copy(p, message.Body)
		}
	}
	return
}

func (r *AmqpReader) Close() error {
	_, err := r.ch.QueueDelete(r.q.Name, false, false, true)
	logger.Info("Gateway output channel closed", "queue", r.q.Name)
	return err
}
