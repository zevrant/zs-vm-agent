package clients

import (
	"github.com/diskfs/go-diskfs"
	"github.com/sirupsen/logrus"
	"os"
)

type OsClient interface {
	initialize(logger *logrus.Logger)
	StatFile(path string) (os.FileInfo, error)
	Mkdir(path string, permissions int) error
	OpenDisk(path string) (DiskWrapper, error)
	CreateFile(path string) (FileWrapper, error)
	SetOwner(path string, ownerId int, groupId int) error
	ReadDir(path string) ([]os.DirEntry, error)
	SetPermissions(path string, permissions int) error
	OpenFile(path string) (*os.File, error)
}

type OsClientImpl struct {
}

func (osClient *OsClientImpl) initialize(logger *logrus.Logger) {
	//TODO implement me
	panic("implement me")
}

func (osClient *OsClientImpl) StatFile(path string) (os.FileInfo, error) {
	fileInfo, readDirectoryError := os.Stat(path)

	if readDirectoryError != nil {
		return nil, readDirectoryError
	}

	return fileInfo, nil
}

func (osClient *OsClientImpl) Mkdir(path string, permissions int) error {
	return os.Mkdir(path, 0755)
}

func (osClient *OsClientImpl) OpenDisk(path string) (DiskWrapper, error) {
	openedDisk, getDiskError := diskfs.Open(path)
	if getDiskError != nil {
		return nil, getDiskError
	}

	return NewDiskWrapper(openedDisk), nil
}

func (osClient *OsClientImpl) CreateFile(path string) (FileWrapper, error) {
	file, getFileError := os.Create(path)

	if getFileError != nil {
		return nil, getFileError
	}

	return NewOsFileWrapper(file), nil
}

func (osClient *OsClientImpl) SetOwner(path string, ownerId int, groupId int) error {
	return os.Chown(path, ownerId, groupId)
}

func (osClient *OsClientImpl) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (osClient *OsClientImpl) SetPermissions(path string, permissions int) error {
	return os.Chmod(path, os.FileMode(permissions))
}

func (osClient *OsClientImpl) OpenFile(path string) (*os.File, error) {
	return os.Open(path)
}
