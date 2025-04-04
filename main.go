package main

import (
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"zs-vm-agent/clients"
	"zs-vm-agent/config-templates/loadbalancer"
	"zs-vm-agent/services"
)

var templateMap map[string]func(logger *logrus.Logger) error = map[string]func(logger *logrus.Logger) error{
	"loadbalancer": loadbalancer.SetupLoadbalancer,
}

func main() {
	logger := initLogging()

	logger.Info("Initializing Clients")
	clients.Initialize(logger)
	logger.Info("Initializing Services")
	services.Initialize(logger)

	tags, _ := clients.GetInfraConfigMapperClient().GetTagsByHostname()

	for _, tag := range tags {
		val, okay := templateMap[tag]
		if okay {
			err := val(logger)
			if err != nil {
				return
			}
			break
		}
	}
}

func initLogging() *logrus.Logger {
	log := logrus.New()
	logLevel := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	logLevelCode := logrus.InfoLevel

	switch logLevel {
	case "DEBUG":
		logLevelCode = logrus.DebugLevel
		break
	case "ERROR":
		logLevelCode = logrus.ErrorLevel
		break
	}

	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true

	log.SetLevel(logLevelCode)
	log.SetFormatter(customFormatter)
	return log
}
