package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/tasks"

	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/broker"

	"github.com/ahamtat/iot-cloud-server/internal/infrastructure/logger"

	"github.com/pkg/errors"

	"github.com/spf13/viper"

	"github.com/ahamtat/iot-cloud-server/internal/domain/entities"

	"github.com/gin-gonic/gin"
)

func init() {
	// Turn off debug noise
	gin.SetMode(gin.ReleaseMode)
}

// Server structure
type Server struct {
	router *gin.Engine
	srv    *http.Server
	mgr    *broker.Manager
}

type commandData struct {
	Command     string   `json:"command,omitempty"`
	Attribute   string   `json:"attribute,omitempty"`
	GatewayIds  []string `json:"gatewayIds,omitempty"`
	DeviceID    string   `json:"deviceId,omitempty"`
	TariffID    uint64   `json:"tariffId,omitempty"`
	Money       uint64   `json:"money,omitempty"`
	Vip         bool     `json:"vip,omitempty"`
	LegalEntity bool     `json:"isLegalEntity,omitempty"`
	UserID      uint64   `json:"userId,omitempty"`
}

// NewServer constructs and initializes REST server
func NewServer(mgr *broker.Manager) *Server {

	server := &Server{
		router: gin.Default(),
		srv:    nil,
		mgr:    mgr,
	}

	// Group using gin.BasicAuth() middleware
	// gin.Accounts is a shortcut for map[string]string
	authorized := server.router.Group("/api/v3", gin.BasicAuth(gin.Accounts{
		viper.GetString("rest.user"): viper.GetString("rest.password"),
	}))

	// Set routing handlers
	authorized.GET("/info", server.handleInfo)
	authorized.GET("/gateway/configure/:gatewayId", server.handleGatewayConfigure)
	authorized.POST("/command", server.handleCommand)

	return server
}

// Get server info
func (s *Server) handleInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"vendor":      entities.VendorName,
		"version":     entities.VeedoVersion,
		"serviceName": entities.ServiceName,
	})
}

// Get gateway configure
// Test with:
// curl -ki -X GET -H "Content-Type: application/json" -H "Authorization: Basic YmFuZGVyc25hdGNoOnNpM25ZUHpqeU4=" http://127.0.0.1:2020/api/v3/gateway/configure/6774f85a-0a5b-4059-9b68-9385ecbdcf8e
func (s *Server) handleGatewayConfigure(c *gin.Context) {

	gatewayID := c.Param("gatewayId")
	logger.Debug("Getting gateway configure", "gateway", gatewayID)

	// Create RPC request for gateway
	request := entities.CreateCloudIotMessage(gatewayID, "")
	request.DeviceType = "gateway"
	request.Protocol = "amqp"
	request.MessageType = "configurationData"
	request.Command = "get"

	response, err := s.mgr.DoGatewayRPC(gatewayID, request)
	if err != nil {
		errorText := "gateway RPC request failed"
		logger.Error(errorText, "error", err, "gateway", gatewayID)
		c.String(http.StatusBadRequest, errorText)
		return
	}
	if response == nil {
		errorText := "no gateway configuration returned"
		logger.Error(errorText, "gateway", gatewayID)
		c.String(http.StatusBadRequest, errorText)
		return
	}

	c.JSON(http.StatusOK, response)
}

// Send command to external services (WSE, Push Notification, gateway)
// Test power switch:
// curl -ki -X POST -H "Content-Type: application/json" -H "Authorization: Basic YmFuZGVyc25hdGNoOnNpM25ZUHpqeU4=" -d '{"command": "switch", "attribute": "on", "deviceId": "20873eb0-dd5e-4213-a175-b99fbbad3118", "gatewayIds": ["6774f85a-0a5b-4059-9b68-9385ecbdcf8e"]}' http://127.0.0.1:2020/api/v3/command
// Test Wowza recording:
// curl -ki -X POST -H "Content-Type: application/json" -H "Authorization: Basic YmFuZGVyc25hdGNoOnNpM25ZUHpqeU4=" -d '{"command": "setRecording", "attribute": "continuous", "gatewayIds": ["6774f85a-0a5b-4059-9b68-9385ecbdcf8e"], "deviceId": "1616453d-30cd-44b7-9bf0-5b7aac54b488", "tariffId": 103, "money": 3355, "vip": false, "isLegalEntity": false}' http://127.0.0.1:2020/api/v3/command
// Test push notification:
// curl -ki -X POST -H "Content-Type: application/json" -H "Authorization: Basic YmFuZGVyc25hdGNoOnNpM25ZUHpqeU4=" -d '{"command": "push", "attribute": "on", "gatewayIds": ["6774f85a-0a5b-4059-9b68-9385ecbdcf8e"], "userId": 649}, ' http://127.0.0.1:2020/api/v3/command
func (s *Server) handleCommand(c *gin.Context) {

	// Parse command data from JSON body
	var data commandData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Command data", "data", data)

	// Iterate through gateway ids
	var gwFound bool
	for _, gatewayID := range data.GatewayIds {

		gwChan := s.mgr.GetGatewayChannel(gatewayID)
		if gwChan == nil {
			logger.Warn("Channel map returned nil object",
				"gateway", gatewayID, "caller", "handleCommand")
			continue
		}
		gwFound = true

		// Cast gateway channel
		ch, ok := gwChan.(*broker.GatewayChannel)
		if !ok || ch == nil {
			logger.Error("type assert failed", "caller", "handleCommand")
			continue
		}

		// Execute command by its type
		switch data.Command {
		case "push":
			// Set push flag
			bl := ch.GetLogic()
			if bl == nil {
				logger.Error("no business logic loaded", "gateway", gatewayID)
				continue
			}
			bl.SetPush(data.Attribute == "on")

		case "switch":
			// Create gateway message to turn on/off smart plug
			message := entities.CreateCloudIotMessage(gatewayID, data.DeviceID)
			message.DeviceType = "sensor"
			message.Protocol = "zwave"
			message.MessageType = "command"
			message.Command = data.Command
			message.Attribute = data.Attribute

			// Send message to gateway
			go tasks.NewSendGatewayMessageTask(gwChan).Run(message)

		case "setRecording":
			// Create message to logic processor
			message := entities.CreateCloudIotMessage(gatewayID, data.DeviceID)
			message.DeviceType = "camera"
			message.Protocol = "onvif"
			message.MessageType = "command"
			message.Command = data.Command
			message.Attribute = data.Attribute
			message.TariffId = data.TariffID
			message.Money = data.Money
			message.Vip = data.Vip
			message.LegalEntity = data.LegalEntity

			// Processing incoming message in a separate goroutine
			go ch.ApplyLogic(*message)
		}
	}

	// Return error if no gateways found for incoming command
	if !gwFound {
		errorText := "no gateways found"
		logger.Error(errorText, "gateways", data.GatewayIds)
		c.String(http.StatusNotFound, errorText)
		return
	}

	c.String(http.StatusOK, "OK")
}

// Start RESTful server for all interfaces
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", viper.GetInt("rest.port"))
	s.srv = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Wrap(err, "failed starting RESTful API server")
	}

	return nil
}

// Stop RESTful API server gracefully
func (s *Server) Stop() error {

	if s.srv == nil {
		return errors.New("server object is not created")
	}

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "failed shutting down RESTful API server")
	}

	return nil
}
