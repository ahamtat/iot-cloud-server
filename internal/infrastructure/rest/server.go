package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"

	"github.com/pkg/errors"

	"github.com/spf13/viper"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"

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
}

// NewServer constructs REST server
func NewServer() *Server {
	return &Server{}
}

// Init router and middleware
func (s *Server) Init() error {
	s.router = gin.Default()

	// Group using gin.BasicAuth() middleware
	// gin.Accounts is a shortcut for map[string]string
	authorized := s.router.Group("/api/v3", gin.BasicAuth(gin.Accounts{
		viper.GetString("rest.user"): viper.GetString("rest.password"),
	}))

	// Get server info
	authorized.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"vendor":      entities.VendorName,
			"version":     entities.VeedoVersion,
			"serviceName": entities.ServiceName,
		})
	})

	// Get gateway configure
	authorized.GET("/gateway/configure/:gatewayId", func(c *gin.Context) {
		gatewayID := c.Param("gatewayId")
		logger.Debug("Getting gateway configure", "gateway", gatewayID)
		//c.JSON(http.StatusOK, configJSON)
	})

	// Send command to gateway
	authorized.POST("/command", func(c *gin.Context) {
		//
	})

	return nil
}

// Start RESTful server at all interfaces
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
