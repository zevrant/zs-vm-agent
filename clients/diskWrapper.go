package clients

import (
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition"
	"io/fs"
)

type DiskWrapper interface {
	GetPartitionTable() (partition.Table, error)
	GetFileSystem(partition int) (FileSystemWrapper, error)
	StatBackend() (fs.FileInfo, error)
}

type DiskWrapperImpl struct {
	disk *disk.Disk
}

func NewDiskWrapper(disk *disk.Disk) DiskWrapper {
	return &DiskWrapperImpl{disk: disk}
}

func (diskClient *DiskWrapperImpl) GetPartitionTable() (partition.Table, error) {
	return diskClient.disk.GetPartitionTable()
}

// partition == 0 means entire disk is the partition
func (diskClient *DiskWrapperImpl) GetFileSystem(partition int) (FileSystemWrapper, error) {
	fileSystem, getFileSystemError := diskClient.disk.GetFilesystem(partition)
	if getFileSystemError != nil {
		return nil, getFileSystemError
	}
	return NewFileSystemWrapper(fileSystem), nil
}

func (diskClient *DiskWrapperImpl) StatBackend() (fs.FileInfo, error) {
	return diskClient.disk.Backend.Stat()
}
