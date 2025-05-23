package services

import (
	"errors"
	"zs-vm-agent/clients"

	"github.com/sirupsen/logrus"
)

type VaultService interface {
	initialize(logger *logrus.Logger)
	UnsealVault(vaultApiUrl string, unsealKeys []string) error
}

type VaultServiceImpl struct {
	logger      *logrus.Logger
	vaultClient clients.VaultClient
}

func (vaultService *VaultServiceImpl) initialize(logger *logrus.Logger) {
	vaultService.logger = logger
	vaultService.vaultClient = clients.GetVaultClient()
}

func (vaultService *VaultServiceImpl) UnsealVault(vaultApiUrl string, unsealKeys []string) error {
	//         curl -i --request PUT --data @/var/zevrant-services/vault-keys/vault-key-1 https://${URL}/v1/sys/unseal
	//		   status=$(curl https://${URL}/v1/sys/seal-status | jq .sealed)

	//Checking if vault is up
	initialized := false
	for !initialized {
		vaultStatus, getVaultStatusError := vaultService.vaultClient.GetVaultStatus(vaultApiUrl)

		if getVaultStatusError != nil {
			return getVaultStatusError
		}
		initialized = vaultStatus.Initialized
	}

	for _, key := range unsealKeys {
		submitUnsealKeyError := vaultService.vaultClient.SubmitUnsealKey(vaultApiUrl, key)

		if submitUnsealKeyError != nil {
			return submitUnsealKeyError
		}
	}

	vaultStatus, getVaultStatusError := vaultService.vaultClient.GetVaultStatus(vaultApiUrl)

	if getVaultStatusError != nil {
		return getVaultStatusError
	}

	if vaultStatus.Sealed != false {
		vaultService.logger.Errorf("Vault was not unsealed after uploading all unseal keys")
		return errors.New("vault was not unsealed after uploading all unseal keys")
	}

	return nil
}
