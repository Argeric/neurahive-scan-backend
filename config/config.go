package config

import (
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
)

// Read system enviroment variables prefixed with "NEURAHIVE".
// eg., `NEURAHIVE_LOG_LEVEL` will override "log.level" config item from the config file.
const viperEnvPrefix = "neurahive"

func init() {
	// init viper
	viper.MustInit(viperEnvPrefix)
	// init logger
	initLogger()
}

func initLogger() {
	// unmarshal config
	var config struct {
		Level      string `default:"info"`
		ForceColor bool
	}
	viper.MustUnmarshalKey("log", &config)

	// set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		logrus.WithError(err).Fatalf("invalid log level configured: %v", config.Level)
	}
	logrus.SetLevel(level)

	// set force color
	if config.ForceColor {
		logrus.SetFormatter(&logrus.TextFormatter{
			ForceColors:   true,
			FullTimestamp: true,
		})
	}
}
