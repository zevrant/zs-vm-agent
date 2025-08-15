package dns

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"zs-vm-agent/clients"
	"zs-vm-agent/services"

	"github.com/sirupsen/logrus"
)

func SetupBind9(logger *logrus.Logger, vmDetails clients.ProxmoxVm) error {

	filesystemService := services.GetFileSystemService()

	copyFilesError := copyDnsFiles(logger, filesystemService)

	if copyFilesError != nil {
		return copyFilesError
	}

	copyFilesError = copyKeepalivedFiles(logger, filesystemService)

	if copyFilesError != nil {
		return copyFilesError
	}

	systemdService := services.GetSystemdService()

	startServicesError := startServices(logger, systemdService)

	if startServicesError != nil {
		return startServicesError
	}

	return nil
}

func copyDnsFiles(logger *logrus.Logger, filesystemService services.FileSystemService) error {
	// create folders
	createFilesystemFolderError := filesystemService.CreateRootFsDirectory("/etc/named/zones", true, 0750)
	if createFilesystemFolderError != nil {
		logger.Errorf("Failed to create zones folder for named: %s", createFilesystemFolderError.Error())
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

		if fileName == "." || fileName == ".." || fileName == "lost+found" {
			continue
		}

		var copyError error = nil
		var setOwnerError error = nil
		var setPermissionsError error = nil
		logger.Debugf("Copying file %s", fileName)

		if fileName == "named.conf" {
			copyError = filesystemService.CopySingleFileToRootFs(fs, fileName, "/etc/named.conf")
			setOwnerError = filesystemService.SetRootFsOwner("/etc/named.conf", "named", false)
			setPermissionsError = filesystemService.SetRootFsPermissions("/etc/named.conf", 0640, false)
		} else if strings.Contains(fileName, "named.conf.") {
			copyError = filesystemService.CopySingleFileToRootFs(fs, fmt.Sprintf("/%s", fileName), fmt.Sprintf("/etc/named/%s", fileName))
			setPermissionsError = filesystemService.SetRootFsPermissions("/etc/named.conf", 0640, false)
		} else if fileName != "vm-config.json" {
			copyError = filesystemService.CopySingleFileToRootFs(fs, fmt.Sprintf("/%s", fileName), fmt.Sprintf("/etc/named/zones/%s", fileName))
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

func startServices(logger *logrus.Logger, systemdService services.SystemdService) error {
	for _, service := range []string{"named", "keepalived"} {
		startServiceError := systemdService.StartService(service)

		if startServiceError != nil {
			return startServiceError
		}
		systemStatus, getSystemdStatusError := systemdService.GetServiceStatus(service)

		for systemStatus == 0 && getSystemdStatusError != nil {
			time.Sleep(2 * time.Second)
			systemStatus, getSystemdStatusError = systemdService.GetServiceStatus(service)
		}

		if getSystemdStatusError != nil {
			return getSystemdStatusError
		}

		if systemStatus == -1 {
			logger.Errorf("%s%s failed to start", strings.ToUpper(service[0:1]), service[1:])
			return errors.New("named failed to start")
		}
	}
	return nil
}

func copyKeepalivedFiles(logger *logrus.Logger, filesystemService services.FileSystemService) error {
	createFilesystemFolderError := filesystemService.CreateRootFsDirectory("/etc/keepalived/", true, 0750)
	if createFilesystemFolderError != nil {
		logger.Errorf("Failed to create keepalived folder: %s", createFilesystemFolderError.Error())
		return createFilesystemFolderError
	}
	fs, getFileSystemError := filesystemService.GetBlockFilesystem("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi2")

	if getFileSystemError != nil {
		return getFileSystemError
	}

	copyError := filesystemService.CopySingleFileToRootFs(fs, "/keepalived.conf", "/etc/keepalived/keepalived.conf")

	if copyError != nil {
		logger.Errorf("Failed to copy file %s to root filesystem: %s", "keepalived.conf", copyError)
		return copyError
	}

	setOwnerError := filesystemService.SetRootFsOwner("/etc/named.conf", "named", false)

	if setOwnerError != nil {
		return setOwnerError
	}

	setPermissionsError := filesystemService.SetRootFsPermissions("/etc/keepalived/keepalived.conf", 0600, false)

	if setPermissionsError != nil {
		return setPermissionsError
	}
	return nil
}
