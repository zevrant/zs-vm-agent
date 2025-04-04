package clients

import "github.com/sirupsen/logrus"

var infraConfigMapperClient InfraConfigMapperClient

func Initialize(logger *logrus.Logger) {
	infraConfigMapperClient = &InfraConfigMapperClientImpl{}

	infraConfigMapperClient.initialize(logger)
}

func GetInfraConfigMapperClient() InfraConfigMapperClient {
	return infraConfigMapperClient
}
