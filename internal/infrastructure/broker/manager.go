package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"

	"github.com/pkg/errors"

	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/database"

	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/logger"

	"github.com/streadway/amqp"
)

// Manager structure keeps parameters for AMQP connection,
// events exchange channel and queue and gateway channels map
type Manager struct {
	ServerID string
	Protocol string
	User     string
	Password string
	Host     string
	Port     int
	CtlPort  int
	Conn     *amqp.Connection
	Ch       *amqp.Channel
	evQue    amqp.Queue
	evChan   <-chan amqp.Delivery
	gwChans  *GatewayChannelsMap
}

// NewManager constructs Manager structure with AMQP connection parameters
func NewManager(ServerID, protocol, user, password, host string, port, ctlPort int) *Manager {
	return &Manager{
		ServerID: ServerID,
		Protocol: protocol,
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		CtlPort:  ctlPort,
		gwChans:  NewGatewayChannelsMap(),
	}
}

// Open AMQP connection and channel for events exchange
func (m *Manager) Open() error {
	var err error
	connURL := fmt.Sprintf("%s://%s:%s@%s:%d/", m.Protocol, m.User, m.Password, m.Host, m.Port)

	// Open connection to broker
	m.Conn, err = amqp.Dial(connURL)
	if err != nil {
		return errors.Wrap(err, "failed connecting to RabbitMQ")
	}

	// Open channel
	m.Ch, err = m.Conn.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to open a channel")
	}

	return nil
}

// Close gateway channels, event exchange and AMQP connection
func (m *Manager) Close() error {
	// Close gateways
	for _, ch := range m.gwChans.GetChannels() {
		if ch == nil {
			continue
		}
		if err := ch.Close(); err != nil {
			return errors.Wrap(err, "error closing stored gateway channel")
		}
	}

	// Delete corresponding queue first
	if len(m.evQue.Name) > 0 {
		_, err := m.Ch.QueueDelete(m.evQue.Name, false, false, true)
		if err != nil {
			logger.Error("failed deleting queue", "caller", "Manager")
		}
	}

	// Close channel
	if m.Ch != nil {
		if err := m.Ch.Close(); err != nil {
			return errors.Wrap(err, "error closing management channel")
		}
	}
	// Close connection
	if m.Conn != nil {
		if err := m.Conn.Close(); err != nil {
			return errors.Wrap(err, "error closing connection to broker")
		}
	}
	return nil
}

// EventExchangeInit creates queue and consumer for events exchange
func (m *Manager) EventExchangeInit() error {
	// Check if connection established
	if m.Conn == nil || m.Ch == nil {
		return errors.New("no connection to RabbitMQ broker")
	}

	// Create queue
	var err error
	m.evQue, err = m.Ch.QueueDeclare(
		m.ServerID, // name
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

	logger.Info("Event exchange manager started")
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

type exchangeEvent map[string]string

func (m *Manager) readExchangeEvent(ctx context.Context) (ee exchangeEvent, err error) {
	select {
	case <-ctx.Done():
		err = errors.New("interrupted reading exchange")
	case message, ok := <-m.evChan:
		if ok {
			ee = exchangeEvent{
				"eventType": message.RoutingKey,
				"queueName": message.Headers["name"].(string),
			}
		} else {
			err = errors.New("error reading exchange event")
		}
	}
	return
}

// ProcessExchangeEvents reads exchange event from queue and processes it
func (m *Manager) ProcessExchangeEvents(ctx context.Context, dbConn *database.Connection) {
	for {
		ee, err := m.readExchangeEvent(ctx)
		if err != nil {
			//logger.Error("error while reading event", "error", err)
			continue
		}

		logger.Debug("RabbitMQ event", "event", ee)

		if eventType, ok := ee["eventType"]; ok {
			queueName, ook := ee["queueName"]
			if !ook {
				logger.Error("error reading queue name from event exchange")
				continue
			}
			strArr := strings.Split(queueName, ".")
			gatewayID := strArr[0]
			if len(strArr) > 1 && strArr[1] == "in" {
				switch eventType {
				case "queue.created":
					ch := NewGatewayChannel(m.Conn, dbConn, m.ServerID, gatewayID)
					ch.Start()
					m.gwChans.Add(gatewayID, ch)
				case "queue.deleted":
					if ch := m.gwChans.Get(gatewayID); ch != nil {
						_ = ch.Close()
						m.gwChans.Remove(gatewayID)
					} else {
						logger.Error("no stored gateway info", "gateway", gatewayID)
					}
				}
			}
		}
	}
}

// DoGatewayRPC sends command for gateway via RabbitMQ broker and
// blocks execution until response or timeout
func (m *Manager) DoGatewayRPC(gatewayID string, request *entities.IotMessage) (*entities.IotMessage, error) {

	// Getting gateway channel from map
	gwChan, ok := m.gwChans.Get(gatewayID).(*GatewayChannel)
	if gwChan == nil || !ok {
		return nil, errors.New("error getting channel with specified gatewayID")
	}

	// Making RPC for gateway channel
	response, err := gwChan.DoRPC(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed making RPC to gateway channel")
	}

	return response, nil
}

// GetGatewayChannel returns gateway channel interface
func (m *Manager) GetGatewayChannel(gatewayID string) io.ReadWriteCloser {
	return m.gwChans.Get(gatewayID)
}

// RestartGateways reads input gateway queues
// and sends restart messages to them
func (m *Manager) RestartGateways() {

	// Create queues request to management plugin
	requestUrl := fmt.Sprintf("http://%s:%d/api/queues/", m.Host, m.CtlPort)
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		logger.Error("failed creating http request",
			"error", err, "caller", "RestartGateways")
		return
	}
	req.SetBasicAuth(m.User, m.Password)

	// Create request context
	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// Get queues list from management plugin
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("error sending http request",
			"error", err, "caller", "RestartGateways")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		queues := make([]amqp.Queue, 0)
		json.NewDecoder(resp.Body).Decode(&queues)

		var gwCounter int
		for _, que := range queues {
			nameArr := strings.Split(que.Name, ".")
			if len(nameArr) > 1 && nameArr[1] == "in" {

				// Create restart message
				gatewayID := nameArr[0]
				message := entities.CreateCloudIotMessage(gatewayID, "")
				message.Protocol = "amqp"
				message.MessageType = "command"
				message.Command = "restart"

				// Marshal message to JSON
				buffer, err := json.Marshal(message)
				if err != nil {
					logger.Error("failed marshalling message to JSON",
						"error", err, "caller", "RestartGateways")
					continue
				}

				// Send JSON to RabbitMQ broker
				wr := NewAmqpWriter(m.Conn, gatewayID)
				if n, err := wr.Write(buffer); err != nil || n != len(buffer) {
					logger.Error("error sending message to broker",
						"error", err, "caller", "RestartGateways")
					continue
				}
				if err := wr.Close(); err != nil {
					logger.Error("error closing gateway input channel",
						"error", err, "caller", "RestartGateways")
				}

				gwCounter++
			}
		}
		if gwCounter > 0 {
			logger.Info("Restarted gateways", "counter", gwCounter)
		}
	}
}
