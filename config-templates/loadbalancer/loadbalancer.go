package loadbalancer

import (
	"encoding/json"
	"errors"
	"zs-vm-agent/clients"
	"zs-vm-agent/services"

	"github.com/sirupsen/logrus"
)

var systemServices = [...]string{
	"keepalived",
	"haproxy",
}

type fileMapping struct {
	path                      string
	permissions               int
	directoryFilesPermissions int
}

func SetupLoadBalancer(logger *logrus.Logger, vmDetails clients.ProxmoxVm) error {
	logger.Info("Setting up as load balancer")
	var fileSystemService services.FileSystemService = services.GetFileSystemService()

	filePermissionError := initializeFileSystem(logger, fileSystemService)

	if filePermissionError != nil {
		return filePermissionError
	}
	logger.Info("Files successfully loaded")

	configurationError := performPrerunConfiguration(logger)

	if configurationError != nil {
		return configurationError
	}

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

func initializeFileSystem(logger *logrus.Logger, filesystemService services.FileSystemService) error {
	fs, getFileSystemError := filesystemService.GetBlockFilesystem("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1")

	logger.Info("Creating directories")

	if getFileSystemError != nil {
		return getFileSystemError
	}

	dirs := map[string]int{
		"/etc/haproxy":        0755,
		"/etc/haproxy/conf.d": 0755,
		"/etc/haproxy/certs":  0700,
	}

	for directory, permissions := range dirs {
		logger.Debugf("Creating directory %s", directory)
		directoryCreationError := filesystemService.CreateRootFsDirectory(directory, true, permissions)

		if directoryCreationError != nil {
			return directoryCreationError
		}
		logger.Debugf("Setting root fs owner for %s to haproxy", directory)
		setOwnerError := filesystemService.SetRootFsOwner(directory, "haproxy", false)

		if setOwnerError != nil {
			return setOwnerError
		}
	}

	logger.Info("Copying config files...")

	copyFilesError := copyFiles(fs, map[string]fileMapping{
		"haproxy.cfg": {
			path:        "/etc/haproxy/haproxy.cfg",
			permissions: 0755,
		},
		"certs": {
			path:                      "/etc/haproxy/certs",
			permissions:               0600,
			directoryFilesPermissions: 0600,
		},
		"conf.d": {
			path:                      "/etc/haproxy/conf.d/",
			permissions:               0755,
			directoryFilesPermissions: 0644,
		},
		"vm-config.json": {
			path:        "/tmp/vm-config.json",
			permissions: 0400,
		},
	}, logger)

	if copyFilesError != nil {
		return copyFilesError
	}

	fs, getFileSystemError = filesystemService.GetBlockFilesystem("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi2")

	if getFileSystemError != nil {
		return getFileSystemError
	}

	copyFilesError = copyFiles(fs, map[string]fileMapping{
		"keepalived.conf": {
			path:        "/etc/keepalived/keepalived.conf",
			permissions: 0600,
		},
	}, logger)

	if copyFilesError != nil {
		return copyFilesError
	}

	selinuxService := services.GetSeLinuxService()

	changeContextError := selinuxService.ChangeContext("/etc/haproxy", "system_u", "object_r", "etc_t", true)

	if changeContextError != nil {
		return changeContextError
	}

	changeContextError = selinuxService.ChangeContext("/etc/keepalived", "system_u", "object_r", "keepalived_var_run_t", true)

	if changeContextError != nil {
		return changeContextError
	}

	return nil
}

func copyFiles(sourceFs clients.FileSystemWrapper, sources map[string]fileMapping, logger *logrus.Logger) error {
	filesystemService := services.GetFileSystemService()
	for sourceFile, destFile := range sources {
		logger.Debugf("Triggering copy for %s to %s", sourceFile, destFile.path)
		copyError := filesystemService.CopyFilesToRootFs(sourceFs, sourceFile, destFile.path, true)
		if copyError != nil {
			return copyError
		}
		logger.Debugf("Setting permissions for %s", sourceFile)
		setPermissionsError := filesystemService.SetRootFsPermissions(destFile.path, destFile.directoryFilesPermissions, true)

		if setPermissionsError != nil {
			logger.Error(setPermissionsError.Error())
			return setPermissionsError
		}
		logger.Debugf("Finished copy for %s", sourceFile)
	}
	return nil
}

func startServices(logger *logrus.Logger) error {

	systemdService := services.GetSystemdService()

	for _, service := range systemServices {
		logger.Infof("Starting service %s", service)
		startServiceError := systemdService.StartService(service)

		if startServiceError != nil {
			return startServiceError
		}
		logger.Infof("%s successfully started", service)
	}

	return nil
}

func checkServicesHealth(logger *logrus.Logger) error {
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

func performPrerunConfiguration(logger *logrus.Logger) error {
	contents, getFileContentsError := services.GetFileSystemService().ReadFileContents("/tmp/vm-config.json")

	if getFileContentsError != nil {
		return getFileContentsError
	}

	logger.Debugf("content is %s", string(contents))

	var c Config

	jsonError := json.Unmarshal(contents, &c)
	logger.Debugf("Unmarshalled json")

	if jsonError != nil {
		logger.Errorf("Failed to parse vm-configuration json: %s", jsonError.Error())
		return jsonError
	}

	logger.Debugf("performing port configurations %d", len(c.Ports))
	test, _ := json.Marshal(c)
	logger.Debugf("Test: %s", string(test))
	for _, port := range c.Ports {
		logger.Debugf("Opening port %d/%s", port.Port, port.Protocol)
		openPortError := services.GetSeLinuxService().OpenInboundPort(port.Port, port.Protocol)
		if openPortError != nil {
			return openPortError
		}
	}

	logger.Debug("Opening haproxy to allow all outbound connections")
	allowOutboundError := services.GetSeLinuxService().AllowAllOutboundConnection()

	if allowOutboundError != nil {
		logger.Errorf("Failed to allow haproxy full outbound access: %s", allowOutboundError.Error())
		return allowOutboundError
	}

	return nil
}

type Config struct {
	Ports []portMapping
}

type portMapping struct {
	Port     int
	Protocol string
}
