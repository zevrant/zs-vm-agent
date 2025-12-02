package k8s

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"zs-vm-agent/clients"
	"zs-vm-agent/services"

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

	certLoadError := loadCertificates(logger, kubeConfig)

	if certLoadError != nil {
		return certLoadError
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

func loadCertificates(logger *logrus.Logger, kubeConfig *k8sConfig) error {
	filesystemService := services.GetFileSystemService()
	_, statFileError := os.Stat("/etc/kubernetes/pki/ca.crt")
	if statFileError != nil && !strings.Contains(statFileError.Error(), "no such file or directory") {
		logger.Errorf("Failed to stat k8s ca cert: %s", statFileError.Error())
		return statFileError
	}
	if statFileError == nil {
		logger.Info("Kubernetes certs already exist, skipping...")
		return nil
	}
	createDirectoryError := filesystemService.CreateRootFsDirectory("/etc/kubernetes/pki", false, 0751)
	if createDirectoryError != nil {
		return createDirectoryError
	}

	certWrtiteError := filesystemService.WriteFileContents("/etc/kubernetes/pki/ca.crt", []byte(kubeConfig.K8sCaInitPublicCert), 0644)

	if certWrtiteError != nil {
		return certWrtiteError
	}
	privateKeyWriteError := filesystemService.WriteFileContents("/etc/kubernetes/pki/ca.key", []byte(kubeConfig.K8sCaInitPrivateKey), 0651)

	if privateKeyWriteError != nil {
		return privateKeyWriteError
	}

	permissionChangeError := filesystemService.SetRootFsPermissions("/etc/kubernetes/pki/ca.key", 0600, false)

	if permissionChangeError != nil {
		return permissionChangeError
	}
	return nil
}

func k8sInit(logger *logrus.Logger, kubeConfig *k8sConfig) error {
	_, statFileError := os.Stat("/etc/kubernetes/kubelet.conf")
	if !errors.Is(statFileError, os.ErrNotExist) {
		logger.Info("Kubernetes config already exists, skipping...")
		return nil
	}
	certLoadError := loadCertificates(logger, kubeConfig)

	if certLoadError != nil {
		return certLoadError
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
	logger.Info("Initializing Kubernetes Cluster, this may take awhile...")
	logger.Debugf("/usr/bin/kubeadm %s", strings.Replace(strings.Join(kubeInitArgs, " "), kubeConfig.K8sInitToken, "XXXXXXX", 1))
	command := exec.Command("/usr/bin/kubeadm", kubeInitArgs...)

	outputText, commandExecutionError := command.CombinedOutput()

	logger.Info(string(outputText))

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

	hash, hashError := generateK8sCaHash(logger)

	if hashError != nil {
		return hashError
	}

	logger.Debugf("Retrieved ca hash is %s", *hash)

	var kubeInitArgs = []string{
		"join",
		"--control-plane",
		fmt.Sprintf("%s:6443", kubeConfig.ControlPlaneEndpoint),
		"--discovery-token-ca-cert-hash",
		fmt.Sprintf("sha256:%s", *hash),
		"--token",
		kubeConfig.K8sInitToken,
	}
	logger.Debugf("/usr/bin/kubeadm %s", strings.Replace(strings.Join(kubeInitArgs, " "), kubeConfig.K8sInitToken, "XXXXXXX", 1))
	command := exec.Command("/usr/bin/kubeadm", kubeInitArgs...)

	outputText, commandExecutionError := command.CombinedOutput()

	logger.Info(string(outputText))

	if commandExecutionError != nil {
		logger.Errorf("Failed to join kubernetes cluster controller: %s", commandExecutionError.Error())
		return commandExecutionError
	}
	return nil

}
