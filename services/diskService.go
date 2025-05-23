package services

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/sirupsen/logrus"
)

type DiskService interface {
	initialize(logger *logrus.Logger)
	GetDisk(path string) (*disk.Disk, error)
	CreatePartition(diskPath string) error
}

type DiskServiceImpl struct {
	logger *logrus.Logger
}

func (diskService *DiskServiceImpl) initialize(logger *logrus.Logger) {
	diskService.logger = logger
}

func (diskService *DiskServiceImpl) GetDisk(devicePath string) (*disk.Disk, error) {
	openedDisk, openDiskError := diskfs.Open(devicePath)

	if openDiskError != nil {
		diskService.logger.Errorf("Failed to open disk at %s: %s", devicePath, openDiskError.Error())
		return nil, openDiskError
	}

	return openedDisk, nil
}

func (diskService *DiskServiceImpl) CreatePartition(diskPath string) error {
	writeLayoutError := os.WriteFile("/tmp/layout", []byte("start=        2048"), 0600)

	if writeLayoutError != nil {
		diskService.logger.Errorf("Failed to write layout config to disk: %s", writeLayoutError.Error())
		return writeLayoutError
	}

	command := exec.Command("bash", "-c", fmt.Sprintf("echo 'start=2048' | sfdisk --label gpt --force --wipe always %s && partprobe", diskPath))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	//reader, writer := io.Pipe()

	commandExecutionError := command.Start()

	if commandExecutionError != nil {
		diskService.logger.Errorf("Failed to create partition at %s: %s", diskPath, commandExecutionError.Error())
		return commandExecutionError
	}

	commandWaitError := command.Wait()

	//for _, line := range strings.Split(string(outputText), "\n") {
	//	diskService.logger.Info(line)
	//}

	if commandWaitError != nil {
		diskService.logger.Errorf("Failed to create partition at %s: %s", diskPath, commandWaitError.Error())
		return commandWaitError
	}
	return nil
}
