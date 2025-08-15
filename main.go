package main

import (
	"bytes"
	"os"
	"strings"
	"time"
	"zs-vm-agent/clients"
	"zs-vm-agent/config-templates/dns"
	"zs-vm-agent/config-templates/loadbalancer"
	"zs-vm-agent/config-templates/vault"
	"zs-vm-agent/services"

	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/sirupsen/logrus"
)

var templateMap = map[string]func(logger *logrus.Logger, vmDetails clients.ProxmoxVm) error{
	"loadbalancer": loadbalancer.SetupLoadBalancer,
	"dns":          dns.SetupBind9,
	"vault":        vault.Setup,
}

func main() {
	logger := initLogging()

	hostname, getHostnameError := loadHostname(logger)

	if getHostnameError != nil {
		logger.Errorf("Failed to read hostname from filesystem: %s", getHostnameError.Error())
		os.Exit(-1)
	}

	logger.Info("Initializing Clients")
	clients.Initialize(logger, *hostname)
	logger.Info("Initializing Services")
	services.Initialize(logger)

	vmDetails, getVmDetailsError := clients.GetInfraConfigMapperClient().GetVmDetailsByHostname()

	if getVmDetailsError != nil {
		logger.Errorf("Failed to retrieve vm details: %s", getVmDetailsError.Error())
		os.Exit(-1)
	}

	if vmDetails == nil {
		logger.Error("Failed to retrieve vm details, retrieved nil")
		os.Exit(-1)
	}
	logger.Debugf("Retrieved vm details for vm %s", vmDetails.VmId)

	for _, tag := range vmDetails.Tags {
		logger.Debugf("Parsing tag %s", tag)
		val, okay := templateMap[tag]
		if okay {
			err := val(logger, *vmDetails)
			if err != nil {
				os.Exit(-1)
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

func retrieveHostname() (*string, error) {
	hostnameFile, openFileError := file.OpenFromPath("/etc/hostname", true)
	if openFileError != nil {
		return nil, openFileError
	}

	readBuffer := make([]byte, 4096)
	bytesRead, readError := hostnameFile.Read(readBuffer)
	byteBuffer := bytes.Buffer{}
	if readError != nil {
		return nil, readError
	}
	for bytesRead > 0 {
		if readError != nil {
			return nil, readError
		}
		for i := range bytesRead {
			byteBuffer.WriteByte(readBuffer[i])
		}
		bytesRead, readError = hostnameFile.Read(readBuffer)
	}
	hostnameBytes := strings.TrimSpace(string(byteBuffer.Bytes()))
	return &hostnameBytes, nil
}

func loadHostname(logger *logrus.Logger) (*string, error) {
	var hostname *string
	var getHostnameError error

	for hostname == nil || *hostname == "localhost" {
		time.Sleep(1 * time.Second)
		hostname, getHostnameError = retrieveHostname()

		if getHostnameError != nil {
			logger.Errorf("Failed to read hostname from filesystem: %s", getHostnameError.Error())
			os.Exit(-1)
		}
	}

	return hostname, nil
}
