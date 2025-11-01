module "test_k8s_cluster" {
  source                   = "git@github.com:zevrant/zevrant-services-terraform//modules/kvm/k8s"
  cluster_default_ssh_keys = var.ssh_keys
  cluster_default_user     = var.default_user
  cluster_name             = "test-k8s-cluster"
  cluster_network_bridge   = "shared"
  cluster_network_gateway  = "10.1.0.1"
  control_plane_endpoint   = "k8s.zevrant-services.internal"
  controller_ip_addresses = {
    "10.1.0.14/24" : {
      controller_cpu    = 2
      controller_memory = 4096 # Controller RAM in MB
      node_name         = "proxmox-03"
      os_storage_name   = "local-zfs"
      os_image_location = "0/k8s-base-0.0.12.qcow2" # should look like 0/k8s-base-xx.xx.xx.qcow2
      power_state       = "running"
      vm_id             = 8000
    },
  }
  k8s_ca_init_private_key       = tls_private_key.zs_k8s_ca_private_key.private_key_pem
  k8s_ca_init_public_cert       = tls_self_signed_cert.zs-k8s_ca_cert.cert_pem
  mass_storage_name             = "local-zfs"
  network_nameserver            = "10.0.0.8"
  os_disk_size                  = 50
  pod_network_cidr              = "10.5.0.0/24"
  service_network_cidr          = "10.6.0.0/24"
  worker_ip_addresses           = {
    "10.1.0.15/24" : {
      controller_cpu    = 2
      controller_memory = 4096 # Controller RAM in MB
      node_name         = "proxmox-03"
      os_storage_name   = "local-zfs"
      os_image_location = "0/k8s-base-0.0.12.qcow2" # should look like 0/k8s-base-xx.xx.xx.qcow2
      power_state       = "running"
      vm_id             = 8001
    }
  }
  worker_container_storage_size = 50
}
