package clients

import "github.com/sirupsen/logrus"

var infraConfigMapperClient InfraConfigMapperClientImpl
var osClient OsClientImpl
var userClient UserClientImpl

func Initialize(logger *logrus.Logger) {

	infraConfigMapperClient.initialize(logger)
	osClient.initialize(logger)
	userClient.initialize(logger)
}

func GetInfraConfigMapperClient() InfraConfigMapperClient {
	return &infraConfigMapperClient
}

func GetOsClient() OsClient { return &osClient }

func GetUserClient() UserClient { return &userClient }
