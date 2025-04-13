package clients

import (
	"github.com/diskfs/go-diskfs/filesystem"
	"os"
)

type FileWrapper interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
	Close() error
}

type FileWrapperImpl struct {
	file           *os.File
	filesystemFile filesystem.File
}

func NewOsFileWrapper(file *os.File) FileWrapper {
	return &FileWrapperImpl{file: file}
}

func NewFilesystemFileWrapper(file filesystem.File) FileWrapper {
	return &FileWrapperImpl{filesystemFile: file}
}

func (fileWrapper *FileWrapperImpl) Write(toBeWritten []byte) (int, error) {
	if fileWrapper.file != nil {
		return fileWrapper.file.Write(toBeWritten)
	} else {
		return fileWrapper.filesystemFile.Write(toBeWritten)
	}
}

func (fileWrapper *FileWrapperImpl) Read(readBuffer []byte) (int, error) {
	if fileWrapper.file != nil {
		return fileWrapper.file.Read(readBuffer)
	} else {
		return fileWrapper.filesystemFile.Read(readBuffer)
	}
}

func (fileWrapper *FileWrapperImpl) Seek(offset int64, whence int) (int64, error) {
	if fileWrapper.file != nil {
		return fileWrapper.file.Seek(offset, whence)
	} else {
		return fileWrapper.filesystemFile.Seek(offset, whence)
	}
}

func (fileWrapper *FileWrapperImpl) Close() error {
	if fileWrapper.file != nil {
		return fileWrapper.file.Close()
	} else {
		return fileWrapper.filesystemFile.Close()
	}
}
