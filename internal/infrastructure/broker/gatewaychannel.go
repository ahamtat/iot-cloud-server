package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic/messages"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic/tasks"
	"github.com/pkg/errors"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"

	"github.com/streadway/amqp"
)

// GatewayChannel structure keeps data for
// gateway channel i/o and message processing
type GatewayChannel struct {
	serverID   string
	gatewayID  string
	conn       *database.Connection
	out        *AmqpReader
	in         *AmqpWriter
	ctx        context.Context
	cancel     context.CancelFunc
	rpcMx      sync.Mutex
	rpcCalls   rpcPendingCallMap
	rpcTimeout time.Duration
	bl         interfaces.Logic
}

type rpcPendingCall struct {
	done chan bool
	data *entities.IotMessage
}

type rpcPendingCallMap map[string]*rpcPendingCall

// NewGatewayChannel function for GatewayChannel structure construction
func NewGatewayChannel(amqpConn *amqp.Connection, dbConn *database.Connection, serverID, gatewayID string) interfaces.Channel {
	// Create cancel context
	ctx, cancel := context.WithCancel(context.Background())

	// Create gateway reader and writer
	out := NewAmqpReader(ctx, amqpConn, gatewayID)
	if out == nil {
		return nil
	}
	in := NewAmqpWriter(amqpConn, gatewayID)
	if in == nil {
		return nil
	}

	return &GatewayChannel{
		serverID:   serverID,
		gatewayID:  gatewayID,
		conn:       dbConn,
		out:        out,
		in:         in,
		ctx:        ctx,
		cancel:     cancel,
		rpcMx:      sync.Mutex{},
		rpcCalls:   make(rpcPendingCallMap),
		rpcTimeout: 5 * time.Second,
		bl:         nil, // Do not create business logic until gateway status come
	}
}

func (c *GatewayChannel) Read(p []byte) (n int, err error) {
	return c.out.Read(p)
}

func (c *GatewayChannel) Write(p []byte) (n int, err error) {
	return c.in.Write(p)
}

// Close reading and writing channels
func (c *GatewayChannel) Close() error {
	c.Stop()

	// Close pending calls to quit blocked goroutines
	c.rpcMx.Lock()
	for _, call := range c.rpcCalls {
		close(call.done)
	}
	c.rpcMx.Unlock()

	// Close i/o channels
	if err := c.out.Close(); err != nil {
		logger.Error("error closing gateway output channel",
			"error", err, "caller", "GatewayChannel")
	}
	if err := c.in.Close(); err != nil {
		logger.Error("error closing gateway input channel",
			"error", err, "caller", "GatewayChannel")
	}

	return nil
}

// Start functions make separate goroutine for message receiving and processing
func (c *GatewayChannel) Start() {
	// Read and process messages from gateway
	go func() {
	OUTER:
		for {
			select {
			case <-c.ctx.Done():
				break OUTER
			default:
				// Read input message
				inputEnvelope, toBeClosed, err := c.out.ReadEnvelope()
				if err != nil {
					logger.Error("error reading channel", "error", err)
					continue
				}
				if toBeClosed {
					// Reading channel possibly is to be closed
					continue
				}

				// Check for RPC responses
				if len(inputEnvelope.Metadata.CorrelationID) > 0 {
					// Make pending call
					c.rpcMx.Lock()
					rpcCall, ok := c.rpcCalls[inputEnvelope.Metadata.CorrelationID]
					c.rpcMx.Unlock()
					if ok {
						rpcCall.data = inputEnvelope.Message
						rpcCall.done <- true
					}
					continue
				}

				// Start processing incoming message in a separate goroutine
				go c.ApplyLogic(*inputEnvelope.Message)
			}
		}
	}()
}

// ApplyLogic checks gateway, loads business logic and starts processing
func (c *GatewayChannel) ApplyLogic(message entities.IotMessage) {

	iotmessage := &message
	// Load business logic if gateway is online and registered in database
	if c.bl == nil {
		exists, err := c.CheckGatewayExistence(iotmessage)
		if err != nil {
			logger.Error("error checking gateway in database",
				"error", err,
				"gateway", c.gatewayID,
				"caller", "GatewayChannel")
			return
		}
		if !exists {
			logger.Warn("Gateway is not registered in cloud database",
				"gateway", c.gatewayID,
				"caller", "GatewayChannel")
			return
		}

		// Create business logic
		c.bl, err = c.CreateLogic()
		if err != nil {
			logger.Error("cannot load business logic params",
				"error", err,
				"gateway", c.gatewayID,
				"caller", "GatewayChannel")
		}
	}

	// Process incoming message
	if err := c.bl.Process(iotmessage); err != nil {
		logger.Error("error processing message",
			"error", err,
			"gateway", c.gatewayID,
			"caller", "GatewayChannel")
	}
}

// CreateLogic function creates business logic and loads params
func (c *GatewayChannel) CreateLogic() (interfaces.Logic, error) {
	bl := logic.NewGatewayLogic(c.ctx, c.conn, c.gatewayID)
	if err := bl.LoadParams(c.in); err != nil {
		return nil, err
	}
	return bl, nil
}

// CheckGatewayExistence checks for gateway records in database and update devices statuses
func (c *GatewayChannel) CheckGatewayExistence(message *entities.IotMessage) (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Search gateway in database
	queryText := `select count(*) from v3_gateways where gateway_id = ?`
	var value int
	err := c.conn.Db.QueryRowContext(ctx, queryText, message.GatewayId).Scan(&value)
	if err != nil {
		return false, errors.Wrap(err, "failed searching gateway in database")
	}
	if value == 0 {
		// No gateway found
		return false, nil
	}

	// Found gateway in database
	return true, nil
}

// Stop message processing and writing off status to database
func (c *GatewayChannel) Stop() {
	// Stop goroutines - fire context cancelling
	c.cancel()

	// Change gateway and all its devices statuses to offline in database
	statusMessage := messages.NewStatusMessage(c.gatewayID, "off")
	tasks.NewUpdateGatewayStatusTask(c.conn).Run(statusMessage)
}

// DoRPC sends command for gateway via RabbitMQ broker and
// blocks execution until response or timeout
func (c *GatewayChannel) DoRPC(request *entities.IotMessage) (response *entities.IotMessage, err error) {

	// Create message envelope
	correlationID := CreateCorrelationID()
	env := &AmqpEnvelope{
		Message: request,
		Metadata: &AmqpMetadata{
			CorrelationID: correlationID,
			ReplyTo:       fmt.Sprintf("%s.out", c.gatewayID),
		},
	}

	// Write envelope to broker
	err = c.in.WriteEnvelope(env)
	if err != nil {
		return nil, errors.Wrap(err, "error writing RPC buffer to broker")
	}

	// Create and keep pending object
	rpcCall := &rpcPendingCall{done: make(chan bool)}
	c.rpcMx.Lock()
	c.rpcCalls[correlationID] = rpcCall
	c.rpcMx.Unlock()

	// Wait until response comes or timeout
	select {
	case <-rpcCall.done:
		response = rpcCall.data
	case <-time.After(c.rpcTimeout):
		err = errors.New("timeout elapsed on RPC request sending")
	}

	// Release pending object
	c.rpcMx.Lock()
	delete(c.rpcCalls, correlationID)
	c.rpcMx.Unlock()

	// Return response to caller
	return nil, nil
}

// GetLogic returns business logic for gateway channel
func (c *GatewayChannel) GetLogic() interfaces.Logic {
	return c.bl
}
