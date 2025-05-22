locals {
  state = (var.is_primary)? "MASTER" : "BACKUP"
}

resource random_bytes rpc_password {
  length  = 32
}

module s3_internal_certificate {
  source         = "/home/zevrant/git/zevrant/zevrant-services-terraform/modules/kvm/vault-certificate"
  common_name    = "s3.zevrant-services.internal"
  pki_backend    = var.pki_role.backend
  pki_issuer_ref = var.pki_role.issuer_ref
  pki_role_name  = var.pki_role.name
  certificate_ttl = 6048000
}

module haproxy_secret_volume {
  source = "/home/zevrant/git/zevrant/zevrant-services-terraform/modules/kvm/secret-volume"
  secrets = [
    {
      filename = "haproxy.cfg"
      value    = file("${path.module}/haproxy.cfg")
    },
    {
      filename = "conf.d/https-ingress.cfg"
      value = file("${path.module}/https-ingress.cfg")
    },
    {
      filename = "conf.d/jenkins-ingress.cfg"
      value = file("${path.module}/jenkins-ingress.cfg")
    },
    {
      filename = "conf.d/garage.cfg"
      value = file("${path.module}/garage.cfg")
    },
    {
      filename = "certs/s3.pem"
      value = <<EOF
${module.s3_internal_certificate.private_pem}
${module.s3_internal_certificate.public_pem}
EOF
    },
    {
      filename = "certs/CA.pem"
      value = <<EOF
-----BEGIN CERTIFICATE-----
MIICnzCCAiSgAwIBAgIUPZa4yl9/sRCFe2sWnoFACjt3C6QwCgYIKoZIzj0EAwMw
JDEiMCAGA1UEAxMZemV2cmFudC1zZXJ2aWNlcy5pbnRlcm5hbDAeFw0yNTAxMTYw
MTM2MjFaFw0zNTAxMTQwMTM2NTFaMCQxIjAgBgNVBAMTGXpldnJhbnQtc2Vydmlj
ZXMuaW50ZXJuYWwwdjAQBgcqhkjOPQIBBgUrgQQAIgNiAASwDruXv2KH1i9hVZZ1
xaQaWZfcgEdZzvBPmvQ8sDvAQxDvvDudk+mHdItkJbm0NQvNilEc5MoN8Jh1LBCe
EQnz3Y1Y1QYx3l+Kz0lXXJU5RyZ0JuXp9n6D+YtXCmiCKG6jggEVMIIBETAOBgNV
HQ8BAf8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUe9aP9qQkIgFy
linNFVdxD8KbQ4swHwYDVR0jBBgwFoAUe9aP9qQkIgFylinNFVdxD8KbQ4swSAYI
KwYBBQUHAQEEPDA6MDgGCCsGAQUFBzAChixodHRwczovL3ZhdWx0LnpldnJhbnQt
c2VydmljZXMuY29tL3YxL3BraS9jYTAkBgNVHREEHTAbghl6ZXZyYW50LXNlcnZp
Y2VzLmludGVybmFsMD4GA1UdHwQ3MDUwM6AxoC+GLWh0dHBzOi8vdmF1bHQuemV2
cmFudC1zZXJ2aWNlcy5jb20vdjEvcGtpL2NybDAKBggqhkjOPQQDAwNpADBmAjEA
j90H3jyO19g3pI8f3q0pb27gC+hi8VYNSTj6ifLbwcmCgtgrXLjxlOVEE+oQcOfc
AjEAjsRsNJGBsyKGIZhC/i3hukR68jGofhmt8piEfjaOk4K4PZif+/y7xeDx5iwY
LRa6
-----END CERTIFICATE-----
EOF
    },
    {
      filename = "vm-config.json"
      value = <<EOF
{
  "ports": [
    {
      "port": 80
      "protocol": "tcp"
    },
    {
      "port": 443
      "protocol": "tcp"
    },
    {
      "port": 8080
      "protocol": "tcp"
    },
    {
      "port": 9000
      "protocol": "tcp"
    },
    {
      "port": 9001
      "protocol": "tcp"
    }

  ]
}
EOF
    }
  ]
}

module keepalived_secret_volume {
  source = "/home/zevrant/git/zevrant/zevrant-services-terraform/modules/kvm/secret-volume"
  secrets = [
    {
      filename = "keepalived.conf"
      value = templatefile("${path.module}/keepalived.conf.tftpl", {
        state = local.state
        is_replica = !var.is_primary
        my_ip_address = split("/", var.ip_address)[0]
        peer_ip_addresses = var.peer_ip_addresses
        replica_priority = var.replica_priority
        router_id = var.virtual_router_id
        vip_password = var.keepalived_password
        virtual_ip_address = var.virtual_ip_address

      })
    },
    {
      filename = "check.sh"
      value = "${path.module}/check.sh"
    }
  ]
}

resource proxmox_vm haproxy {
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
    "loadbalancer"
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
    import_path      = "0/haproxy-base-0.0.39.qcow2"
  }

  disk {
    #certs & config
    bus_type         = "scsi"
    storage_location = var.mass_storage
    size             = "1G"
    order            = 1
    import_from = "local"
    import_path = format("0/%s", module.haproxy_secret_volume.secret_volume_name)
  }

  disk {
    #keepalived
    bus_type         = "scsi"
    storage_location = var.mass_storage
    size             = "1G"
    order            = 2
    import_from      = "local"
    import_path = format("0/%s", module.keepalived_secret_volume.secret_volume_name)
  }


  network_interface {
    mac_address = "BC:24:11:${random_bytes.mac_address_1.hex}:${random_bytes.mac_address_2.hex}:${random_bytes.mac_address_3.hex}"
    bridge      = "shared"
    firewall    = true
    order       = 0
    mtu         = 1412
  }

  depends_on = [module.haproxy_secret_volume, module.keepalived_secret_volume]

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