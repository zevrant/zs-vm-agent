package services

import (
	"github.com/sirupsen/logrus"
	"zs-vm-agent/clients"
)

var diskService DiskServiceImpl
var filesystemService FileSystemServiceImpl
var systemdService SystemdServiceImpl
var selinuxService SeLinuxServiceImpl

func Initialize(logger *logrus.Logger) {
	diskService.initialize(logger)
	filesystemService.initialize(logger, clients.GetOsClient(), clients.GetUserClient())
	systemdService.initialize(logger)
	selinuxService.initialize(logger)
}

func GetDiskService() DiskService {
	return &diskService
}

func GetFileSystemService() FileSystemService {
	return &filesystemService
}

func GetSystemdService() SystemdService {
	return &systemdService
}

func GetSeLinuxService() SeLinuxService {
	return &selinuxService
}
