package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/pkg/errors"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"

	"github.com/streadway/amqp"
)

// AmqpReader structure for reading channel
type AmqpReader struct {
	gatewayID string
	ctx       context.Context
	cwq       *ChannelWithQueue
	msgs      <-chan amqp.Delivery
}

// NewAmqpReader function for AmqpReader structure construction
func NewAmqpReader(ctx context.Context, conn *amqp.Connection, gatewayID string) *AmqpReader {

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
		gatewayID: gatewayID,
		ctx:       ctx,
		cwq:       ch,
		msgs:      msgs,
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

// ReadEnvelope reads and unmarshals message from RabbitMQ queue. Returns message envelope or error
func (r *AmqpReader) ReadEnvelope() (env *AmqpEnvelope, close bool, err error) {
	select {
	case <-r.ctx.Done():
		logger.Debug("Context cancelled", "caller", "ReadEnvelope")
		close = true
	case message, ok := <-r.msgs:
		if ok {
			inputMessage := entities.IotMessage{}

			// Create message buffer
			bodyLength := len(message.Body)
			buffer := make([]byte, bodyLength)
			n := copy(buffer, message.Body)
			if n != bodyLength {
				err = errors.Wrap(err, "error copying message body to buffer")
				return
			}

			// Unmarshal input message from JSON to structure
			err = json.Unmarshal(buffer, &inputMessage)
			if err != nil {
				err = errors.Wrap(err, "can not unmarshal incoming gateway message")
				return
			}

			// Print copy of incoming message to log
			r.PrintMessage(inputMessage)

			// Create envelope
			env = &AmqpEnvelope{
				Message: &inputMessage,
				Metadata: &AmqpMetadata{
					CorrelationID: message.CorrelationId,
					ReplyTo:       "",
				},
			}
		}
	}
	return
}

// PrintMessage prints incoming message to log
func (r AmqpReader) PrintMessage(message entities.IotMessage) {
	// Slim long preview
	if len(message.Preview) > 0 {
		message.Preview = "Some Base64 code ;)"
	}
	logger.Debug("Message from gateway", "message", message, "gateway", r.gatewayID)
}

// Close function releases RabbitMQ channel and corresponding queue
func (r *AmqpReader) Close() error {
	if err := r.cwq.Close(); err != nil {
		return errors.Wrap(err, "failed closing gateway output channel")
	}
	return nil
}
