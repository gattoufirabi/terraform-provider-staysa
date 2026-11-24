# Let one of the organization's servers run up to two builds at once.
resource "staysa_server_builds" "builder1" {
  server     = "c-k3x9q2m7wd"
  labels     = ["large"]
  max_builds = 2
}
