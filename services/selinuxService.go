package services

import (
	"github.com/sirupsen/logrus"
	"os/exec"
	"strconv"
)

//TODO: Hook into C++ selinux api directly rather than exec commands

type SeLinuxService interface {
	initialize(logger *logrus.Logger)
	ChangeContext(path string, u string, r string, t string, recursive bool) error
	OpenInboundPort(port int, protocol PortProtocol) error
	AllowAllOutboundConnection() error
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
	args := []string{"port", strconv.FormatInt(int64(port), 10), "--add", "--type", "http_port_t", "--proto", protocol}
	selinuxService.logger.Debugf("port command is %s %s", "/usr/sbin/semanage", args)
	command := exec.Command("/usr/sbin/semanage", args...)

	selinuxService.logger.Debugf("Opening port %d/%s", port, protocol)

	outputText, executeCommandError := command.CombinedOutput()

	selinuxService.logger.Infof("command output: %s", outputText)

	if executeCommandError != nil {
		selinuxService.logger.Errorf("Failed to enable inbound port with SEManage: %s", executeCommandError.Error())
		return executeCommandError
	}
	return nil
}

func (selinuxService *SeLinuxServiceImpl) AllowAllOutboundConnection() error {
	command := exec.Command("/sbin/setsebool", "-P", "haproxy_connect_any", "1")
	outputText, executeCommandError := command.CombinedOutput()

	selinuxService.logger.Infof("command output: %s", outputText)

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
