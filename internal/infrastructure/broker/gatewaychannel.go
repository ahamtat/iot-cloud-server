package broker

import (
	"context"
	"encoding/json"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"
	"io"

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

	// Create business logic and load params
	bl := logic.NewGatewayLogic(ctx, conn, gatewayId)
	if err := bl.LoadParams(); err != nil {
		logger.Error("cannot load business logic params",
			"error", err,
			"gateway", gatewayId)
	}

	// Create and initialize gateway i/o channel
	c := &GatewayChannel{
		serverId:  serverId,
		gatewayId: gatewayId,
		ctx:       ctx,
		cancel:    cancel,
		bl:        bl,
	}
	if c.out = NewAmqpReader(ctx, ch, gatewayId); c.out == nil {
		return nil
	}
	if c.in = NewAmqpWriter(ch, gatewayId); c.in == nil {
		return nil
	}

	return c
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
					iotmessage := entities.IotMessage{}
					if err := json.Unmarshal(buffer, iotmessage); err != nil {
						logger.Error("Can not unmarshal incoming gateway message", "error", err)
					}
					if err := c.bl.Process(iotmessage); err != nil {
						logger.Error("Error processing message from gateway", "error", err)
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
