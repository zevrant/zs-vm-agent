terraform {
  required_providers {
    proxmox = {
      source = "app.terraform.io/zevrant-services/proxmox"
    }
    vault = {
      source  = "hashicorp/vault"
    }
  }

}

provider vault {
  address = var.VAULT_ADDR
  token   = var.VAULT_TOKEN
  skip_tls_verify = true //TODO: fix this
}

provider "proxmox" {
  verify_tls = false
  host       = "https://10.0.0.2:8006"
  username   = var.proxmox_username
  password   = var.proxmox_password
}
