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
	GetVmDetailsByHostname() (ProxmoxVm, error)
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

func (infraMapperClient *InfraConfigMapperClientImpl) GetVmDetailsByHostname() ([]string, error) {
	request, requestCreationError := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/state/vm/%s", infraMapperClient.configMapperUrl, infraMapperClient.httpClient.hostURL),
		nil)

	if requestCreationError != nil {
		infraMapperClient.logger.Error(fmt.Sprintf("Failed to create detailed whoami request object %s", requestCreationError.Error()))
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
		infraMapperClient.logger.Errorf("Failed to parse json response into proxmox vm details, %s", jsonProcessingError.Error())
		return nil, jsonProcessingError
	}

	return parsedResponse, nil
}

type ProxmoxVm struct {
	Acpi                    bool     `json:"acpi"`
	Bios                    string   `json:"bios"`
	BootOrder               []string `json:"boot_order"`
	CloudInitStorageName    string   `json:"cloud_init_storage_name"`
	Cores                   float64  `json:"cores"`
	CpuLimit                float64  `json:"cpu_limit"`
	CpuType                 string   `json:"cpu_type"`
	DefaultUser             string   `json:"default_user"`
	Description             string   `json:"description"`
	HostStartupOrder        float64  `json:"host_startup_order"`
	Kvm                     bool     `json:"kvm"`
	Memory                  float64  `json:"memory"`
	Name                    string   `json:"name"`
	Nameserver              string   `json:"nameserver"`
	NodeName                string   `json:"node_name"`
	NumaActive              bool     `json:"numa_active"`
	OsType                  string   `json:"os_type"`
	PerformCloudInitUpgrade bool     `json:"perform_cloud_init_upgrade"`
	PowerState              string   `json:"power_state"`
	Protection              bool     `json:"protection"`
	QemuAgentEnabled        bool     `json:"qemu_agent_enabled"`
	ScsiHw                  string   `json:"scsi_hw"`
	Sockets                 float64  `json:"sockets"`
	SshKeys                 []string `json:"ssh_keys"`
	StartOnBoot             bool     `json:"start_on_boot"`
	Tags                    []string `json:"tags"`
	VmId                    string   `json:"vm_id"`
	Vmgenid                 string   `json:"vmgenid"`
	Disk                    []struct {
		AsyncIo         string `json:"async_io"`
		BackupEnabled   bool   `json:"backup_enabled"`
		BusType         string `json:"bus_type"`
		Cache           string `json:"cache"`
		DiscardEnabled  bool   `json:"discard_enabled"`
		Id              int    `json:"id"`
		ImportFrom      string `json:"import_from"`
		ImportPath      string `json:"import_path"`
		IoThread        bool   `json:"io_thread"`
		Order           int    `json:"order"`
		ReadOnly        bool   `json:"read_only"`
		Replicate       bool   `json:"replicate"`
		Size            string `json:"size"`
		SsdEmulation    bool   `json:"ssd_emulation"`
		StorageLocation string `json:"storage_location"`
	} `json:"disk"`
	IpConfig []struct {
		Gateway   string `json:"gateway"`
		IpAddress string `json:"ip_address"`
		Order     int    `json:"order"`
	} `json:"ip_config"`
	NetworkInterface []struct {
		Bridge     string `json:"bridge"`
		Firewall   bool   `json:"firewall"`
		MacAddress string `json:"mac_address"`
		Mtu        int    `json:"mtu"`
		Order      int    `json:"order"`
		Type       string `json:"type"`
	} `json:"network_interface"`
}
