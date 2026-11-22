---
page_title: "staysa_database Data Source - Staysa Cloud"
subcategory: ""
description: |-
  An existing managed database and how to connect to it.
---

# staysa_database (Data Source)

An existing managed database and how to connect to it.

## Example Usage

```terraform
data "staysa_database" "main" {
  project = "shop"
  id      = "main"
}
```

## Schema

### Required

- `id` (string)
- `project` (string)

### Read-Only

- `database` (string)
- `engine` (string)
- `host` (string)
- `password` (string, Sensitive)
- `port` (number)
- `status` (string)
- `url` (string, Sensitive)
- `username` (string)
- `version` (string)

