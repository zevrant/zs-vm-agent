package loadbalancer

import (
	"errors"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/sirupsen/logrus"
	"zs-vm-agent/services"
)

func SetupLoadBalancer(logger *logrus.Logger) error {
	logger.Info("Setting up as load balancer")
	var diskService services.DiskService = services.GetDiskService()
	var fileSystemService services.FileSystemService = services.GetFileSystemService()

	filePermissionError := initializeFileSystem(logger, diskService, fileSystemService)

	if filePermissionError != nil {
		return nil
	}
	logger.Info("Files successfully loaded")

	startServicesError := startServices(logger)

	if startServicesError != nil {
		return startServicesError
	}

	checkHealthError := checkServicesHealth(logger)

	if checkHealthError != nil {
		return checkHealthError
	}

	return nil
}

func mountDisks(diskService services.DiskService) error {
	mountHaproxyConfigError := diskService.MountPartition("scsi-0QEMU_QEMU_HARDDISK_drive-scsi1", "/var/run/zevrant-services/haproxy/")
	if mountHaproxyConfigError != nil {
		return mountHaproxyConfigError
	}

	mountKeepalivedConfigError := diskService.MountPartition("scsi-0QEMU_QEMU_HARDDISK_drive-scsi2", "/etc/keepalived/")

	if mountKeepalivedConfigError != nil {
		return mountHaproxyConfigError
	}

	return nil
}
func initializeFileSystem(logger *logrus.Logger, diskService services.DiskService, filesystemService services.FileSystemService) error {
	fs, getFileSystemError := filesystemService.GetBlockFilesystem("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1")

	if getFileSystemError != nil {
		return getFileSystemError
	}

	dirs := []string{
		"/etc/haproxy/conf.d",
		"/etc/haproxy/certs",
		"/var/run/zevrant-services/haproxy",
	}
	directoryCreationError := createRootFsDirectories(dirs, filesystemService)

	if directoryCreationError != nil {
		return directoryCreationError
	}

	setOwnerError := filesystemService.SetRootFsFileOwner("/etc/haproxy", "haproxy", true)

	if setOwnerError != nil {
		return setOwnerError
	}

	setPermissionsError := filesystemService.SetRootFsFilePermissions("/etc/haproxy/certs/", 0600, true)

	if setPermissionsError != nil {
		return setPermissionsError
	}

	copyFilesError := copyFiles(fs, map[string]string{
		"haproxy.cfg": "/etc/haproxy/haproxy.cfg",
		"certs":       "/etc/haproxy/certs",
		"conf.d":      "/etc/haproxy/conf.d/",
	})

	if copyFilesError != nil {
		return copyFilesError
	}

	return nil
}

func createRootFsDirectories(dirs []string, filesystemService services.FileSystemService) error {
	for _, directory := range dirs {
		setPermissionsError := filesystemService.CreateRootFsDirectory(directory, true)

		if setPermissionsError != nil {
			return setPermissionsError
		}
	}
	return nil
}

func copyFiles(sourceFs filesystem.FileSystem, sources map[string]string) error {
	for sourceFile, destFile := range sources {
		copyError := services.GetFileSystemService().CopyFilesToRootFs(sourceFs, sourceFile, destFile, true)
		if copyError != nil {
			return copyError
		}
	}
	return nil
}

func startServices(logger *logrus.Logger) error {
	systemdService := services.GetSystemdService()

	logger.Info("Starting service keepalived")
	startKeepalivedError := systemdService.StartService("keepalived")

	if startKeepalivedError != nil {
		return startKeepalivedError
	}
	logger.Info("Keepalived successfully started")

	logger.Info("Starting service haproxy")
	startHaproxyError := systemdService.StartService("haproxy")
	if startHaproxyError != nil {
		return startHaproxyError
	}

	logger.Info("Haproxy successfully started")
	return nil
}

func checkServicesHealth(logger *logrus.Logger) error {
	systemServices := []string{"haproxy", "keepalived"}
	systemdService := services.GetSystemdService()

	for _, service := range systemServices {
		var status int = 0
		var getStatusError error
		for status == 0 {
			status, getStatusError = systemdService.GetServiceStatus(service)
			if getStatusError != nil {
				return getStatusError
			}
		}
		if status != 1 {
			logger.Errorf("Service %s failed to fully start", service)
			return errors.New("failed to start service " + service)
		}
	}
	return nil
}
