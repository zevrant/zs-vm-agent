package services

import "github.com/sirupsen/logrus"

var diskService DiskServiceImpl
var filesystemService FileSystemServiceImpl
var systemdService SystemdServiceImpl

func Initialize(logger *logrus.Logger) {
	diskService.initialize(logger)
	filesystemService.initialize(logger)
	systemdService.initialize(logger)
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
