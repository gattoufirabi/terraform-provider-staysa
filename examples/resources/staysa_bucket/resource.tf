resource "staysa_bucket" "uploads" {
  name     = "uploads"
  quota_mb = 1024
}
