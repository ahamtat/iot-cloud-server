package rest

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/spf13/viper"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"

	"github.com/gin-gonic/gin"
)

// Server structure
type Server struct {
	router *gin.Engine
}

// NewServer constructs REST server
func NewServer() *Server {
	return &Server{}
}

// Init router and middleware
func (s *Server) Init() {
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
	authorized.GET("/gateway/configure", func(c *gin.Context) {
		//
	})
}

// Start RESTful server at all interfaces
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", viper.GetInt("rest.port"))
	if err := s.router.Run(addr); err != nil {
		return errors.Wrap(err, "failed running RESTful server")
	}
	return nil
}
