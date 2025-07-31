terraform {
  backend "pg" {
    # conn_str = "postgres://${var.tf_state_user}:${var.tf_state_password}@${var.tf_state_host}:${var.tf_state_port}/cicd_tf_backend"
    schema_name = "shared_remote_state"
  }
  required_providers {
    proxmox = {
      source = "app.terraform.io/zevrant-services/proxmox"
    }
    vault = {
      source  = "hashicorp/vault"
    }
    minio = {
      source = "aminueza/minio"
      version = "3.3.0"
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

provider minio {
  // required
  minio_server       = "s3.zevrant-services.internal"

  // optional
  minio_region      = "garage"
  minio_ssl         = true
}