package logging

import (
	"github.com/astronomerio/event-router/config"
	"github.com/sirupsen/logrus"
)

// Singleton logger for application
var log *logrus.Logger

// Configure logger on startup
func init() {
	log = logrus.New()

	if config.AppConfig.LogFormat == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	if config.AppConfig.DebugMode {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}
}

// GetLogger returns the singleton logger
func GetLogger(fields logrus.Fields) *logrus.Entry {
	return log.WithFields(fields)
}
