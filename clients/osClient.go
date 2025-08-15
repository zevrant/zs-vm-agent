package clients

import (
	"os"
	"strings"

	"github.com/diskfs/go-diskfs"
	"github.com/sirupsen/logrus"
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
	logger *logrus.Logger
}

func (osClient *OsClientImpl) initialize(logger *logrus.Logger) {
	osClient.logger = logger
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
	sanitizedPath := strings.ReplaceAll(path, "//", "/")
	osClient.logger.Debugf("Creating file %s", sanitizedPath)

	file, getFileError := os.Create(sanitizedPath)

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
