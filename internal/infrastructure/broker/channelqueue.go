package broker

import (
	"fmt"

	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/logger"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

const (
	exchangeName = "veedo.gateways"
)

// ChannelWithQueue keeps RabbitMQ channel and corresponding queue
type ChannelWithQueue struct {
	Ch  *amqp.Channel
	Que amqp.Queue
}

// NewChannelWithQueue function for ChannelWithQueue construction
func NewChannelWithQueue(conn *amqp.Connection, queueName *string) (*ChannelWithQueue, error) {

	// Open channel
	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open a channel")
	}

	// Open exchange
	err = ch.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to declare an exchange")
	}

	// Create queue if its name exists
	var que amqp.Queue
	if queueName != nil {
		que, err = ch.QueueDeclare(
			*queueName, // name
			false,      // durable
			false,      // delete when unused
			true,       // exclusive
			false,      // no-wait
			nil,        // arguments
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to declare a queue")
		}

		// Binding queue to exchange
		routingKey := fmt.Sprintf("gateway.%s", *queueName)
		logger.Debug(
			"Binding queue to exchange with routing key",
			"queue", que.Name,
			"exchange", exchangeName,
			"routing_key", routingKey)
		err = ch.QueueBind(
			que.Name,     // queue name
			routingKey,   // routing key
			exchangeName, // exchange
			false,
			nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to bind a queue")
		}
	}

	return &ChannelWithQueue{
		Ch:  ch,
		Que: que,
	}, nil
}

// Close function releases RabbitMQ channel and corresponding queue
func (cwq *ChannelWithQueue) Close() error {
	// Delete corresponding queue first
	if len(cwq.Que.Name) > 0 {
		_, _ = cwq.Ch.QueueDelete(cwq.Que.Name, false, false, true)
		//if err != nil {
		//	logger.Error("failed deleting queue", "caller", "ChannelWithQueue")
		//}
	}
	if err := cwq.Ch.Close(); err != nil {
		return errors.Wrap(err, "failed closing channel")
	}
	return nil
}
