---
page_title: "staysa_dns_zone Resource - Staysa Cloud"
subcategory: ""
description: |-
  A DNS zone hosted on the platform. Set the nameservers at your registrar; the zone is served once the delegation is seen. Deleting it needs an admin token.
---

# staysa_dns_zone (Resource)

A DNS zone hosted on the platform. Set the nameservers at your registrar; the zone is served once the delegation is seen. Deleting it needs an admin token.

## Example Usage

```terraform
resource "staysa_dns_zone" "main" {
  domain = "example.com"
}
```

## Schema

### Required

- `domain` (string) Domain, such as example.com.

### Read-Only

- `nameservers` (list(string)) Name servers to set at your registrar.
- `status` (string) pending until the registrar delegates the domain, then active.

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_dns_zone.main example.com
```
