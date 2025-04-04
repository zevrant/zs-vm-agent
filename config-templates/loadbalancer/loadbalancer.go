package loadbalancer

import (
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/sirupsen/logrus"
	"zs-vm-agent/services"
)

func SetupLoadbalancer(logger *logrus.Logger) error {
	logger.Info("Setting up as loadbalancer")
	var diskService services.DiskService = services.GetDiskService()
	var fileSystemService services.FileSystemService = services.GetFileSystemService()

	filePermissionError := initializeFileSystem(logger, diskService, fileSystemService)

	if filePermissionError != nil {
		return nil
	}

	mountError := mountDisks(diskService)
	if mountError != nil {
		return mountError
	}

	return nil
}

func mountDisks(diskService services.DiskService) error {
	mountHaproxyConfigError := diskService.MountPartition("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1", "/etc/haproxy/")
	if mountHaproxyConfigError != nil {
		return mountHaproxyConfigError
	}

	mountKeepalivedConfigError := diskService.MountPartition("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi2", "/etc/keepalived/")

	if mountKeepalivedConfigError != nil {
		return mountHaproxyConfigError
	}

	return nil
}
func initializeFileSystem(logger *logrus.Logger, diskService services.DiskService, filesystemService services.FileSystemService) error {
	d, openDiskError := diskService.GetDisk("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1")

	if openDiskError != nil {
		return openDiskError
	}

	fs, getFileSystemError := filesystemService.GetFilesystem(d, 0)

	if getFileSystemError != nil {
		return getFileSystemError
	}

	dirs := []string{
		"/etc/haproxy/conf.d",
		"/etc/haproxy/certs",
	}
	directoryCreationError := createDirectories(dirs, fs, filesystemService)

	if directoryCreationError != nil {
		return directoryCreationError
	}

	setOwnerError := filesystemService.SetFileOwner(fs, "/etc/haproxy", "haproxy", true)

	if setOwnerError != nil {
		return setOwnerError
	}

	setPermissionsError := filesystemService.SetFilePermissions(fs, "/etc/haproxy/certs/", 0600, true)

	if setPermissionsError != nil {
		return setPermissionsError
	}

	return nil
}

func createDirectories(dirs []string, filesystem filesystem.FileSystem, filesystemService services.FileSystemService) error {
	for _, directory := range dirs {
		setPermissionsError := filesystemService.CreateDirectory(filesystem, directory, false)

		if setPermissionsError != nil {
			return setPermissionsError
		}
	}
	return nil
}
