package clients

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

type InfraConfigMapperClient interface {
	initialize(logger *logrus.Logger)
	GetTagsByHostname() ([]string, error)
}

type InfraConfigMapperClientImpl struct {
	configMapperUrl string
	httpClient      *Client
	logger          *logrus.Logger
}

func (infraMapperClient *InfraConfigMapperClientImpl) initialize(logger *logrus.Logger) {
	infraMapperClient.logger = logger
	infraMapperClient.configMapperUrl = os.Getenv("INFRA_CONFIG_MAPPER_URL")
	hostname := os.Getenv("HOSTNAME")
	infraMapperClient.httpClient = NewClient(hostname, "", "", false, logger)
}

func (infraMapperClient *InfraConfigMapperClientImpl) GetTagsByHostname() ([]string, error) {
	request, requestCreationError := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/state/vm/%s/tags", infraMapperClient.configMapperUrl, infraMapperClient.httpClient.hostURL),
		nil)

	if requestCreationError != nil {
		infraMapperClient.logger.Error(fmt.Sprintf("Failed to create whoami request object %s", requestCreationError.Error()))
		return nil, requestCreationError
	}

	response, getIdentityError := infraMapperClient.httpClient.doRequest(request, "application/json")

	if getIdentityError != nil {
		infraMapperClient.logger.Errorf("Failed to retrieve identity, %s", getIdentityError.Error())
		return nil, getIdentityError
	}

	var parsedResponse []string

	jsonProcessingError := json.Unmarshal(response, &parsedResponse)

	if jsonProcessingError != nil {
		infraMapperClient.logger.Errorf("Failed to parse json response into list of tags, %s", jsonProcessingError.Error())
		return nil, jsonProcessingError
	}

	return parsedResponse, nil
}
