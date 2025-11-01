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

func WorkerSetup(logger *logrus.Logger, vmDetails clients.ProxmoxVm) error {
	setupError := Setup(logger, vmDetails)
	if setupError != nil {
		return setupError
	}

	logger.Debug("Setup Successful")

	kubeConfig, loadConfigError := loadConfig(logger)

	if loadConfigError != nil {
		return loadConfigError
	}

	certLoadError := loadCertificates(logger, kubeConfig)

	if certLoadError != nil {
		return certLoadError
	}

	return k8sWorkerJoin(logger, kubeConfig)
}

func k8sWorkerJoin(logger *logrus.Logger, kubeConfig *k8sConfig) error {
	_, statFileError := os.Stat("/etc/kubernetes/kubelet.conf")
	if !errors.Is(statFileError, os.ErrNotExist) {
		logger.Info("Kubernetes config already exists, skipping...")
		return nil
	}

	hash, hashError := generateK8sCaHash(logger)

	if hashError != nil {
		return hashError
	}

	// if these fail the files likely don't exist or other errors that will cause subsequent failures
	_ = os.Remove("/etc/kubernetes/pki/ca.crt")
	_ = os.Remove("/etc/kubernetes/pki/ca.key")

	logger.Debugf("Retrieved ca hash is %s", *hash)

	var kubeInitArgs = []string{
		"join",
		fmt.Sprintf("%s:6443", kubeConfig.ControlPlaneEndpoint),
		"--discovery-token-ca-cert-hash",
		fmt.Sprintf("sha256:%s", *hash),
		"--token",
		kubeConfig.K8sInitToken,
	}
	logger.Debugf("/usr/bin/kubeadm %s", strings.Replace(strings.Join(kubeInitArgs, " "), kubeConfig.K8sInitToken, "XXXXXXX", 1))
	command := exec.Command("/usr/bin/kubeadm", kubeInitArgs...)

	outputText, commandExecutionError := command.CombinedOutput()

	logger.Info(outputText)

	if commandExecutionError != nil {
		logger.Errorf("Failed to join kubernetes cluster controller: %s", commandExecutionError.Error())
		return commandExecutionError
	}
	return nil

}
