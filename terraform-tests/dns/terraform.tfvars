dns_servers = {
  "test" : {
    cpu = 8
    default_user = "zevrant"
    description  = "vm-agent-testing"
    gateway      = "10.1.0.1"
    hostname     = "dns-shared-green-01"
    ip_address   = "10.1.0.100/24"
    is_primary   = true

    mass_storage_name = "local-zfs"
    memory_mbs        = 8192
    nameserver        = "10.0.0.8"
    peer_ip_addresses = []
    power_state       = "running"
    protection        = false
    proxmox_host      = "proxmox-03"
    replica_priority  = 1
    ssd_storage_name  = "local-zfs"
    start_on_boot     = true
    vm_id             = 9002
  }
}
