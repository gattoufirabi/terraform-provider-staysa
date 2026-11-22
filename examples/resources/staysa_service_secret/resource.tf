resource "staysa_service_secret" "db" {
  project = "shop"
  service = "web"
  name    = "DATABASE_URL"
  value   = staysa_database.main.url
}
