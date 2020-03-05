package broker

import (
	"context"
	"encoding/json"
	"io"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
	"github.com/AcroManiac/iot-cloud-server/internal/domain/logic"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"

	"github.com/streadway/amqp"
)

type GatewayChannel struct {
	serverId  string
	gatewayId string
	out       io.ReadCloser
	in        io.Writer
	ctx       context.Context
	cancel    context.CancelFunc
	bl        interfaces.Logic
}

func NewGatewayChannel(ch *amqp.Channel, serverId, gatewayId string, conn *database.Connection) interfaces.Channel {
	// Create cancel context
	ctx, cancel := context.WithCancel(context.Background())

	// Create gateway reader and writer
	out := NewAmqpReader(ctx, ch, gatewayId)
	if out == nil {
		return nil
	}
	in := NewAmqpWriter(ch, gatewayId)
	if in == nil {
		return nil
	}

	// TODO: Create business logic when gateway is registered in database
	// Create business logic and load params
	bl := logic.NewGatewayLogic(ctx, conn, gatewayId)
	if err := bl.LoadParams(in); err != nil {
		logger.Error("cannot load business logic params",
			"error", err,
			"gateway", gatewayId)
	}

	return &GatewayChannel{
		serverId:  serverId,
		gatewayId: gatewayId,
		out:       out,
		in:        in,
		ctx:       ctx,
		cancel:    cancel,
		bl:        bl,
	}
}

func (c *GatewayChannel) Read(p []byte) (n int, err error) {
	return c.out.Read(p)
}

func (c *GatewayChannel) Write(p []byte) (n int, err error) {
	return c.in.Write(p)
}

func (c *GatewayChannel) Close() error {
	c.Stop()
	return c.out.Close()
}

func (c *GatewayChannel) Start() {
	// Read and process messages from gateway
	go func() {
		buffer := make([]byte, 50*1024)
	OUTER:
		for {
			select {
			case <-c.ctx.Done():
				break OUTER
			default:
				if _, err := c.Read(buffer); err != nil {
					logger.Error("Error reading channel", "error", err)
					continue
				}
				message := string(buffer)
				logger.Debug("Message from gateway", "message", message)

				// Start processing incoming message in a separate goroutine
				go func() {
					// Check if business logic is loaded
					if c.bl == nil {
						logger.Error("Gateway logic is not loaded", "gateway", c.gatewayId)
						return
					}
					iotmessage := entities.IotMessage{}
					if err := json.Unmarshal(buffer, iotmessage); err != nil {
						logger.Error("can not unmarshal incoming gateway message",
							"error", err,
							"gateway", c.gatewayId)
					}
					if err := c.bl.Process(&iotmessage); err != nil {
						logger.Error("error processing message",
							"error", err,
							"gateway", c.gatewayId)
					}
				}()
			}
		}
	}()
}

func (c *GatewayChannel) Stop() {
	// Stop goroutines - fire context cancelling
	c.cancel()
}
