package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type VaultClient interface {
	SubmitUnsealKey(vaultApiUrl string, unsealKey string) error
	GetVaultStatus(vaultApiUrl string) (*VaultStatusResponse, error)
}

type VaultClientImpl struct {
	logger     *logrus.Logger
	httpClient *Client
}

func (vaultClient *VaultClientImpl) initialize(logger *logrus.Logger) {
	vaultClient.logger = logger
	vaultClient.httpClient = nil
}

func (vaultClient *VaultClientImpl) SubmitUnsealKey(vaultApiUrl string, unsealKey string) error {
	vaultClient.logger.Debugf("Vault URL is %s", vaultApiUrl)
	vaultClient.httpClient = NewClient(vaultApiUrl, "", "", true, vaultClient.logger)

	request, requestCreationError := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/v1/sys/unseal", vaultClient.httpClient.hostURL),
		bytes.NewBufferString(fmt.Sprintf("{\"key\": \"%s\"}", unsealKey)))

	if requestCreationError != nil {
		vaultClient.logger.Errorf("Failed to create request to unseal vault at %s: %s", vaultClient.httpClient.hostURL, requestCreationError.Error())
		return requestCreationError
	}

	_, doRequestError := vaultClient.httpClient.doRequest(request, "")

	if doRequestError != nil {
		vaultClient.logger.Errorf("Failed to perform request to unseal vault at %s: %s", vaultClient.httpClient.hostURL, doRequestError.Error())
		return doRequestError
	}

	return nil
}

func (vaultClient *VaultClientImpl) GetVaultStatus(vaultApiUrl string) (*VaultStatusResponse, error) {
	vaultClient.logger.Debugf("Vault URL is %s", vaultApiUrl)
	vaultClient.httpClient = NewClient(vaultApiUrl, "", "", true, vaultClient.logger)
	request, requestCreationError := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/v1/sys/seal-status", vaultClient.httpClient.hostURL),
		nil)

	if requestCreationError != nil {
		vaultClient.logger.Errorf("Failed to create request to get vault status at %s: %s", vaultClient.httpClient.hostURL, requestCreationError.Error())
		return nil, requestCreationError
	}

	response, doRequestError := vaultClient.httpClient.doRequest(request, "")

	if doRequestError != nil {
		vaultClient.logger.Errorf("Failed to perform request to check vault status at %s: %s", vaultClient.httpClient.hostURL, doRequestError.Error())
		return nil, doRequestError
	}

	var vaultStatus VaultStatusResponse

	unmarshalError := json.Unmarshal(response, &vaultStatus)

	if unmarshalError != nil {
		vaultClient.logger.Errorf("Failed to unmarshal vault status response into a known response: %s", unmarshalError)
		return nil, unmarshalError
	}

	return &vaultStatus, nil
}

type VaultStatusResponse struct {
	Type         string    `json:"type"`
	Initialized  bool      `json:"initialized"`
	Sealed       bool      `json:"sealed"`
	T            int       `json:"t"`
	N            int       `json:"n"`
	Progress     int       `json:"progress"`
	Nonce        string    `json:"nonce"`
	Version      string    `json:"version"`
	BuildDate    time.Time `json:"build_date"`
	Migration    bool      `json:"migration"`
	RecoverySeal bool      `json:"recovery_seal"`
	StorageType  string    `json:"storage_type"`
}
