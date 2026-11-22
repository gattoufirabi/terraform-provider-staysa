resource "staysa_project_variable" "region" {
  project = "shop"
  name    = "REGION"
  value   = "eu-west"
  secret  = false
}
