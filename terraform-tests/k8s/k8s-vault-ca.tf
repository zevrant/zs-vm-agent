variable "ca_valid_days" {
  default = 1825
}

# Generate a private key resource using the ECDSA algorithm
resource "tls_private_key" "zs_k8s_ca_private_key" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P521"
}

# Generate a self-signed certificate for our local Certificate Authority
# Right now the only variable in use is ca_valid_days
# Placing the certificate subject into variables is also useful for creating multiple certificates
resource "tls_self_signed_cert" "zs-k8s_ca_cert" {

  # Required Fields
  private_key_pem = tls_private_key.zs_k8s_ca_private_key.private_key_pem

  validity_period_hours = var.ca_valid_days * 24

  allowed_uses = [
    "digital_signature",
    "cert_signing",
    "crl_signing",
  ]

  # The resources does not 'require' this field, however, creating a CA requires it
  is_ca_certificate = true

  # Optional Fields
  # Reference the TLS provider documentation for additional information on fields
  subject {
    country      = "US"
    common_name  = "Zevrant Services K8s Root CA"
    organization = "zevrant-services.internal"
  }
}