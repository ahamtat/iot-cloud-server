package application

import (
	"flag"
	"log"

	"github.com/AcroManiac/iot-cloud-server/internal/infrastructure/logger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Init application with configuration file
func Init(defaultConfigPath string) {
	// using standard library "flag" package
	flag.String("config", defaultConfigPath, "path to configuration flag")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)

	// Reading configuration from file
	configFile := viper.GetString("config") // retrieve value from viper
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Couldn't read configuration file: %s", err.Error())
	}

	// Setting log parameters
	logger.Init(
		viper.GetString("log.log_level"),
		viper.GetString("log.log_file"),
		viper.GetBool("log.log_rotate"))
}
