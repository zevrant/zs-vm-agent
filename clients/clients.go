package clients

import "github.com/sirupsen/logrus"

var infraConfigMapperClient InfraConfigMapperClientImpl
var osClient OsClientImpl
var userClient UserClientImpl
var vaultClient VaultClientImpl

func Initialize(logger *logrus.Logger, hostname string) {

	infraConfigMapperClient.initialize(logger, hostname)
	osClient.initialize(logger)
	userClient.initialize(logger)
	vaultClient.initialize(logger)
}

func GetInfraConfigMapperClient() InfraConfigMapperClient {
	return &infraConfigMapperClient
}

func GetOsClient() OsClient { return &osClient }

func GetUserClient() UserClient { return &userClient }

func GetVaultClient() VaultClient { return &vaultClient }
