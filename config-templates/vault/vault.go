package vault

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"zs-vm-agent/clients"
	"zs-vm-agent/services"

	"github.com/sirupsen/logrus"
)

func Setup(logger *logrus.Logger, vmDetails clients.ProxmoxVm) error {
	filesystemService := services.GetFileSystemService()
	diskService := services.GetDiskService()
	systemdService := services.GetSystemdService()

	// scsi-0QEMU_QEMU_HARDDISK_drive-scsi2
	configDrive, getFileSystemError := filesystemService.GetBlockFilesystem("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi2")

	if getFileSystemError != nil {
		return getFileSystemError
	}
	initError := initializeDataStore(logger, diskService, filesystemService)

	if initError != nil {
		return initError
	}

	logger.Info("Copying Vault Configurations.")

	copyFilesError := copyFiles(logger, filesystemService, configDrive)

	if copyFilesError != nil {
		return copyFilesError
	}

	logger.Info("Starting Vault Service")

	startServiceError := systemdService.StartService("vault")

	if startServiceError != nil {
		return startServiceError
	}

	logger.Info("Unsealing Vault")

	vaultUnsealError := unsealVault(logger, filesystemService, configDrive)

	if vaultUnsealError != nil {
		return nil
	}

	return nil
}

func initializeDataStore(logger *logrus.Logger, diskService services.DiskService, filesystemService services.FileSystemService) error {
	diskPath := "/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1"
	logger.Debug("Initializing Data Store")
	_, statFileError := clients.GetOsClient().StatFile(fmt.Sprintf("%s-part1", diskPath))
	logger.Debugf("Stat file attempt made on %s-part1", diskPath)
	if statFileError != nil && !strings.Contains(statFileError.Error(), fmt.Sprintf("stat %s-part1: no such file or directory", diskPath)) {
		errorMessage := statFileError.Error()
		logger.Debug(errorMessage)
		return statFileError
	}

	logger.Debugf("Successfully located %s", fmt.Sprintf("%s-part1", diskPath))

	dataDrive, getDiskError := diskService.GetDisk(diskPath)
	if getDiskError != nil {
		return getDiskError
	}

	partTable, getPartTableError := dataDrive.GetPartitionTable()

	if (getPartTableError != nil && strings.Contains(getPartTableError.Error(), "unknown disk partition type")) || len(partTable.GetPartitions()) == 0 {
		createDiskPartitionError := diskService.CreatePartition(diskPath)

		if createDiskPartitionError != nil {
			logger.Errorf("Failed to create partition for disk %s: %s", diskPath, createDiskPartitionError.Error())
			return createDiskPartitionError
		}
		time.Sleep(5 * time.Second)
	} else if getPartTableError != nil {
		logger.Errorf("Failed to get Partition Table from disk %s: %s", diskPath, getPartTableError.Error())
		return getPartTableError
	}

	partTable, getPartTableError = dataDrive.GetPartitionTable()

	if getPartTableError != nil {
		logger.Errorf("Failed to get Partition Table from disk after creation %s: %s", diskPath, getPartTableError.Error())
		return getPartTableError
	}
	if len(partTable.GetPartitions()) == 0 {
		logger.Error("No partitions found after creating a new partition")
		return errors.New("no partitions found after creating new partition")
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
			return createFilesystemError
		}
	}

	mountError := filesystemService.MountFilesystem(fmt.Sprintf("%s-part1", diskPath), "/opt/vault")

	if mountError != nil {
		return mountError
	}

	setFolderOwnerError := filesystemService.SetRootFsOwner("/opt/vault", "vault", true)
	if setFolderOwnerError != nil {
		return setFolderOwnerError
	}

	return nil
}

func copyFiles(logger *logrus.Logger, filesystemService services.FileSystemService, configs clients.FileSystemWrapper) error {

	logger.Debug("Copying vault.hcl")
	copyFileError := filesystemService.CopyFilesToRootFs(configs, "vault.hcl", "/etc/vault.d/vault.hcl", false)

	if copyFileError != nil {
		return copyFileError
	}

	logger.Debug("Copying public cert")
	copyFileError = filesystemService.CopyFilesToRootFs(configs, "vault-public.pem", "/etc/vault.d/tls.crt", false)

	if copyFileError != nil {
		return copyFileError
	}

	logger.Debug("Copying private key")
	copyFileError = filesystemService.CopyFilesToRootFs(configs, "vault-private.pem", "/etc/vault.d/tls.pem", false)

	return copyFileError
}

func unsealVault(logger *logrus.Logger, filesystemService services.FileSystemService, configs clients.FileSystemWrapper) error {
	vaultkeyBytes, readKeyError := filesystemService.ReadFileContentsFromFilesystem(configs, "vault-key-1")

	if readKeyError != nil {
		return readKeyError
	}

	vaultKey1 := string(vaultkeyBytes)

	vaultkeyBytes, readKeyError = filesystemService.ReadFileContentsFromFilesystem(configs, "vault-key-2")
	if readKeyError != nil {
		return readKeyError
	}

	vaultKey2 := string(vaultkeyBytes)
	vaultkeyBytes, readKeyError = filesystemService.ReadFileContentsFromFilesystem(configs, "vault-key-3")
	if readKeyError != nil {
		return readKeyError
	}

	vaultKey3 := string(vaultkeyBytes)

	vaultApiBytes, readApiError := filesystemService.ReadFileContentsFromFilesystem(configs, "vault-api-url")
	if readApiError != nil {
		return readKeyError
	}

	vaultApiUrl := string(vaultApiBytes)

	unsealError := services.GetVaultService().UnsealVault(vaultApiUrl, []string{vaultKey1, vaultKey2, vaultKey3})

	if unsealError != nil {
		return unsealError
	}

	return nil
}
