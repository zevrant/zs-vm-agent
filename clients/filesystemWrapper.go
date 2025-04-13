package clients

import (
	"github.com/diskfs/go-diskfs/filesystem"
	"os"
)

type FileSystemWrapper interface {
	OpenFile(path string, flag int) (FileWrapper, error)
	ReadDir(path string) ([]os.FileInfo, error)
}

type FileSystemWrapperImpl struct {
	fileSystem filesystem.FileSystem
}

func NewFileSystemWrapper(system filesystem.FileSystem) FileSystemWrapper {
	return &FileSystemWrapperImpl{fileSystem: system}
}

func (filesystemWrapper *FileSystemWrapperImpl) OpenFile(path string, flag int) (FileWrapper, error) {
	file, openFileError := filesystemWrapper.fileSystem.OpenFile(path, flag)
	if openFileError != nil {
		return nil, openFileError
	}
	return NewFilesystemFileWrapper(file), nil
}

func (filesystemWrapper *FileSystemWrapperImpl) ReadDir(path string) ([]os.FileInfo, error) {
	return filesystemWrapper.fileSystem.ReadDir(path)
}

//func (filesystemWrapper *FileSystemWrapperImpl)
