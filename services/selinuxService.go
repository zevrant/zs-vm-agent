package services

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
	"strconv"
)

//TODO: Hook into C++ selinux api directly rather than exec commands

type SeLinuxService interface {
	initialize(logger *logrus.Logger)
	ChangeContext(path string, u string, r string, t string, recursive bool) error
	OpenInboundPort(port int, protocol PortProtocol) error
}

type SeLinuxServiceImpl struct {
	logger *logrus.Logger
}

type PortProtocol = string

const (
	TCP PortProtocol = "TCP"
	UDP PortProtocol = "UDP"
)

func (selinuxService *SeLinuxServiceImpl) initialize(logger *logrus.Logger) {
	selinuxService.logger = logger
}

func (selinuxService *SeLinuxServiceImpl) OpenInboundPort(port int, protocol PortProtocol) error {
	command := exec.Command("/usr/sbin/semanage", "port", "--add", "--type", "http_port_t", fmt.Sprintf("--proto %s", protocol), strconv.FormatInt(int64(port), 10))

	outputText, executeCommandError := command.CombinedOutput()

	selinuxService.logger.Info(outputText)

	if executeCommandError != nil {
		selinuxService.logger.Errorf("Failed to enable inbound port with SEManage: %s", executeCommandError.Error())
		return executeCommandError
	}
	return nil
}

func (selinuxService *SeLinuxServiceImpl) ChangeContext(path string, u string, r string, t string, recursive bool) error {
	var args []string

	if recursive {
		args = append(args, "-R")
	}

	args = append(args, "-u", u, "-r", r, "-t", t, path)

	command := exec.Command("/usr/bin/chcon", args...)

	outputText, executeCommandError := command.CombinedOutput()

	selinuxService.logger.Info(outputText)

	if executeCommandError != nil {
		selinuxService.logger.Errorf("Failed to chcon on %s: %s", path, executeCommandError.Error())
		return executeCommandError
	}
	return nil
}
