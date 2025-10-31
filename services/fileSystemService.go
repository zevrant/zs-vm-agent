package services

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"zs-vm-agent/clients"

	"github.com/moby/sys/mount"
	"github.com/sirupsen/logrus"
)

type FileSystemService interface {
	initialize(logger *logrus.Logger, osClient clients.OsClient, userClient clients.UserClient)
	CreateRootFsDirectory(path string, recursive bool, permissions int) error
	SetRootFsOwner(path string, owner string, recursive bool) error
	SetRootFsPermissions(path string, permissions int, recursive bool) error
	GetFilesystem(diskWrapper clients.DiskWrapper, partition int) (clients.FileSystemWrapper, error)
	GetBlockFilesystem(devicePath string) (clients.FileSystemWrapper, error)
	CopyFilesToRootFs(sourceFilesystem clients.FileSystemWrapper, sourcePath string, destPath string, recursive bool) error
	CopySingleFileToRootFs(sourceFilesystem clients.FileSystemWrapper, sourceFilePath string, destPath string) error
	ReadFileContents(path string) ([]byte, error)
	ReadFileContentsFromFilesystem(fs clients.FileSystemWrapper, path string) ([]byte, error)
	MountFilesystem(deviceLocation string, mountLocation string) error
	CreateXfsFileSystem(partitionPath string) error
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
	var pathParts = strings.Split(path, "/")
	var currentPath = ""
	for _, pathPart := range pathParts {
		currentPath = strings.ReplaceAll(fmt.Sprintf("%s%s/", currentPath, pathPart), "//", "/")
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
			setOwnerError := filesystemService.SetRootFsPermissions(strings.ReplaceAll(fmt.Sprintf("%s/%s", path, entry.Name()), "//", "/"), permissions, recursive)
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
	filesystemService.logger.Infof("Copying %s to %s", sourcePath, destPath)
	fileInfos, readSourceError := filesystemService.attemptReadDir(sourceFilesystem, sourcePath)
	isSourceADir := true
	if readSourceError != nil && readSourceError.Error() != fmt.Sprintf("error reading directory %s: cannot create directory at %s since it is a file", sourcePath, sourcePath) {
		filesystemService.logger.Debugf("Failed to read source file at %s, cannot continue copy operation", sourcePath)
		return readSourceError
	}

	if fileInfos == nil {
		isSourceADir = false
		fileInfo, _, getFileInfoError := filesystemService.getSingleFileInfo(sourceFilesystem, sourcePath, destPath)
		if getFileInfoError != nil {
			return getFileInfoError
		}
		fileInfos = []os.FileInfo{fileInfo}
	}
	for _, fileInfo := range fileInfos {
		fileName := fileInfo.Name()
		filesystemService.logger.Debugf("Processing %s", fileName)
		filesystemService.logger.Debugf("Is Directory? %v", fileInfo.IsDir())
		//filesystemService.logger.Debugf("Processing %s", fileName)
		//filesystemService.logger.Debugf("Processing %s", fileName)

		if fileInfo.IsDir() && fileName != "." && fileName != ".." {
			filesystemService.logger.Debugf("Starting copy for directory %s", fileName)
			copyError := filesystemService.CopyFilesToRootFs(sourceFilesystem, fmt.Sprintf("%s/%s", sourcePath, fileName), destPath, recursive)
			if copyError != nil {
				return copyError
			}

		} else if fileName != "." && fileName != ".." {
			filesystemService.logger.Debugf("Starting copy for file %s", fileName)
			filePath := sourcePath
			if isSourceADir {
				filePath = sourcePath + "/" + fileName
			}
			filesystemService.logger.Debugf("Copying filepath %s", filePath)
			copyFileError := filesystemService.CopySingleFileToRootFs(sourceFilesystem, filePath, destPath)
			if copyFileError != nil {
				return copyFileError
			}
		}
		filesystemService.logger.Debugf("Copied %s to %s", sourcePath, destPath)
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

func (filesystemService *FileSystemServiceImpl) CopySingleFileToRootFs(sourceFilesystem clients.FileSystemWrapper, sourceFilePath string, destPath string) error {
	sourceFile, readSourceError := sourceFilesystem.OpenFile(sourceFilePath, 0)
	if readSourceError != nil {
		filesystemService.logger.Errorf("Failed to open file %s: %s", sourceFilePath, readSourceError.Error())
		return readSourceError
	}
	filesystemService.logger.Debugf("File %s opened", sourceFilePath)
	var fileBytes = make([]byte, 4096)
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
		filePathParts := strings.Split(sourceFilePath, "/")
		osFile, createFileError = filesystemService.osClient.CreateFile(fmt.Sprintf("%s/%s", destPath, filePathParts[len(filePathParts)-1]))
	}
	if createFileError != nil {
		filesystemService.logger.Errorf("Failed to create file to copy source to %s: %s", destPath, createFileError.Error())
		return createFileError
	}

	if osFile == nil {
		filesystemService.logger.Errorf("Failed to retrieve file to copy source %s to: %s, file was nil", sourceFilePath, destPath)
		return fmt.Errorf("failed to retrieve file to copy source %s to: %s, file was nil", sourceFilePath, destPath)
	}

	bytesWritten, writeFileError := osFile.Write(fileBuffer.Bytes())

	if writeFileError != nil {
		filesystemService.logger.Errorf("Failed to write source file to destination %s: %s", destPath, writeFileError.Error())
		return writeFileError
	}

	if bytesWritten != fileBuffer.Len() {
		filesystemService.logger.Errorf("Bytes written %d to %s does not match the number of bytes read %d from the source file", bytesWritten, destPath, fileBuffer.Len())
		return fmt.Errorf("bytes written %d to %s does not match the number of bytes read %d from the source file", bytesWritten, destPath, fileBuffer.Len())
	}

	return nil
}

func (filesystemService *FileSystemServiceImpl) getSingleFileInfo(system clients.FileSystemWrapper, sourcePath string, destPath string) (os.FileInfo, *string, error) {
	filesystemService.logger.Debugf("Getting file info for %s", sourcePath)
	sourceParts := strings.Split(sourcePath, "/")
	sourceDir := "/"
	if len(sourceParts) > 1 {
		for _, pathPart := range sourceParts {
			sourceDir = fmt.Sprintf("%s/%s", sourceDir, pathPart)
		}
	}
	filesystemService.logger.Debugf("File info for source dir is %s", sourceDir)
	for strings.Contains(sourceDir, "//") {
		sourceDir = strings.ReplaceAll(sourceDir, "//", "/")
	}
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
		return nil, nil, fmt.Errorf("file %s could not be found", sourcePath)
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
			setOwnerError := filesystemService.SetRootFsOwner(strings.ReplaceAll(fmt.Sprintf("%s/%s", path, entry.Name()), "//", "/"), owner, recursive)
			if setOwnerError != nil {
				return setOwnerError
			}
		}
	}

	fileSys := directoryInfo.Sys().(*syscall.Stat_t)

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

	setOwnerError := filesystemService.osClient.SetOwner(path, uid, int(fileSys.Gid))
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

func (filesystemService *FileSystemServiceImpl) ReadFileContentsFromFilesystem(fs clients.FileSystemWrapper, path string) ([]byte, error) {
	file, openFileError := fs.OpenFile(path, 0)

	if openFileError != nil {
		filesystemService.logger.Errorf("Failed to open file %s on file system %s: %s", path, fs.GetFilesystemLabel(), openFileError.Error())
		return nil, openFileError
	}

	readBuffer := make([]byte, 4096)
	byteBuffer := bytes.NewBuffer(make([]byte, 0))
	bytesRead, readError := file.Read(readBuffer)

	for bytesRead > 0 {
		if readError != nil && readError.Error() != "EOF" {
			filesystemService.logger.Errorf("Failed to read from file %s on file system %s: %s", path, fs.GetFilesystemLabel(), readError.Error())
			return nil, readError
		}
		for i := range bytesRead {
			byteBuffer.WriteByte(readBuffer[i])
		}
		bytesRead, readError = file.Read(readBuffer)
	}

	return byteBuffer.Bytes(), nil
}

func (filesystemService *FileSystemServiceImpl) MountFilesystem(deviceLocation string, mountLocation string) error {
	mountError := mount.Mount(deviceLocation, mountLocation, "xfs", "")
	if mountError != nil {
		filesystemService.logger.Errorf("Failed to mount device %s at %s: %s", deviceLocation, mountLocation, mountError.Error())
		return mountError
	}
	return nil
}

func (filesystemService *FileSystemServiceImpl) CreateXfsFileSystem(partitionPath string) error {
	command := exec.Command("/usr/sbin/mkfs.xfs", partitionPath)

	outputText, commandExecutionError := command.CombinedOutput()

	for _, line := range strings.Split(string(outputText), "\n") {
		filesystemService.logger.Info(line)
	}

	if commandExecutionError != nil {
		filesystemService.logger.Errorf("Failed to create filesystem at %s: %s", partitionPath, commandExecutionError.Error())
		commandExecutionError = errors.New(commandExecutionError.Error() + " " + string(outputText))
		return commandExecutionError
	}

	return nil
}
