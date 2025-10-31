package k8s

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os/exec"
	"strings"
	"time"
	"zs-vm-agent/clients"
	"zs-vm-agent/services"

	"github.com/sirupsen/logrus"
)

type k8sConfig struct {
	ControlPlaneEndpoint  string   `json:"controlPlaneEndpoint"`
	ControllerIpAddresses []string `json:"controllerIpAddresses"`
	K8sInitToken          string   `json:"k8sInitToken"`
	K8sCaInitPrivateKey   string   `json:"k8sCaInitPrivateKey"`
	K8sCaInitPublicCert   string   `json:"k8sCaInitPublicCert"`
	PodNetworkCidr        string   `json:"podNetworkCidr"`
	ServiceNetworkCidr    string   `json:"serviceNetworkCidr"`
	WorkerIpAddresses     []string `json:"workerIpAddresses"`
}

var driveMappings = map[string]string{
	"/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1": "/etc/kubernetes/",
	"/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi2": "/var/lib/kubelet/",
	"/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi3": "/var/lib/etcd/",
}

var requiredServices = []string{
	"kubelet",
	"containerd",
}

func Setup(logger *logrus.Logger, vmDetails clients.ProxmoxVm) error {
	//mount k8s config drives
	mountDrivesError := mountDrives(logger)

	if mountDrivesError != nil {
		return mountDrivesError
	}

	systemdService := services.GetSystemdService()

	// turn on kubelet systemd service
	// turn on containerd systemd service
	for _, service := range requiredServices {
		startServiceError := systemdService.StartService(service)
		if startServiceError != nil {
			return startServiceError
		}
	}

	return nil
}

func mountDrives(logger *logrus.Logger) error {
	filesystemService := services.GetFileSystemService()
	diskService := services.GetDiskService()
	for diskPath := range maps.Keys(driveMappings) {
		logger.Debugf("Creating Directory %s", driveMappings[diskPath])
		createDirectoryError := filesystemService.CreateRootFsDirectory(driveMappings[diskPath], false, 0640)

		if createDirectoryError != nil {
			return createDirectoryError
		}

		dataDrive, getDiskError := diskService.GetDisk(diskPath)
		if getDiskError != nil && strings.Contains(getDiskError.Error(), "device or resource busy") {
			logger.Infof("Disk %s is busy, skipping...", diskPath)
			continue

		} else if getDiskError != nil {
			return getDiskError
		}

		partTable, getPartTableError := dataDrive.GetPartitionTable()

		if (getPartTableError != nil && strings.Contains(getPartTableError.Error(), "unknown disk partition type")) || len(partTable.GetPartitions()) == 0 {
			logger.Debugf("No Partitions found for %s, creating...", diskPath)
			createDiskPartitionError := diskService.CreatePartition(diskPath)

			if createDiskPartitionError != nil {
				logger.Errorf("Failed to create partition for disk %s: %s", diskPath, createDiskPartitionError.Error())
				return createDiskPartitionError
			}
			time.Sleep(5 * time.Second)
			logger.Debugf("Getting updated partition table for disk %s", diskPath)
			partTable, getPartTableError = dataDrive.GetPartitionTable()
			if getPartTableError != nil {
				logger.Errorf("Failed to get Partition Table from disk after creation %s: %s", diskPath, getPartTableError.Error())
				return getPartTableError
			}
			if len(partTable.GetPartitions()) == 0 {
				logger.Error("No partitions found after creating a new partition")
				return errors.New("no partitions found after creating new partition")
			}
		} else if getPartTableError != nil {
			logger.Errorf("Failed to get Partition Table from disk %s: %s", diskPath, getPartTableError.Error())
			return getPartTableError
		}

		closeDiskError := dataDrive.Close()

		if closeDiskError != nil {
			logger.Errorf("Failed to close disk %s: %s", diskPath, closeDiskError.Error())
			return closeDiskError
		}

		createFilesystemError := filesystemService.CreateXfsFileSystem(fmt.Sprintf("%s-part1", diskPath))

		if createFilesystemError != nil {
			errorMessage := createFilesystemError.Error()
			if !strings.Contains(errorMessage, "appears to contain an existing filesystem") {
				logger.Debug("Actual Error create filesystem, returning error")
				return createFilesystemError
			}
		}

		logger.Debugf("Mounting filesystemd for partition on %s-part1 to %s", diskPath, driveMappings[diskPath])

		mountError := filesystemService.MountFilesystem(fmt.Sprintf("%s-part1", diskPath), driveMappings[diskPath])

		if mountError != nil {
			return mountError
		}

		//setFolderOwnerError := filesystemService.SetRootFsOwner("/opt/vault", "vault", true)
		//if setFolderOwnerError != nil {
		//	return setFolderOwnerError
		//}
	}

	return nil
}

func loadConfig(logger *logrus.Logger) (*k8sConfig, error) {
	filesystemService := services.GetFileSystemService()

	logger.Debug("Loading config filesystem")
	////Get filesystem containing k8s config
	configDrive, getFileSystemError := filesystemService.GetBlockFilesystem("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi4")

	if getFileSystemError != nil {
		return nil, getFileSystemError
	}

	logger.Debug("Reading config file")
	//Read K8s config from filesystem
	k8sConfigBytes, readConfigFileError := filesystemService.ReadFileContentsFromFilesystem(configDrive, "k8s-config.json")

	if readConfigFileError != nil {
		return nil, readConfigFileError
	}

	var parsedConfig k8sConfig

	logger.Debug("Reading Json")
	jsonProcessingError := json.Unmarshal(k8sConfigBytes, &parsedConfig)

	if jsonProcessingError != nil {
		return nil, jsonProcessingError
	}

	//return config object

	logger.Debug("Config load complete!")
	return &parsedConfig, nil
}

func generateK8sCaHash(logger *logrus.Logger) (*string, error) {
	keyLoadError := loadPublicKey(logger)

	if keyLoadError != nil {
		return nil, keyLoadError
	}

	derData, pemConvertError := convertPenToDer(logger)

	if pemConvertError != nil {
		return nil, pemConvertError
	}

	sum := strings.ReplaceAll(fmt.Sprintf("%x", sha256.Sum256(derData)), "\"", "")
	return &sum, nil
}

func loadPublicKey(logger *logrus.Logger) error {
	var hashArgs = []string{
		"x509",
		"-in",
		"/etc/kubernetes/pki/ca.crt",
		"-pubkey",
		"-nocert",
		"-out",
		"/tmp/ca-pub.key",
	}

	logger.Debugf("/bin/openssl %s", strings.Join(hashArgs, " "))
	command := exec.Command("/bin/openssl", hashArgs...)

	outputText, commandExecutionError := command.CombinedOutput()

	for _, line := range strings.Split(string(outputText), "\n") {
		logger.Info(line)
	}

	if commandExecutionError != nil {
		logger.Errorf("Failed to load ca cert: %s", commandExecutionError.Error())
		return commandExecutionError
	}
	return nil
}

func convertPenToDer(logger *logrus.Logger) ([]byte, error) {
	var hashArgs = []string{
		"pkey",
		"-pubin",
		"-in",
		"/tmp/ca-pub.key",
		"-outform",
		"DER",
	}

	logger.Debugf("/bin/openssl %s", strings.Join(hashArgs, " "))
	command := exec.Command("/bin/openssl", hashArgs...)

	outputText, commandExecutionError := command.CombinedOutput()

	for _, line := range strings.Split(string(outputText), "\n") {
		logger.Info(line)
	}

	if commandExecutionError != nil {
		logger.Errorf("Failed to load ca cert: %s", commandExecutionError.Error())
		return nil, commandExecutionError
	}
	return outputText, nil
}
