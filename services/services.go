package services

import (
	"zs-vm-agent/clients"

	"github.com/sirupsen/logrus"
)

var diskService DiskServiceImpl
var filesystemService FileSystemServiceImpl
var systemdService SystemdServiceImpl
var selinuxService SeLinuxServiceImpl
var vaultService VaultServiceImpl

func Initialize(logger *logrus.Logger) {
	diskService.initialize(logger)
	filesystemService.initialize(logger, clients.GetOsClient(), clients.GetUserClient())
	systemdService.initialize(logger)
	selinuxService.initialize(logger)
	vaultService.initialize(logger)
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

func GetVaultService() VaultService {
	return &vaultService
}
