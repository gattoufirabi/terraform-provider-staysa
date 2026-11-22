---
page_title: "staysa_dns_record Resource - Staysa Cloud"
subcategory: ""
description: |-
  A record in a hosted DNS zone. Any change replaces the record.
---

# staysa_dns_record (Resource)

A record in a hosted DNS zone. Any change replaces the record.

## Example Usage

```terraform
resource "staysa_dns_record" "www" {
  zone  = "example.com"
  name  = "www"
  type  = "CNAME"
  value = "web-shop-acme.apps.example-cloud.net"
}
```

## Schema

### Required

- `name` (string) Name inside the zone (www), or @ for the zone itself.
- `type` (string) A, AAAA, CNAME, MX, TXT, CAA, SRV or NS.
- `value` (string)
- `zone` (string) Zone domain.

### Optional

- `priority` (number) MX and SRV priority.
- `ttl` (number) Time to live in seconds.

### Read-Only

- `id` (string)

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_dns_record.www example.com/RECORD-ID
```
