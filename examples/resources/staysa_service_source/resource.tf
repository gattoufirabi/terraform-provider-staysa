# Build the application from an on-premises GitLab, on the organization's own
# servers that have the "large" label.
resource "staysa_service_source" "web" {
  project      = "shop"
  service      = "web"
  repo         = "https://gitlab.internal.example/shop/web"
  branch       = "main"
  token        = var.gitlab_read_token
  build_on     = "servers"
  build_labels = ["large", "amd64"]
}
