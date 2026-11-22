resource "staysa_database" "main" {
  project     = "shop"
  id          = "main"
  engine      = "postgres:17"
  memory_mb   = 512
  backup_hour = 3
}
