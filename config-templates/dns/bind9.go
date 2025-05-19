package dns

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
	"zs-vm-agent/services"
)

func SetupBind9(logger *logrus.Logger) error {
	filesystemService := services.GetFileSystemService()

	copyFilesError := copyFiles(logger, filesystemService)

	if copyFilesError != nil {
		return copyFilesError
	}

	systemdService := services.GetSystemdService()

	startServiceError := systemdService.StartService("named")

	if startServiceError != nil {
		return startServiceError
	}

	systemStatus, getSystemdStatusError := systemdService.GetServiceStatus("named")

	for systemStatus == 0 && getSystemdStatusError != nil {
		time.Sleep(2 * time.Second)
		systemStatus, getSystemdStatusError = systemdService.GetServiceStatus("named")
	}

	if getSystemdStatusError != nil {
		return getSystemdStatusError
	}

	if systemStatus == -1 {
		logger.Error("Named failed to start")
		return errors.New("named failed to start")
	}

	return nil
}

func copyFiles(logger *logrus.Logger, filesystemService services.FileSystemService) error {
	// create folders
	createFilesystemFolderError := filesystemService.CreateRootFsDirectory("/etc/named/zones", true, 0750)
	if createFilesystemFolderError != nil {
		logger.Errorf("Failed to create zones forlder for named: %s", createFilesystemFolderError.Error())
		return createFilesystemFolderError
	}

	fs, getFileSystemError := filesystemService.GetBlockFilesystem("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1")

	if getFileSystemError != nil {
		return getFileSystemError
	}

	fileInfos, readDirectoryError := fs.ReadDir("/")

	if readDirectoryError != nil {
		logger.Errorf("Failed to read root directory of filesystem %s: %s", fs.GetFilesystemLabel(), readDirectoryError.Error())
		return readDirectoryError
	}

	for _, info := range fileInfos {
		fileName := info.Name()
		var copyError error = nil
		var setOwnerError error = nil
		var setPermissionsError error = nil
		if fileName == "named.conf" {
			copyError = filesystemService.CopyFilesToRootFs(fs, fmt.Sprintf("/%s", fileName), "/etc/named.conf", false)
			setOwnerError = filesystemService.SetRootFsOwner("/etc/named.conf", "named", false)
			setPermissionsError = filesystemService.SetRootFsPermissions("/etc/named.conf", 0640, false)
		} else if strings.Contains(fileName, "named.conf.") {
			copyError = filesystemService.CopyFilesToRootFs(fs, fmt.Sprintf("/%s", fileName), fmt.Sprintf("/etc/named/%s", fileName), false)
			setPermissionsError = filesystemService.SetRootFsPermissions("/etc/named.conf", 0640, false)
		} else if fileName != "vm-config.json" {
			copyError = filesystemService.CopyFilesToRootFs(fs, fmt.Sprintf("/%s", fileName), fmt.Sprintf("/etc/named/zones/%s", fileName), false)
			setPermissionsError = filesystemService.SetRootFsPermissions(fmt.Sprintf("/etc/named/zones/%s", fileName), 0640, false)
		}

		if copyError != nil {
			logger.Errorf("Failed to copy file %s to root filesystem: %s", fileName, copyError)
			return copyError
		}

		if setOwnerError != nil {
			return setOwnerError
		}

		if setPermissionsError != nil {
			return setPermissionsError
		}

	}

	setOwnerError := filesystemService.SetRootFsOwner("/etc/named/", "named", true)

	if setOwnerError != nil {
		return setOwnerError
	}

	setPermissionsError := filesystemService.SetRootFsPermissions("/etc/named/", 0750, false)

	if setPermissionsError != nil {
		return setPermissionsError
	}

	return nil
}
