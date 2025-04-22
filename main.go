package main

import (
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"zs-vm-agent/clients"
	"zs-vm-agent/config-templates/loadbalancer"
	"zs-vm-agent/services"
)

var templateMap = map[string]func(logger *logrus.Logger, vmDetails clients.ProxmoxVm) error{
	"loadbalancer": loadbalancer.SetupLoadBalancer,
}

func main() {
	logger := initLogging()

	logger.Info("Initializing Clients")
	clients.Initialize(logger)
	logger.Info("Initializing Services")
	services.Initialize(logger)

	vmDetails, getVmDetailsError := clients.GetInfraConfigMapperClient().GetVmDetailsByHostname()

	if getVmDetailsError != nil {
		logger.Errorf("Failed to retrieve vm details: %s", getVmDetailsError.Error())
		return
	}

	if vmDetails == nil {
		logger.Error("Failed to retrieve vm details, retrieved nil")
		return
	}
	logger.Debugf("Retrieved vm details for vm %s", vmDetails.VmId)

	for _, tag := range vmDetails.Tags {
		val, okay := templateMap[tag]
		if okay {
			err := val(logger, *vmDetails)
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
