package services

import (
	"errors"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/sirupsen/logrus"
)

type FileSystemService interface {
	initialize(logger *logrus.Logger)
	CreateDirectory(system filesystem.FileSystem, path string, recursive bool) error
	SetFileOwner(system filesystem.FileSystem, path string, owner string, recursive bool) error
	SetFilePermissions(system filesystem.FileSystem, path string, permissions int, recursive bool) error
	GetFilesystem(disk *disk.Disk, partition int) (filesystem.FileSystem, error)
}

type FileSystemServiceImpl struct {
	logger *logrus.Logger
}

func (filesystemService *FileSystemServiceImpl) initialize(logger *logrus.Logger) {
	filesystemService.logger = logger
}

func (filesystemService *FileSystemServiceImpl) CreateDirectory(system filesystem.FileSystem, path string, recursive bool) error {

	return nil
}

func (filesystemService *FileSystemServiceImpl) SetFileOwner(system filesystem.FileSystem, path string, owner string, recursive bool) error {
	return nil
}

func (filesystemService *FileSystemServiceImpl) SetFilePermissions(system filesystem.FileSystem, path string, permissions int, recursive bool) error {
	return nil
}

func (filesystemService *FileSystemServiceImpl) GetFilesystem(disk *disk.Disk, partition int) (filesystem.FileSystem, error) {
	if disk == nil {
		filesystemService.logger.Error("Disk provided was nil")
		return nil, errors.New("cannot get filesystem from nil disk pointer")
	}
	table, getPartitionsError := disk.GetPartitionTable()

	if getPartitionsError != nil {
		filesystemService.logger.Errorf("Failed to retrieve disk partitions: %s", getPartitionsError)
		return nil, getPartitionsError
	}

	for _, diskPart := range table.GetPartitions() {
		filesystemService.logger.Debugf("Partition %s found", diskPart.UUID())
	}

	fileSystem, getFileSystemError := disk.GetFilesystem(partition)

	fileinfo, _ := disk.Backend.Stat()
	if getFileSystemError != nil {
		filesystemService.logger.Errorf("Failed to retrieve file system from disk %s at partition %d", fileinfo.Name(), partition)
		return nil, getFileSystemError
	}

	fileInfos, readDirError := fileSystem.ReadDir("/")

	if readDirError != nil {
		filesystemService.logger.Errorf("Failed to read filesystem: %s", readDirError.Error())
	}

	for _, file := range fileInfos {
		filesystemService.logger.Debugf(file.Name())
	}

	return fileSystem, nil
}
