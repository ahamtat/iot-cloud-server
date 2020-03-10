package broker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"

	"github.com/streadway/amqp"
)

type Manager struct {
	ServerId string
	Protocol string
	User     string
	Password string
	Host     string
	Port     int
	// TODO: Make separate AMQP connections and channels for publish/subscribe concurrency
	// TODO: https://github.com/streadway/amqp/issues/327
	Conn    *amqp.Connection
	Ch      *amqp.Channel
	evQue   amqp.Queue
	evChan  <-chan amqp.Delivery
	gwChans GatewayChannelsMap
}

func NewManager(serverId, protocol, user, password, host string, port int) *Manager {
	return &Manager{
		ServerId: serverId,
		Protocol: protocol,
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
	}
}

func (m *Manager) Open() error {
	var err error
	connUrl := fmt.Sprintf("%s://%s:%s@%s:%d/", m.Protocol, m.User, m.Password, m.Host, m.Port)

	// Open connection to broker
	m.Conn, err = amqp.Dial(connUrl)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to connect to RabbitMQ: %v", err))
	}

	// Open channel
	m.Ch, err = m.Conn.Channel()
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to open a channel: %v", err))
	}

	// Open exchange
	err = m.Ch.ExchangeDeclare(
		"veedo.gateways", // name
		"topic",          // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to declare an exchange: %v", err))
	}

	return nil
}

func (m *Manager) Close() error {
	// Close gateways
	for _, ch := range m.gwChans.GetChannels() {
		if err := ch.Close(); err != nil {
			return nil
		}
	}

	// Close channel
	if m.Ch != nil {
		if err := m.Ch.Close(); err != nil {
			return err
		}
	}
	// Close connection
	if m.Conn != nil {
		if err := m.Conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) EventExchangeInit() error {
	// Check if connection established
	if m.Conn == nil || m.Ch == nil {
		return errors.New("no connection to RabbitMQ broker")
	}

	// Create queue
	var err error
	m.evQue, err = m.Ch.QueueDeclare(
		m.ServerId, // name
		false,      // durable
		false,      // delete when unused
		true,       // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		logger.Error("Failed to declare an event queue", "error", err)
		return err
	}

	// Binding queue to exchange
	exchange := "amq.rabbitmq.event"
	routing := "queue.*"
	logger.Debug(
		"Binding queue to exchange with routing key",
		"queue", m.evQue.Name, "exchange", exchange, "routing_key", routing)
	err = m.Ch.QueueBind(
		m.evQue.Name, // queue name
		routing,      // routing key
		exchange,     // exchange
		false,
		nil)
	if err != nil {
		logger.Error("Failed to bind an event queue", "error", err)
		return err
	}

	// Create consuming channel
	m.evChan, err = m.Ch.Consume(
		m.evQue.Name, // queue
		"",           // consumer
		true,         // auto ack
		false,        // exclusive
		false,        // no local
		false,        // no wait
		nil,          // args
	)
	if err != nil {
		logger.Error("Failed to register an event consumer", "error", err)
		return err
	}

	return nil
}

// Read one message from RabbitMQ event exchange.
// Returns message length in bytes
func (m *Manager) Read(p []byte) (n int, err error) {
	message, ok := <-m.evChan
	if ok {
		n = copy(p, message.Headers["name"].(string))
	}
	return
}

type ExchangeEvent map[string]string

func (m *Manager) ReadExchangeEvent(ctx context.Context) (ee ExchangeEvent, err error) {
	select {
	case <-ctx.Done():
		err = errors.New("interrupted reading exchange")
	case message, ok := <-m.evChan:
		if ok {
			ee = ExchangeEvent{
				"eventType": message.RoutingKey,
				"queueName": message.Headers["name"].(string),
			}
		} else {
			err = errors.New("error reading exchange event")
		}
	}
	return
}

func (m *Manager) ProcessExchangeEvents(ctx context.Context, conn *database.Connection) {
	for {
		ee, err := m.ReadExchangeEvent(ctx)
		if err != nil {
			//logger.Error("Error while reading event", "error", err)
			continue
		}

		logger.Debug("RabbitMQ event", "event", ee)

		if eventType, ok := ee["eventType"]; ok {
			queueName, ook := ee["queueName"]
			if !ook {
				logger.Error("Error reading queue name from event exchange")
				continue
			}
			strArr := strings.Split(queueName, ".")
			gatewayId := strArr[0]
			if len(strArr) > 1 && strArr[1] == "in" {
				switch eventType {
				case "queue.created":
					ch := NewGatewayChannel(m.Ch, m.ServerId, gatewayId, conn)
					ch.Start()
					m.gwChans.Add(gatewayId, ch)
				case "queue.deleted":
					if ch := m.gwChans.Get(gatewayId); ch != nil {
						_ = ch.Close()
						m.gwChans.Remove(gatewayId)
					} else {
						logger.Error("No stored gateway info", "gateway", gatewayId)
					}
				}
			}
		}
	}
}
