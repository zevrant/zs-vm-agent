data minio_s3_object vpc_configurations {
  object_name = "shared-common.json"
  bucket_name = "vm-configuration"
}


resource proxmox_vm hashi-vault-agent-test {
  name               = var.hostname
  qemu_agent_enabled = true
  cores              = "${var.cpu_cores}"
  memory             = "${var.memory_mbs}"
  os_type            = "l26"
  description        = var.description
  node_name          = var.proxmox_host
  vm_id              = "${ var.vm_id }"
  cpu_type           = "host"
  boot_order = ["scsi0"]
  host_startup_order = var.host_startup_order
  protection         = var.is_protected
  nameserver         = var.nameserver
  default_user       = var.default_user
  start_on_boot      = var.start_on_boot
  ssh_keys           = var.ssh_keys
  power_state = "running"
  tags = [
    "dns"
  ]
  ip_config {
    ip_address = var.ip_address
    gateway    = var.gateway
    order      = 0
  }

  disk {
    bus_type         = "scsi"
    storage_location = var.ssd_storage
    size             = "50G"
    order            = 0
    import_from      = "local"
    //Must be preloaded at this location, full path is /var/lib/vz/images/0/AlmaLinux-9-GenericCloud-latest.x86_64.qcow2
    //Long term recommendation is to use an nfs mount or something that supports RWM
    import_path      = "0/hashicorp-vault-base-0.0.30.qcow2"
  }

  disk {
    #certs & config
    bus_type         = "scsi"
    storage_location = var.mass_storage
    size             = "1G"
    order            = 1
    import_from = "local"
    import_path = format("0/%s", module.vault_secret_volume.secret_volume_name)
  }


  network_interface {
    mac_address = "BC:24:11:${random_bytes.mac_address_1.hex}:${random_bytes.mac_address_2.hex}:${random_bytes.mac_address_3.hex}"
    bridge      = "shared"
    firewall    = true
    order       = 0
    mtu         = 1412
  }

}

resource random_bytes mac_address_1 {
  length = 1
}
resource random_bytes mac_address_2 {
  length = 1
}
resource random_bytes mac_address_3 {
  length = 1
}

module vault_certificate {
  source      = "../modules/kvm/certificates"
  common_name = "vault.zevrant-services.com"
}


module vault_secret_volume {
  source = "../modules/kvm/secret-volume"
  secrets = [
    {
      filename = "vault-public.pem"
      value    = module.vault_certificate.public_pem
    },
    {
      filename = "vault-private.pem"
      value    = module.vault_certificate.private_pem
    },
    {
      filename = "vault-issuer.pem"
      value    = module.vault_certificate.issuer_pem
    },
    {
      filename = "vault-key-1"
      value    = trimspace(data.vault_kv_secret_v2.key1.data.password)
    },
    {
      filename = "vault-key-2"
      value    = trimspace(data.vault_kv_secret_v2.key2.data.password)
    },
    {
      filename = "vault-key-3"
      value    = trimspace(data.vault_kv_secret_v2.key3.data.password)
    }
  ]
}