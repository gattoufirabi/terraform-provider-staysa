resource "staysa_service" "web" {
  project     = "shop"
  id          = "web"
  image       = "nginx:1.27-alpine"
  port        = 80
  health_path = "/"
  cpu_millis  = 250
  memory_mb   = 256
  env = {
    MODE = "production"
  }
  autoscaling = {
    min         = 1
    max         = 4
    cpu_percent = 70
  }
}
