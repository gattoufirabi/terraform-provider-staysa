---
page_title: "staysa_project Data Source - Staysa Cloud"
subcategory: ""
description: |-
  An existing project.
---

# staysa_project (Data Source)

An existing project.

## Example Usage

```terraform
data "staysa_project" "shop" {
  id = "shop"
}
```

## Schema

### Required

- `id` (string) Project ID.

### Read-Only

- `name` (string) Display name.

