package services

import (
	"bytes"
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
	diskService.logger.Debugf("Mounting device %s at %s", partitionPath, mountLocation)
	diskService.logger.Debugf("/usr/bin/mount ID=%s %s", partitionPath, mountLocation)
	command := exec.Command("/usr/bin/mount", fmt.Sprintf("ID=%s", partitionPath), mountLocation)
	//command := exec.Command("whoami")
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	executionError := command.Run()
	diskService.logger.Debug(string(output.Bytes()))
	if executionError != nil {
		diskService.logger.Errorf("Failed to execute mount, %s", executionError.Error())
		return executionError
	}

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
