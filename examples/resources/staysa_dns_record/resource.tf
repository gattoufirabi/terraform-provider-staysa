resource "staysa_dns_record" "www" {
  zone  = "example.com"
  name  = "www"
  type  = "CNAME"
  value = "web-shop-acme.apps.example-cloud.net"
}
