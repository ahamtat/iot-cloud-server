package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/application"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/rest"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/broker"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/viper"
)

func init() {
	application.Init("../../configs/gocloudserver.dev.yaml")
}

func main() {
	serverID := fmt.Sprintf("iot-cloud-server-%s", viper.GetString("server_id"))

	logger.Info("")
	logger.Info("")
	logger.Info("********************************************************************************")
	logger.Info("Starting application",
		"name", "gocloudserver", "version", entities.VeedoVersion,
		"caller", "main()")

	// Open database connection
	conn := database.NewDatabaseConnection(
		viper.GetString("db.cloud.user"),
		viper.GetString("db.cloud.password"),
		viper.GetString("db.cloud.host"),
		viper.GetString("db.cloud.database"),
		viper.GetInt("db.cloud.port"))
	if err := conn.Init(); err != nil {
		logger.Fatal("error connecting to database", "error", err)
	}

	// Create and initialize broker
	manager := broker.NewManager(
		serverID,
		viper.GetString("amqp.protocol"),
		viper.GetString("amqp.user"),
		viper.GetString("amqp.password"),
		viper.GetString("amqp.host"),
		viper.GetInt("amqp.port"),
		viper.GetInt("amqp.ctlPort"),
	)
	if err := manager.Open(); err != nil {
		logger.Fatal("could not open broker", "error", err)
	}

	// Restart connected gateways to renew their statuses
	manager.RestartGateways()

	// Initialize event exchange to process gateways statuses
	if err := manager.EventExchangeInit(); err != nil {
		logger.Fatal("could not initialize event exchange", "error", err)
	}

	// Make cancel context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Start RabbitMQ events processor
	go manager.ProcessExchangeEvents(ctx, conn)

	// Create RESTful API server
	restAPI := rest.NewServer(manager)
	if restAPI == nil {
		logger.Fatal("could not initialize RESTful API server")
	}

	// Start RESTful API server in a separate goroutine
	go func() {
		if err := restAPI.Start(); err != nil {
			logger.Fatal("could not start RESTful API server", "error", err)
		}
	}()

	// Set interrupt handler
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Application started. Press Ctrl+C to exit...")

	// Wait for interruption events
	select {
	case <-ctx.Done():
		logger.Info("Main context cancelled")
	case <-done:
		logger.Info("User or OS interrupted program")
		cancel()
	}

	// Make RabbitMQ connection shutdown
	if err := manager.Close(); err != nil {
		logger.Error("error while closing connection", "error", err)
	}

	// Stop RESTful server
	if err := restAPI.Stop(); err != nil {
		logger.Error("could not stop RESTful API server", "error", err)
	}

	// Close database connection
	if err := conn.Close(); err != nil {
		logger.Error("failed closing database connection", "error", err)
	}

	logger.Info("Application exited properly")
}
