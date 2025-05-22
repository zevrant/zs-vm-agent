cpu_cores = 8
default_user = "zevrant"
description = "vm-agent-testing"
gateway = "10.1.0.1"
hostname = "haproxy-ingress-02"
ip_address = "10.1.0.100/24"
is_primary = true
keepalived_password = "ppass"
mass_storage = "local-zfs"
nameserver = "10.0.0.8"
peer_ip_addresses = []
pki_role = { #TODO use terraform remote state instead of hardcoding
  backend = "pki_shared"
  issuer_ref = "7149394f-5f66-0579-fced-27db21503f89"
  name = "zevrant-services-shared"
}
proxmox_host = "proxmox-03"
replica_priority = 1
ssd_storage = "local-zfs"
ssh_keys = [
  "ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQAAAAIbmlzdHAzODQAAABhBLtOxtriPtNmisKkmfHfCByaTYCHRsDHyzQAi0yL6LUeKybjYExfR6N0xBMcIj6M/b5U3aafjKayX4nMvV7s7/vcrpBfW+WvxOCBWTlhKGNpUmAS9ApFDn51/FTuRgB/YA=="
]
virtual_ip_address = "10.1.0.222/24"
memory_mbs = 8192
virtual_router_id = 2
vm_id = 9001