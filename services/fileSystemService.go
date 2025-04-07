package services

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

type FileSystemService interface {
	initialize(logger *logrus.Logger)
	CreateRootFsDirectory(path string, recursive bool) error
	SetRootFsFileOwner(path string, owner string, recursive bool) error
	SetRootFsFilePermissions(path string, permissions int, recursive bool) error
	GetFilesystem(disk *disk.Disk, partition int) (filesystem.FileSystem, error)
	GetBlockFilesystem(devicePath string) (filesystem.FileSystem, error)
	CopyFilesToRootFs(sourceFilesystem filesystem.FileSystem, sourcePath string, destPath string, recursive bool) error
}

type FileSystemServiceImpl struct {
	logger *logrus.Logger
}

func (filesystemService *FileSystemServiceImpl) initialize(logger *logrus.Logger) {
	filesystemService.logger = logger
}

func (filesystemService *FileSystemServiceImpl) CreateRootFsDirectory(path string, recursive bool) error {
	var pathParts []string = strings.Split(path, "/")
	var currentPath = ""
	for _, pathPart := range pathParts {
		currentPath = strings.Replace(fmt.Sprintf("%s%s/", currentPath, pathPart), "//", "/", -1)
		_, readDirectoryError := os.Stat(currentPath)

		if readDirectoryError != nil {
			expectedErrorString := fmt.Sprintf("stat %s: no such file or directory", currentPath)
			filesystemService.logger.Debugf("%s == %s", readDirectoryError.Error(), expectedErrorString)
			if readDirectoryError.Error() == expectedErrorString {
				filesystemService.logger.Debugf("true")
			}
		}

		if readDirectoryError != nil && readDirectoryError.Error() == fmt.Sprintf("stat %s: no such file or directory", currentPath) && (recursive || strings.Contains(currentPath, path)) {
			createDirectoryError := os.Mkdir(currentPath, 0755)
			if createDirectoryError != nil {
				filesystemService.logger.Errorf("Failed to create directory %s: %s", currentPath, createDirectoryError.Error())
				return createDirectoryError
			}
		} else if readDirectoryError != nil {
			readDirectoryErrorString := readDirectoryError.Error()
			filesystemService.logger.Errorf("Failed to read directory %s: %s", currentPath, readDirectoryErrorString)
			return readDirectoryError
		}
		filesystemService.logger.Infof("Directory %s already exists, skipping...", currentPath)
	}
	return nil
}

func (filesystemService *FileSystemServiceImpl) SetRootFsFileOwner(path string, owner string, recursive bool) error {
	return nil
}

func (filesystemService *FileSystemServiceImpl) SetRootFsFilePermissions(path string, permissions int, recursive bool) error {
	return nil
}

func (filesystemService *FileSystemServiceImpl) GetFilesystem(disk *disk.Disk, partition int) (filesystem.FileSystem, error) {
	if disk == nil {
		filesystemService.logger.Error("Disk provided was nil")
		return nil, errors.New("cannot get filesystem from nil disk pointer")
	}
	table, getPartitionsError := disk.GetPartitionTable()

	if getPartitionsError != nil {
		filesystemService.logger.Errorf("Failed to retrieve disk partitions: %s", getPartitionsError.Error())
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

func (filesystemService *FileSystemServiceImpl) GetBlockFilesystem(devicePath string) (filesystem.FileSystem, error) {
	blockDevice, getDeviceError := diskfs.Open(devicePath)

	if getDeviceError != nil {
		filesystemService.logger.Errorf("Failed to retrieve block device at specified path %s: %s", devicePath, getDeviceError.Error())
		return nil, getDeviceError
	}

	blockFilesystem, getBlockFilesystemError := blockDevice.GetFilesystem(0) //no partition table so 0th partition

	if getBlockFilesystemError != nil {
		filesystemService.logger.Errorf("Failed to retrieve filesystem from block device %s: %s", devicePath, getBlockFilesystemError.Error())
		return nil, getBlockFilesystemError
	}

	filesystemService.logger.Debugf("Successfully retrieved filesystem from block device %s", devicePath)
	return blockFilesystem, nil
}

func (filesystemService *FileSystemServiceImpl) CopyFilesToRootFs(sourceFilesystem filesystem.FileSystem, sourcePath string, destPath string, recursive bool) error {
	var sourceFile filesystem.File
	localSourcePath := sourcePath
	fileInfos, readSourceError := filesystemService.attemptReadDir(sourceFilesystem, sourcePath)

	if readSourceError != nil {
		filesystemService.logger.Debugf("Failed to read source file at %s, cannot continue copy operation", sourcePath)
		return readSourceError
	}

	if fileInfos == nil {
		fileInfo, newDestPath, getFileInfoError := filesystemService.getSingleFileInfo(sourceFilesystem, sourcePath, destPath)
		if getFileInfoError != nil {
			return getFileInfoError
		}
		fileInfos = []os.FileInfo{fileInfo}
		localSourcePath = *newDestPath
	}
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() && (fileInfo.Name() != "." && fileInfo.Name() != "..") {
			copyError := filesystemService.CopyFilesToRootFs(sourceFilesystem, fmt.Sprintf("%s/%s", sourcePath, fileInfo.Name()), fmt.Sprintf("%s/%s", destPath, fileInfo.Name()), recursive)
			if copyError != nil {
				return copyError
			}
		} else if fileInfo.Name() != "." && fileInfo.Name() != ".." {
			sourceFile, readSourceError = sourceFilesystem.OpenFile(fmt.Sprintf("%s/%s", localSourcePath, fileInfo.Name()), 0)
			if readSourceError != nil {
				return readSourceError
			}
			copyFileError := filesystemService.copySingleFileToRootFs(sourceFile, fileInfo.Name(), destPath)
			if copyFileError != nil {
				return copyFileError
			}
			filesystemService.logger.Debugf("Copied %s to %s", sourcePath, destPath)
		}

	}

	return nil
}

func (filesystemService *FileSystemServiceImpl) attemptReadDir(sourceFilesystem filesystem.FileSystem, sourcePath string) ([]os.FileInfo, error) {
	fileInfos, readSourceError := sourceFilesystem.ReadDir(sourcePath)
	expectedErrorMessage := fmt.Sprintf("error reading directory %s: cannot create directory at /%s since it is a file", sourcePath, sourcePath)
	if readSourceError != nil && readSourceError.Error() == expectedErrorMessage {
		return nil, nil
	} else if readSourceError != nil {
		return nil, readSourceError
	}

	return fileInfos, nil
}

func (filesystemService *FileSystemServiceImpl) copySingleFileToRootFs(sourceFile filesystem.File, sourceFileName string, destPath string) error {
	var fileBytes []byte = make([]byte, 4096)
	var fileBuffer = bytes.Buffer{}
	bytesRead, readBytesError := sourceFile.Read(fileBytes)
	for bytesRead > 0 {
		if readBytesError != nil && readBytesError.Error() != "EOF" {
			filesystemService.logger.Errorf("Failed to read bytes from source file -> %s: %s", destPath, readBytesError.Error())
			return readBytesError
		}
		for i := range bytesRead {
			fileBuffer.WriteByte(fileBytes[i])
		}
		fileBytes = make([]byte, 4096)
		bytesRead, readBytesError = sourceFile.Read(fileBytes)
	}
	osFile, createFileError := os.Create(destPath)
	if createFileError != nil && strings.Contains(createFileError.Error(), "is a directory") {
		osFile, createFileError = os.Create(fmt.Sprintf("%s/%s", destPath, sourceFileName))
	} else if createFileError != nil {
		filesystemService.logger.Errorf("Failed to create file to copy source to %s: %s", destPath, createFileError.Error())
		return createFileError
	}

	bytesWritten, writeFileError := osFile.Write(fileBuffer.Bytes())

	if writeFileError != nil {
		filesystemService.logger.Errorf("Failed to write source file to destination %s: %s", destPath, writeFileError.Error())
		return writeFileError
	}

	if bytesWritten != fileBuffer.Len() {
		filesystemService.logger.Errorf("Bytes written %d to %s does not match the number of bytes read %d from the source file", bytesWritten, destPath, fileBuffer.Len())
		return errors.New(fmt.Sprintf("Bytes written %d to %s does not match the number of bytes read %d from the source file", bytesWritten, destPath, fileBuffer.Len()))
	}

	return nil
}

func (filesystemService *FileSystemServiceImpl) getSingleFileInfo(system filesystem.FileSystem, sourcePath string, destPath string) (os.FileInfo, *string, error) {
	pathParts := strings.Split(destPath, "/")
	localDestPath := ""
	for i, part := range pathParts {
		if i != len(pathParts)-1 {
			localDestPath = fmt.Sprintf("%s/%s", localDestPath, part)
		}
	}
	localDestPath = strings.Replace(localDestPath, "//", "/", -1)
	sourceParts := strings.Split(sourcePath, "/")
	sourceDir := "/"
	if len(sourceParts) > 1 {
		for _, pathPart := range sourceParts {
			sourceDir = fmt.Sprintf("%s/%s", sourceDir, pathPart)
		}
	}
	sourceDir = strings.Replace(sourceDir, "//", "/", -1)
	sourceDirectoryFiles, statFileError := system.ReadDir(sourceDir)

	if statFileError != nil {
		filesystemService.logger.Errorf("Source directory %s could not be found: %s", sourceDir, statFileError.Error())
		return nil, nil, statFileError
	}

	var sourceFile os.FileInfo = nil

	for _, fileInfo := range sourceDirectoryFiles {
		strippedFilePath := fmt.Sprintf("%s%s", strings.Replace(sourceDir, "/", "", 1), fileInfo.Name())
		if fmt.Sprintf("%s/%s", sourceDir, fileInfo.Name()) == sourcePath || strippedFilePath == sourcePath {
			sourceFile = fileInfo
			break
		}
	}
	if sourceFile == nil {
		filesystemService.logger.Errorf("The requested file %s could not be matched", sourcePath)
		return nil, nil, errors.New(fmt.Sprintf("file %s could not be found", sourcePath))
	}
	return sourceFile, &sourceDir, nil
}
