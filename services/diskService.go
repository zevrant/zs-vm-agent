package services

import (
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/sirupsen/logrus"
	"os/exec"
)

type DiskService interface {
	initialize(logger *logrus.Logger)
	MountPartition(partitionPath string, mountLocation string) error
	GetDisk(path string) (*disk.Disk, error)
}

type DiskServiceImpl struct {
	logger *logrus.Logger
}

func (diskService *DiskServiceImpl) initialize(logger *logrus.Logger) {
	diskService.logger = logger
}

func (diskService *DiskServiceImpl) MountPartition(partitionPath string, mountLocation string) error {
	command := exec.Command("mount", "-c", "/usr/bin/mount", partitionPath, mountLocation)

	output, executionError := command.Output()

	if executionError != nil {
		diskService.logger.Errorf("Failed to execute mount, %s", executionError.Error())
		return executionError
	}

	fmt.Println(output)

	return nil
}

func (diskService *DiskServiceImpl) GetDisk(devicePath string) (*disk.Disk, error) {
	openedDisk, openDiskError := diskfs.Open(devicePath)

	if openDiskError != nil {
		diskService.logger.Errorf("Failed to open disk at %s: %s", devicePath, openDiskError.Error())
		return nil, openDiskError
	}

	return openedDisk, nil
}
