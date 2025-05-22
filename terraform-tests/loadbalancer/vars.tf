variable auto_start {
  type = bool
  default = true
}

variable cpu_cores {
  type = number
}

variable default_user {
  type = string
}

variable description {
  type = string
}

variable gateway {
  type = string
}

variable hostname {
  type = string
}

variable host_startup_order{
  type = number
  default = 1
}

variable ip_address {
  type = string
  validation {
    condition = length(regex("\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}/\\d\\d", var.ip_address)) > 0
    error_message = "String does not appear to be a ipaddress/cidr, it should look like 10.0.0.1/24"
  }
}

variable is_primary {
  type = bool
}

variable is_protected {
  type = bool
  default = false
}

variable keepalived_password {
  type = string
  sensitive = true
  validation {
    condition = length(var.keepalived_password) <= 8
    error_message = "The max size password keepalived can use is 8 characters"
  }
}

variable mass_storage {
  type = string
}

variable memory_mbs{
  type = number
}

variable nameserver {
  type = string
}

variable peer_ip_addresses {
  type = list(string)
}

variable pki_role {
  type = object({
    backend = string
    issuer_ref = string
    name = string
  })
}

variable proxmox_host {
  type = string
}

variable replica_priority {
  type = number
  validation {
    condition = var.replica_priority <= 255
    error_message = "Replica priority should be less than or equal to 255"
  }
}

variable ssd_storage {
  type = string
}

variable ssh_keys {
  type = list(string)
}

variable start_on_boot {
  type = bool
  default = false
}

variable virtual_ip_address {
  type = string
  validation {
    condition = length(regex("\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}/\\d\\d", var.virtual_ip_address)) > 0
    error_message = "String does not appear to be a ipaddress/cidr, it should look like 10.0.0.1/24"
  }
}

variable virtual_router_id {
  type = number
}

variable vm_id {
  type = number
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
