package services

import "github.com/sirupsen/logrus"

var diskService DiskServiceImpl
var filesystemService FileSystemServiceImpl

func Initialize(logger *logrus.Logger) {
	diskService.initialize(logger)
	filesystemService.initialize(logger)
}

func GetDiskService() DiskService {
	return &diskService
}

func GetFileSystemService() FileSystemService {
	return &filesystemService
}
