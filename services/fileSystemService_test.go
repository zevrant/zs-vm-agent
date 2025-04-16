package services

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"zs-vm-agent/clients"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestFileSystemServiceImpl_CreateRootFsDirectory_alreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockOsClient := clients.NewMockOsClient(ctrl)
	mockUserClient := clients.NewMockUserClient(ctrl)
	testPath := "testPath"

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, mockOsClient, mockUserClient)
	mockOsClient.
		EXPECT().
		StatFile(gomock.Eq(testPath+"/")).
		Times(1).
		Return(nil, nil)

	createRootFsDirError := testFilesystemService.CreateRootFsDirectory(testPath, false, 0700)

	assert.Nil(t, createRootFsDirError)
}

func TestFileSystemServiceImpl_CreateRootFsDirectory_readFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockOsClient := clients.NewMockOsClient(ctrl)
	mockUserClient := clients.NewMockUserClient(ctrl)
	testPath := "testPath"

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, mockOsClient, mockUserClient)

	testError := errors.New("failed to read directory")

	mockOsClient.
		EXPECT().
		StatFile(gomock.Eq(testPath+"/")).
		Times(1).
		Return(nil, testError)

	createRootFsDirError := testFilesystemService.CreateRootFsDirectory(testPath, false, 0700)

	assert.ErrorIs(t, createRootFsDirError, testError)
}

func TestFileSystemServiceImpl_CreateRootFsDirectory_doesntExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOsClient := clients.NewMockOsClient(ctrl)
	mockUserClient := clients.NewMockUserClient(ctrl)

	testPath := "testPath"

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, mockOsClient, mockUserClient)
	mockOsClient.
		EXPECT().
		StatFile(gomock.Eq(testPath+"/")).
		Times(1).
		Return(nil, errors.New(fmt.Sprintf("stat %s/: no such file or directory", testPath)))

	mockOsClient.
		EXPECT().
		Mkdir(gomock.Eq(fmt.Sprintf("%s/", testPath)), gomock.Eq(0700)).
		Times(1).
		Return(nil)

	createRootFsDirError := testFilesystemService.CreateRootFsDirectory(testPath, false, 0700)

	assert.Nil(t, createRootFsDirError)
}

func TestFileSystemServiceImpl_CreateRootFsDirectory_doesntExist_createFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOsClient := clients.NewMockOsClient(ctrl)
	mockUserClient := clients.NewMockUserClient(ctrl)

	testPath := "testPath"

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, mockOsClient, mockUserClient)
	mockOsClient.
		EXPECT().
		StatFile(gomock.Eq(testPath+"/")).
		Times(1).
		Return(nil, fmt.Errorf("stat %s/: no such file or directory", testPath))

	testError := errors.New("i failed")

	mockOsClient.
		EXPECT().
		Mkdir(gomock.Eq(fmt.Sprintf("%s/", testPath)), gomock.Eq(0755)).
		Times(1).
		Return(testError)

	createRootFsDirError := testFilesystemService.CreateRootFsDirectory(testPath, false, 0755)

	assert.ErrorIs(t, createRootFsDirError, testError)
}

func TestFileSystemServiceImpl_GetFilesystem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPartitionNumber := 1

	mockFileSystemWrapper := clients.NewMockFileSystemWrapper(ctrl)
	mockFileSystemWrapper.EXPECT().
		ReadDir(gomock.Any()).
		Times(1).
		Return(nil, nil)

	mockDiskWrapper := clients.NewMockDiskWrapper(ctrl)
	mockDiskWrapper.EXPECT().GetFileSystem(gomock.Eq(testPartitionNumber)).Return(mockFileSystemWrapper, nil)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, nil, nil)

	retrievedFilesystem, getFilesystemError := testFilesystemService.GetFilesystem(mockDiskWrapper, testPartitionNumber)

	assert.Nil(t, getFilesystemError)
	assert.Equal(t, mockFileSystemWrapper, retrievedFilesystem)
}

func TestFileSystemServiceImpl_GetFilesystem_UnableToRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPartitionNumber := 1

	mockFileSystemWrapper := clients.NewMockFileSystemWrapper(ctrl)
	readDirError := errors.New("test error")
	mockFileSystemWrapper.EXPECT().
		ReadDir(gomock.Any()).
		Times(1).
		Return(nil, readDirError)

	mockDiskWrapper := clients.NewMockDiskWrapper(ctrl)
	mockDiskWrapper.EXPECT().GetFileSystem(gomock.Eq(testPartitionNumber)).Return(mockFileSystemWrapper, nil)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, nil, nil)

	retrievedFilesystem, getFilesystemError := testFilesystemService.GetFilesystem(mockDiskWrapper, testPartitionNumber)

	assert.NotNil(t, getFilesystemError)
	assert.Equal(t, readDirError, getFilesystemError)
	assert.Nil(t, retrievedFilesystem)
}

func TestFileSystemServiceImpl_GetFilesystem_FailedToGetFileSystem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPartitionNumber := 1

	mockFileSystemWrapper := clients.NewMockFileSystemWrapper(ctrl)
	mockFileSystemWrapper.EXPECT().
		ReadDir(gomock.Any()).
		Times(0).
		Return(nil, nil)

	getFileSystemTestError := errors.New("test error")

	mockFileSystem := NewMockFileSystem(ctrl)
	mockFileSystem.EXPECT().ReadDir(gomock.Eq("/")).Times(0)

	mockDiskWrapper := clients.NewMockDiskWrapper(ctrl)
	mockDiskWrapper.EXPECT().GetFileSystem(gomock.Eq(testPartitionNumber)).Return(nil, getFileSystemTestError)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, nil, nil)

	retrievedFilesystem, getFilesystemError := testFilesystemService.GetFilesystem(mockDiskWrapper, testPartitionNumber)

	assert.NotNil(t, getFilesystemError)
	assert.Equal(t, getFileSystemTestError, getFilesystemError)
	assert.Nil(t, retrievedFilesystem)
}

func TestFileSystemServiceImpl_GetFilesystem_NilDisk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPartitionNumber := 1

	mockFileSystemWrapper := clients.NewMockFileSystemWrapper(ctrl)

	mockFileSystemWrapper.EXPECT().
		ReadDir(gomock.Any()).
		Times(0).
		Return(nil, nil)

	mockFileSystem := NewMockFileSystem(ctrl)
	mockFileSystem.EXPECT().ReadDir(gomock.Eq("/")).Times(0)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, nil, nil)

	retrievedFilesystem, getFilesystemError := testFilesystemService.GetFilesystem(nil, testPartitionNumber)

	assert.NotNil(t, getFilesystemError)
	assert.Equal(t, "cannot get filesystem from nil disk pointer", getFilesystemError.Error())
	assert.Nil(t, retrievedFilesystem)
}

func TestFileSystemServiceImpl_GetBlockFilesystem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testPath := "testPath"

	mockFileSystemWrapper := clients.NewMockFileSystemWrapper(ctrl)
	mockUserClient := clients.NewMockUserClient(ctrl)

	mockDiskWrapper := clients.NewMockDiskWrapper(ctrl)
	mockDiskWrapper.EXPECT().GetFileSystem(gomock.Eq(0)).Times(1).Return(mockFileSystemWrapper, nil)

	mockOsClient := clients.NewMockOsClient(ctrl)
	mockOsClient.EXPECT().OpenDisk(gomock.Eq(testPath)).Times(1).Return(mockDiskWrapper, nil)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, mockOsClient, mockUserClient)

	retrievedFilesystem, getFilesystemError := testFilesystemService.GetBlockFilesystem(testPath)

	assert.Nil(t, getFilesystemError)
	assert.NotNil(t, retrievedFilesystem)
	assert.Equal(t, mockFileSystemWrapper, retrievedFilesystem)

}

func TestFileSystemServiceImpl_GetBlockFilesystem_FailedGetFilesystem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testPath := "testPath"
	testError := errors.New("i'm a test error")

	mockUserClient := clients.NewMockUserClient(ctrl)

	mockDiskWrapper := clients.NewMockDiskWrapper(ctrl)
	mockDiskWrapper.EXPECT().GetFileSystem(gomock.Eq(0)).Times(1).Return(nil, testError)

	mockOsClient := clients.NewMockOsClient(ctrl)
	mockOsClient.EXPECT().OpenDisk(gomock.Eq(testPath)).Times(1).Return(mockDiskWrapper, nil)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, mockOsClient, mockUserClient)

	retrievedFilesystem, getFilesystemError := testFilesystemService.GetBlockFilesystem(testPath)

	assert.Nil(t, retrievedFilesystem)
	assert.NotNil(t, getFilesystemError)
	assert.Equal(t, testError, getFilesystemError)

}

func TestFileSystemServiceImpl_GetBlockFilesystem_FailedGetDisk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testPath := "testPath"
	testError := errors.New("i'm a test error")

	mockUserClient := clients.NewMockUserClient(ctrl)

	mockDiskWrapper := clients.NewMockDiskWrapper(ctrl)
	mockDiskWrapper.EXPECT().GetFileSystem(gomock.Any()).Times(0)

	mockOsClient := clients.NewMockOsClient(ctrl)
	mockOsClient.EXPECT().OpenDisk(gomock.Eq(testPath)).Times(1).Return(mockDiskWrapper, testError)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, mockOsClient, mockUserClient)

	retrievedFilesystem, getFilesystemError := testFilesystemService.GetBlockFilesystem(testPath)

	assert.Nil(t, retrievedFilesystem)
	assert.NotNil(t, getFilesystemError)
	assert.Equal(t, testError, getFilesystemError)

}

func TestFileSystemServiceImpl_CopyFilesToRootFs_CopySingleDirectory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPath := "testPath"
	testFile := "testFile.txt"

	testBytes := []byte("testBytes")

	mockDestFile := clients.NewMockFileWrapper(ctrl)
	mockDestFile.EXPECT().Write(gomock.Eq(testBytes)).Times(1).Return(len(testBytes), nil)

	osClient := clients.NewMockOsClient(ctrl)
	osClient.EXPECT().CreateFile("destPath").Times(1).Return(mockDestFile, nil)
	mockUserClient := clients.NewMockUserClient(ctrl)

	mockFileInfo := NewMockFileInfo(ctrl)
	mockFileInfo.EXPECT().IsDir().Times(1).Return(false)
	mockFileInfo.EXPECT().Name().Times(4).Return(testFile)

	mockSourceFile := clients.NewMockFileWrapper(ctrl)
	i := 0
	mockSourceFile.EXPECT().Read(gomock.AssignableToTypeOf([]uint8{})).Times(2).Return(len(testBytes), nil).DoAndReturn(func(fileBytes []uint8) (int, error) {
		if i == 0 {
			i = 1
			for index := range len(testBytes) {
				fileBytes[index] = testBytes[index]
			}
			return len(testBytes), nil
		}
		return 0, nil
	})

	mockFileSystem := clients.NewMockFileSystemWrapper(ctrl)
	mockFileSystem.EXPECT().ReadDir(gomock.Eq(testPath)).Times(1).Return([]os.FileInfo{mockFileInfo}, nil)
	mockFileSystem.EXPECT().OpenFile(gomock.Eq(fmt.Sprintf("%s/%s", testPath, testFile)), gomock.Eq(0)).Return(mockSourceFile, nil)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, osClient, mockUserClient)

	getFilesystemError := testFilesystemService.CopyFilesToRootFs(mockFileSystem, testPath, "destPath", false)

	assert.Nil(t, getFilesystemError)

}

func TestFileSystemServiceImpl_CopyFilesToRootFs_CopySingleDirectory_ErrorCopyingFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPath := "testPath"
	testFile := "testFile.txt"

	testBytes := []byte("testBytes")

	mockDestFile := clients.NewMockFileWrapper(ctrl)
	mockDestFile.EXPECT().Write(gomock.Eq(testBytes)).Times(1).Return(len(testBytes)-5, nil)

	osClient := clients.NewMockOsClient(ctrl)
	osClient.EXPECT().CreateFile("destPath").Times(1).Return(mockDestFile, nil)

	mockFileInfo := NewMockFileInfo(ctrl)
	mockFileInfo.EXPECT().IsDir().Times(1).Return(false)
	mockFileInfo.EXPECT().Name().Times(4).Return(testFile)

	mockSourceFile := clients.NewMockFileWrapper(ctrl)
	i := 0
	mockSourceFile.EXPECT().Read(gomock.AssignableToTypeOf([]uint8{})).Times(2).Return(len(testBytes), nil).DoAndReturn(func(fileBytes []uint8) (int, error) {
		if i == 0 {
			i = 1
			for index := range len(testBytes) {
				fileBytes[index] = testBytes[index]
			}
			return len(testBytes), nil
		}
		return 0, nil
	})
	mockUserClient := clients.NewMockUserClient(ctrl)

	mockFileSystem := clients.NewMockFileSystemWrapper(ctrl)
	mockFileSystem.EXPECT().ReadDir(gomock.Eq(testPath)).Times(1).Return([]os.FileInfo{mockFileInfo}, nil)
	mockFileSystem.EXPECT().OpenFile(gomock.Eq(fmt.Sprintf("%s/%s", testPath, testFile)), gomock.Eq(0)).Return(mockSourceFile, nil)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, osClient, mockUserClient)

	getFilesystemError := testFilesystemService.CopyFilesToRootFs(mockFileSystem, testPath, "destPath", false)

	assert.NotNil(t, getFilesystemError)
	assert.ErrorContainsf(t, getFilesystemError, "does not match the number of bytes read", "Filesystem error should mention incorrect number of bytes being read & written")

}

func TestFileSystemServiceImpl_CopyFilesToRootFs_CopySingleDirectory_ErrorOpeningFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPath := "testPath"
	testFile := "testFile.txt"

	testBytes := []byte("testBytes")

	mockDestFile := clients.NewMockFileWrapper(ctrl)
	mockDestFile.EXPECT().Write(gomock.Eq(testBytes)).Times(0)

	mockUserClient := clients.NewMockUserClient(ctrl)

	osClient := clients.NewMockOsClient(ctrl)
	osClient.EXPECT().CreateFile("destPath").Times(0)

	mockFileInfo := NewMockFileInfo(ctrl)
	mockFileInfo.EXPECT().IsDir().Times(1).Return(false)
	mockFileInfo.EXPECT().Name().Times(3).Return(testFile)

	mockSourceFile := clients.NewMockFileWrapper(ctrl)
	mockSourceFile.EXPECT().Read(gomock.AssignableToTypeOf([]uint8{})).Times(0)

	mockFileSystem := clients.NewMockFileSystemWrapper(ctrl)
	mockFileSystem.EXPECT().ReadDir(gomock.Eq(testPath)).Times(1).Return([]os.FileInfo{mockFileInfo}, nil)
	mockFileSystem.EXPECT().OpenFile(gomock.Eq(fmt.Sprintf("%s/%s", testPath, testFile)), gomock.Eq(0)).Return(nil, errors.New("i failed to open the file"))

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, osClient, mockUserClient)

	getFilesystemError := testFilesystemService.CopyFilesToRootFs(mockFileSystem, testPath, "destPath", false)

	assert.NotNil(t, getFilesystemError)
	assert.ErrorContainsf(t, getFilesystemError, "i failed to open the file", "test error message about opening a file is not correct")

}

func TestFileSystemServiceImpl_CopyFilesToRootFs_CopySingleDirectory_ErrorReadingSourceDirectory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testPath := "testPath"
	testFile := "testFile.txt"

	testBytes := []byte("testBytes")

	mockDestFile := clients.NewMockFileWrapper(ctrl)
	mockDestFile.EXPECT().Write(gomock.Eq(testBytes)).Times(0)

	osClient := clients.NewMockOsClient(ctrl)
	osClient.EXPECT().CreateFile("destPath").Times(0)

	mockFileInfo := NewMockFileInfo(ctrl)
	mockFileInfo.EXPECT().IsDir().Times(0).Return(false)
	mockFileInfo.EXPECT().Name().Times(0).Return(testFile)

	mockSourceFile := clients.NewMockFileWrapper(ctrl)
	mockSourceFile.EXPECT().Read(gomock.Any()).Times(0)

	errorMessage := "test failure"

	mockFileSystem := clients.NewMockFileSystemWrapper(ctrl)
	mockFileSystem.EXPECT().ReadDir(gomock.Eq(testPath)).Times(1).Return(nil, errors.New(errorMessage))
	mockFileSystem.EXPECT().OpenFile(gomock.Any(), gomock.Any()).Times(0)

	mockUserClient := clients.NewMockUserClient(ctrl)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, osClient, mockUserClient)

	getFilesystemError := testFilesystemService.CopyFilesToRootFs(mockFileSystem, testPath, "destPath", false)

	assert.NotNil(t, getFilesystemError)
	assert.ErrorContainsf(t, getFilesystemError, errorMessage, "test error message about reading a directory is not correct")

}

func TestFileSystemServiceImpl_CopyFilesToRootFs_CopySingleFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testFile := "testFile.txt"

	testBytes := []byte("testBytes")

	mockDestFile := clients.NewMockFileWrapper(ctrl)
	mockDestFile.EXPECT().Write(gomock.Eq(testBytes)).Times(1).Return(len(testBytes), nil)

	osClient := clients.NewMockOsClient(ctrl)
	osClient.EXPECT().CreateFile("destPath").Times(1).Return(mockDestFile, nil)

	mockSourceFile := clients.NewMockFileWrapper(ctrl)
	i := 0
	mockSourceFile.EXPECT().Read(gomock.AssignableToTypeOf([]uint8{})).Times(2).Return(len(testBytes), nil).DoAndReturn(func(fileBytes []uint8) (int, error) {
		if i == 0 {
			i = 1
			for index := range len(testBytes) {
				fileBytes[index] = testBytes[index]
			}
			return len(testBytes), nil
		}
		return 0, nil
	})

	singleFileInfo := NewMockFileInfo(ctrl)
	singleFileInfo.EXPECT().Name().Times(2).Return(testFile)
	singleFileInfo.EXPECT().IsDir().Times(1).Return(false)
	singleFileInfo.EXPECT().Name().Times(4).Return(testFile)
	mockFileSystem := clients.NewMockFileSystemWrapper(ctrl)
	mockFileSystem.EXPECT().ReadDir(gomock.Any()).Times(2).DoAndReturn(func(sourceDir string) ([]os.FileInfo, error) {
		if sourceDir == "/" {
			return []os.FileInfo{singleFileInfo}, nil
		}
		return nil, nil
	})
	mockFileSystem.EXPECT().OpenFile(gomock.Eq(fmt.Sprintf("%s/%s", "", testFile)), gomock.Eq(0)).Return(mockSourceFile, nil)

	mockUserClient := clients.NewMockUserClient(ctrl)

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, osClient, mockUserClient)

	getFilesystemError := testFilesystemService.CopyFilesToRootFs(mockFileSystem, testFile, "destPath", false)

	assert.Nil(t, getFilesystemError)

}

func TestFileSystemServiceImpl_CopyFilesToRootFs_CopySingleFile_NilFileInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testFile := "testFile.txt"

	mockFileSystem := clients.NewMockFileSystemWrapper(ctrl)
	mockFileSystem.EXPECT().ReadDir(gomock.Any()).Times(2).DoAndReturn(func(sourceDir string) ([]os.FileInfo, error) {
		if sourceDir == "/" {
			return nil, nil
		}
		return nil, nil
	})

	testFilesystemService := GetFileSystemService()
	testFilesystemService.initialize(&logrus.Logger{}, nil, nil)

	getFilesystemError := testFilesystemService.CopyFilesToRootFs(mockFileSystem, testFile, "destPath", false)

	assert.NotNil(t, getFilesystemError)
	assert.Errorf(t, getFilesystemError, fmt.Sprintf("file %s could not be found", testFile))

}
