package k8s

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"zs-vm-agent/clients"

	"github.com/sirupsen/logrus"
)

func ControllerSetup(logger *logrus.Logger, vmDetails clients.ProxmoxVm) error {
	setupError := Setup(logger, vmDetails)
	if setupError != nil {
		return setupError
	}

	logger.Debug("Setup Successful")

	kubeConfig, loadConfigError := loadConfig(logger)

	if loadConfigError != nil {
		return loadConfigError
	}

	controllerIps := kubeConfig.ControllerIpAddresses

	logger.Debug("Determining my IP Address")

	logger.Debugf("I have %d IP Addresses", len(vmDetails.IpConfig))

	myIp := vmDetails.IpConfig[0].IpAddress

	logger.Debugf("Unedited IP is %s", myIp)

	myIp = vmDetails.IpConfig[0].IpAddress

	logger.Debugf("My IP Address is %s", myIp)

	logger.Debugf("First Controller IP is %s", controllerIps[0])

	if myIp == controllerIps[0] {
		return k8sInit(logger, kubeConfig)
	}

	return k8sControllerJoin(logger, kubeConfig)
}

func k8sInit(logger *logrus.Logger, kubeConfig *k8sConfig) error {
	_, statFileError := os.Stat("/etc/kubernetes/kubelet.conf")
	if !errors.Is(statFileError, os.ErrNotExist) {
		logger.Info("Kubernetes config already exists, skipping...")
		return nil
	}
	var kubeInitArgs = []string{
		"init",
		fmt.Sprintf("--control-plane-endpoint=%s", kubeConfig.ControlPlaneEndpoint),
		"--upload-certs",
		fmt.Sprintf("--pod-network-cidr=%s", kubeConfig.PodNetworkCidr),
		fmt.Sprintf("--service-cidr=%s", kubeConfig.ServiceNetworkCidr),
		"--token",
		kubeConfig.K8sInitToken,
	}
	logger.Debugf("/usr/bin/kubeadm %s", strings.Replace(strings.Join(kubeInitArgs, " "), kubeConfig.K8sInitToken, "XXXXXXX", 1))
	command := exec.Command("/usr/bin/kubeadm", kubeInitArgs...)

	outputText, commandExecutionError := command.CombinedOutput()

	for _, line := range strings.Split(string(outputText), "\n") {
		logger.Info(line)
	}

	if commandExecutionError != nil {
		logger.Errorf("Failed to initialize kubernetes cluster controller: %s", commandExecutionError.Error())
		return commandExecutionError
	}
	return nil

}

func k8sControllerJoin(logger *logrus.Logger, kubeConfig *k8sConfig) error {
	_, statFileError := os.Stat("/etc/kubernetes/kubelet.conf")
	if !errors.Is(statFileError, os.ErrNotExist) {
		logger.Info("Kubernetes config already exists, skipping...")
		return nil
	}
	var kubeInitArgs = []string{
		"join",
		fmt.Sprintf("%s:6443", kubeConfig.ControlPlaneEndpoint),
		"--discovery-token-unsafe-skip-ca-verification",
		"--token",
		kubeConfig.K8sInitToken,
	}
	logger.Debugf("/usr/bin/kubeadm %s", strings.Replace(strings.Join(kubeInitArgs, " "), kubeConfig.K8sInitToken, "XXXXXXX", 1))
	command := exec.Command("/usr/bin/kubeadm", kubeInitArgs...)

	outputText, commandExecutionError := command.CombinedOutput()

	for _, line := range strings.Split(string(outputText), "\n") {
		logger.Info(line)
	}

	if commandExecutionError != nil {
		logger.Errorf("Failed to join kubernetes cluster controller: %s", commandExecutionError.Error())
		return commandExecutionError
	}
	return nil

}
