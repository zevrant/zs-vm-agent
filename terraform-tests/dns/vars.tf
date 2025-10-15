variable dns_servers {
  type = map(object({
    cpu          = number
    default_user = string
    description  = string
    gateway      = string
    hostname     = string
    ip_address   = string
    is_primary   = bool

    mass_storage_name = string
    memory_mbs        = number
    nameserver        = string
    peer_ip_addresses = list(string)
    power_state       = string
    protection        = bool
    proxmox_host      = string
    replica_priority  = number
    ssd_storage_name  = string
    start_on_boot     = bool
    vm_id             = number
  }))
}

variable VAULT_ADDR {
  type      = string
  sensitive = true
}

variable VAULT_TOKEN {
  type      = string
  sensitive = true
}

variable proxmox_username {
  type = string
}

variable proxmox_password {
  type = string
}