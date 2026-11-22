---
page_title: "staysa_service Data Source - Staysa Cloud"
subcategory: ""
description: |-
  An existing application.
---

# staysa_service (Data Source)

An existing application.

## Example Usage

```terraform
data "staysa_service" "web" {
  project = "shop"
  id      = "web"
}
```

## Schema

### Required

- `id` (string)
- `project` (string)

### Read-Only

- `image` (string)
- `instances` (number)
- `internal_host` (string)
- `port` (number)
- `status` (string)
- `url` (string)

