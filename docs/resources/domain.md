---
page_title: "staysa_domain Resource - Staysa Cloud"
subcategory: ""
description: |-
  A custom domain for a web application. In a zone hosted on the platform its DNS records are written and it is verified at once; elsewhere, create dns_records at your DNS provider, then verify it in the console.
---

# staysa_domain (Resource)

A custom domain for a web application. In a zone hosted on the platform its DNS records are written and it is verified at once; elsewhere, create dns_records at your DNS provider, then verify it in the console.

## Example Usage

```terraform
resource "staysa_domain" "shop" {
  project  = "shop"
  service  = "web"
  hostname = "shop.example.com"
}
```

## Schema

### Required

- `hostname` (string) Domain name, such as shop.example.com.
- `project` (string)
- `service` (string) Application ID.

### Read-Only

- `dns_records` (list(object)) Records to create at your DNS provider: the CNAME to the application and the TXT ownership check.
- `verified` (bool) Ownership verified; HTTPS is then served.

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_domain.shop shop/web/shop.example.com
```
