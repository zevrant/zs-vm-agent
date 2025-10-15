locals {
  environment = "test"
}

data minio_s3_object vpc_configurations {
  object_name = "shared-common.json"
  bucket_name = "vm-configuration"
}


module dns_server {
  source = "git@github.com:zevrant/zevrant-services-terraform//modules/kvm/dns"
  dns_servers = var.dns_servers
  environment = local.environment
  configurations_volume_path = jsondecode(data.minio_s3_object.vpc_configurations.content).dns_secret_config
  keepalived_password = jsondecode(data.minio_s3_object.vpc_configurations.content).dns_keepalived_password
  virtual_router_id   = jsondecode(data.minio_s3_object.vpc_configurations.content).dns_virtual_router_id
  virtual_ip_address  = jsondecode(data.minio_s3_object.vpc_configurations.content).dns_virtual_ip_address
}

# data proxmox_health_check_systemd vm_agent {
#   service_name = "zs-vm-agent"
#   address = "10.1.0.100"
#   custom_port = 9100
#   path = "/metrics"
#   depends_on = [module.dns_server]
# }
#
# data proxmox_health_check_systemd named {
#
# }
