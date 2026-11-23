# A project with a PostgreSQL database and an autoscaled web application that gets the
# database URL as a secret. Set STAYSA_ENDPOINT and STAYSA_TOKEN, then: tofu init && tofu apply
terraform {
  required_providers {
    staysa = {
      source = "registry.opentofu.org/gattoufirabi/staysa"
    }
  }
}

provider "staysa" {}

resource "staysa_project" "shop" {
  id   = "shop"
  name = "Shop"
}

resource "staysa_database" "main" {
  project   = staysa_project.shop.id
  id        = "main"
  engine    = "postgres"
  memory_mb = 512
}

resource "staysa_service" "web" {
  project    = staysa_project.shop.id
  id         = "web"
  image      = "nginx:1.27-alpine"
  port       = 80
  cpu_millis = 250
  memory_mb  = 256
  env = {
    MODE = "production"
  }
  autoscaling = {
    min = 1
    max = 4
  }
}

resource "staysa_service_secret" "db" {
  project = staysa_project.shop.id
  service = staysa_service.web.id
  name    = "DATABASE_URL"
  value   = staysa_database.main.url
}

output "address" {
  value = staysa_service.web.url
}
