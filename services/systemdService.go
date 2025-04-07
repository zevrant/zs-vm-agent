package services

import (
	"github.com/sirupsen/logrus"
	"os/exec"
	"strings"
)

type SystemdService interface {
	initialize(logger *logrus.Logger)
	StartService(serviceName string) error
	GetServiceStatus(serviceName string) (int, error)
}

type SystemdServiceImpl struct {
	logger *logrus.Logger
}

func (systemdService *SystemdServiceImpl) initialize(logger *logrus.Logger) {
	systemdService.logger = logger
}

func (systemdService *SystemdServiceImpl) StartService(serviceName string) error {
	command := exec.Command("/usr/bin/systemctl", "start", serviceName)

	outputText, commandExecutionError := command.CombinedOutput()

	systemdService.logger.Info(string(outputText))

	if commandExecutionError != nil {
		systemdService.logger.Errorf("Failed to start systemd service %s: %s", serviceName, commandExecutionError.Error())
		_ = systemdService.getServiceLogs(serviceName)

		return commandExecutionError
	}
	return nil
}

func (systemdService *SystemdServiceImpl) getServiceLogs(serviceName string) error {
	command := exec.Command("/usr/bin/journalctl", "-u", serviceName, "-n", "25")

	outputText, commandExecutionError := command.CombinedOutput()

	for _, line := range strings.Split(string(outputText), "\n") {
		systemdService.logger.Info(line)
	}

	if commandExecutionError != nil {
		systemdService.logger.Errorf("Failed to journal logs from service %s: %s", serviceName, commandExecutionError.Error())
		return commandExecutionError
	}
	return nil

}

func (systemdService *SystemdServiceImpl) GetServiceStatus(serviceName string) (int, error) { //-1: fail, 0: stil starting, 1: successfully started
	command := exec.Command("/usr/bin/systemctl", "is-active", serviceName)

	outputText, commandExecutionError := command.CombinedOutput()

	systemdService.logger.Info(string(outputText))

	if commandExecutionError != nil {
		systemdService.logger.Errorf("Failed to journal logs from service %s: %s", serviceName, commandExecutionError.Error())
		return -1, commandExecutionError
	}

	var status int
	statusText := strings.TrimSpace(string(outputText))
	if string(statusText) == "activating" {
		status = 0
	} else if string(statusText) == "active" {
		status = 1
	} else {
		status = -1
	}

	return status, nil
}
