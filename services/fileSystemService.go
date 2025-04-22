package services

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"syscall"
	"zs-vm-agent/clients"
)

type FileSystemService interface {
	initialize(logger *logrus.Logger, osClient clients.OsClient, userClient clients.UserClient)
	CreateRootFsDirectory(path string, recursive bool, permissions int) error
	SetRootFsOwner(path string, owner string, recursive bool) error
	SetRootFsPermissions(path string, permissions int, recursive bool) error
	GetFilesystem(diskWrapper clients.DiskWrapper, partition int) (clients.FileSystemWrapper, error)
	GetBlockFilesystem(devicePath string) (clients.FileSystemWrapper, error)
	CopyFilesToRootFs(sourceFilesystem clients.FileSystemWrapper, sourcePath string, destPath string, recursive bool) error
	ReadFileContents(path string) ([]byte, error)
}

type FileSystemServiceImpl struct {
	logger     *logrus.Logger
	osClient   clients.OsClient
	userClient clients.UserClient
}

func (filesystemService *FileSystemServiceImpl) initialize(logger *logrus.Logger, osClient clients.OsClient, userClient clients.UserClient) {
	filesystemService.logger = logger
	filesystemService.osClient = osClient
	filesystemService.userClient = userClient
}

func (filesystemService *FileSystemServiceImpl) CreateRootFsDirectory(path string, recursive bool, permissions int) error {
	var pathParts []string = strings.Split(path, "/")
	var currentPath = ""
	for _, pathPart := range pathParts {
		currentPath = strings.Replace(fmt.Sprintf("%s%s/", currentPath, pathPart), "//", "/", -1)
		_, readDirectoryError := filesystemService.osClient.StatFile(currentPath)

		if readDirectoryError != nil && readDirectoryError.Error() == fmt.Sprintf("stat %s: no such file or directory", currentPath) && (recursive || strings.Contains(currentPath, path)) {
			createDirectoryError := filesystemService.osClient.Mkdir(currentPath, permissions)
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

func (filesystemService *FileSystemServiceImpl) SetRootFsPermissions(path string, permissions int, recursive bool) error {
	fileInfo, getDirectoryInfoError := filesystemService.osClient.StatFile(path)

	if getDirectoryInfoError != nil {
		filesystemService.logger.Errorf("Failed to retrieve directory %s before updating permissions: %s", path, getDirectoryInfoError.Error())
		return getDirectoryInfoError
	}

	if fileInfo.IsDir() && recursive {
		directoryEntries, readDirectoryError := filesystemService.osClient.ReadDir(path)
		if readDirectoryError != nil {
			filesystemService.logger.Errorf("Failed to read directory for recursive ownership change %s: %s", path, readDirectoryError)
			return readDirectoryError
		}
		for _, entry := range directoryEntries {
			setOwnerError := filesystemService.SetRootFsPermissions(strings.Replace(fmt.Sprintf("%s/%s", path, entry.Name()), "//", "/", -1), permissions, recursive)
			if setOwnerError != nil {
				return setOwnerError
			}
		}
	}
	setPermissionsError := filesystemService.osClient.SetPermissions(path, permissions)

	if setPermissionsError != nil {
		filesystemService.logger.Errorf("Failed to set permissions on %s: %s", path, setPermissionsError.Error())
		return setPermissionsError
	}

	return nil
}

func (filesystemService *FileSystemServiceImpl) GetFilesystem(diskWrapper clients.DiskWrapper, partition int) (clients.FileSystemWrapper, error) {
	if diskWrapper == nil {
		filesystemService.logger.Error("Disk provided was nil")
		return nil, errors.New("cannot get filesystem from nil disk pointer")
	}

	fileSystem, getFileSystemError := diskWrapper.GetFileSystem(partition)

	if getFileSystemError != nil {
		filesystemService.logger.Errorf("Failed to retrieve file system from disk at partition %d", partition)
		return nil, getFileSystemError
	}

	_, readDirError := fileSystem.ReadDir("/")

	if readDirError != nil {
		filesystemService.logger.Errorf("Failed to read filesystem: %s", readDirError.Error())
		return nil, readDirError
	}

	return fileSystem, nil
}

func (filesystemService *FileSystemServiceImpl) GetBlockFilesystem(devicePath string) (clients.FileSystemWrapper, error) {
	blockDevice, getDeviceError := filesystemService.osClient.OpenDisk(devicePath)

	if getDeviceError != nil {
		filesystemService.logger.Errorf("Failed to retrieve block device at specified path %s: %s", devicePath, getDeviceError.Error())
		return nil, getDeviceError
	}

	blockFilesystem, getBlockFilesystemError := blockDevice.GetFileSystem(0) //no partition table so 0th partition

	if getBlockFilesystemError != nil {
		filesystemService.logger.Errorf("Failed to retrieve filesystem from block device %s: %s", devicePath, getBlockFilesystemError.Error())
		return nil, getBlockFilesystemError
	}

	filesystemService.logger.Debugf("Successfully retrieved filesystem from block device %s", devicePath)
	return blockFilesystem, nil
}

func (filesystemService *FileSystemServiceImpl) CopyFilesToRootFs(sourceFilesystem clients.FileSystemWrapper, sourcePath string, destPath string, recursive bool) error {
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
		localSourcePath, _ = strings.CutPrefix(*newDestPath, "/")
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

func (filesystemService *FileSystemServiceImpl) attemptReadDir(sourceFilesystem clients.FileSystemWrapper, sourcePath string) ([]os.FileInfo, error) {
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
	osFile, createFileError := filesystemService.osClient.CreateFile(destPath)
	if createFileError != nil && strings.Contains(createFileError.Error(), "is a directory") {
		osFile, createFileError = os.Create(fmt.Sprintf("%s/%s", destPath, sourceFileName))
	} else if createFileError != nil {
		filesystemService.logger.Errorf("Failed to create file to copy source to %s: %s", destPath, createFileError.Error())
		return createFileError
	}

	if osFile == nil {
		filesystemService.logger.Errorf("Failed to retrieve file to copy source to: %s, file was nil", destPath)
		return errors.New(fmt.Sprintf("Failed to retrieve file to copy source to: %s, file was nil", destPath))
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

func (filesystemService *FileSystemServiceImpl) getSingleFileInfo(system clients.FileSystemWrapper, sourcePath string, destPath string) (os.FileInfo, *string, error) {
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

func (filesystemService *FileSystemServiceImpl) SetRootFsOwner(path string, owner string, recursive bool) error {
	directoryInfo, getDirectoryInfoError := filesystemService.osClient.StatFile(path)

	if getDirectoryInfoError != nil {
		filesystemService.logger.Errorf("Failed to retrieve directory %s before updating ownership: %s", path, getDirectoryInfoError.Error())
		return getDirectoryInfoError
	}

	if directoryInfo.IsDir() && recursive {
		directoryEntries, readDirectoryError := filesystemService.osClient.ReadDir(path)
		if readDirectoryError != nil {
			filesystemService.logger.Errorf("Failed to read directory for recursive ownership change %s: %s", path, readDirectoryError)
			return readDirectoryError
		}
		for _, entry := range directoryEntries {
			setOwnerError := filesystemService.SetRootFsOwner(strings.Replace(fmt.Sprintf("%s/%s", path, entry.Name()), "//", "/", -1), owner, recursive)
			if setOwnerError != nil {
				return setOwnerError
			}
		}
	}

	fileSys := directoryInfo.Sys().(*syscall.Stat_t)

	directoryGuid := fmt.Sprint(fileSys.Gid)

	guid, guidConversionError := strconv.Atoi(directoryGuid)

	if guidConversionError != nil {
		filesystemService.logger.Errorf("Failed to convert guid string %s to integer: %s", directoryGuid, guidConversionError)
		return guidConversionError
	}

	ownerUser, getUserUidError := filesystemService.userClient.GetUserByName(owner)

	if getUserUidError != nil {
		filesystemService.logger.Errorf("Failed to retrieve UID for user %s: %s", owner, getUserUidError.Error())
		return getUserUidError
	}

	uid, uidConversionError := strconv.Atoi(ownerUser.Uid)

	if uidConversionError != nil {
		filesystemService.logger.Errorf("Failed to convert uid string %s to integer: %s", ownerUser.Uid, uidConversionError)
		return uidConversionError
	}

	setOwnerError := filesystemService.osClient.SetOwner(path, uid, guid)
	if setOwnerError != nil {
		logrus.Errorf("Failed to set owner for %s: %s", path, setOwnerError)
		return setOwnerError
	}
	return nil
}

func (filesystemService *FileSystemServiceImpl) ReadFileContents(path string) ([]byte, error) {
	filesystemService.logger.Debugf("Reading File at %s", path)
	file, getFileError := filesystemService.osClient.OpenFile(path)
	if getFileError != nil {
		filesystemService.logger.Errorf("Failed to open file at %s: %s", path, getFileError.Error())
		return nil, getFileError
	}
	byteSlice := make([]byte, 4096)
	var fileBuffer bytes.Buffer
	bytesRead, readError := file.Read(byteSlice)

	if bytesRead == 0 {
		filesystemService.logger.Error("No data read empty file")
	}

	for bytesRead > 0 {
		if readError != nil {
			filesystemService.logger.Errorf("Error occured while reading from file %s: %s", path, readError.Error())
		}

		for i := range bytesRead {
			fileBuffer.WriteByte(byteSlice[i])
		}
		bytesRead, readError = file.Read(byteSlice)
	}

	return fileBuffer.Bytes(), nil
}
