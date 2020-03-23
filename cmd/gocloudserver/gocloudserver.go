package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/entities"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/rest"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/database"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/broker"
	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	// using standard library "flag" package
	flag.String("config", "../../configs/gocloudserver.dev.yaml", "path to configuration flag")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)

	// Reading configuration from file
	configPath := viper.GetString("config") // retrieve value from viper
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Couldn't read configuration file: %s", err.Error())
	}

	// Setting log parameters
	logger.Init(viper.GetString("log.log_level"), viper.GetString("log.log_file"))
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
	)
	if err := manager.Open(); err != nil {
		logger.Fatal("could not open broker", "error", err)
	}

	if err := manager.EventExchangeInit(); err != nil {
		logger.Fatal("could not initialize event exchange", "error", err)
	}

	// Create RESTful API server
	restAPI := rest.NewServer()
	if err := restAPI.Init(); err != nil {
		logger.Fatal("could not initialize RESTful API server", "error", err)
	}

	// Make cancel context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Set interrupt handler
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start RabbitMQ events processor
	go manager.ProcessExchangeEvents(ctx, conn)

	// Start RESTful API server in a separate goroutine
	go func() {
		if err := restAPI.Start(); err != nil {
			logger.Fatal("could not start RESTful API server", "error", err)
		}
	}()

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
