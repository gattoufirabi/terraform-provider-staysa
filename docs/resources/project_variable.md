---
page_title: "staysa_project_variable Resource - Staysa Cloud"
subcategory: ""
description: |-
  A variable shared by every application of a project (applied at their next deployment).
---

# staysa_project_variable (Resource)

A variable shared by every application of a project (applied at their next deployment).

## Example Usage

```terraform
resource "staysa_project_variable" "region" {
  project = "shop"
  name    = "REGION"
  value   = "eu-west"
  secret  = false
}
```

## Schema

### Required

- `name` (string) Uppercase letters, digits and underscores.
- `project` (string)
- `value` (string, Sensitive)

### Optional

- `secret` (bool) Encrypted and never shown again (default true). Plain variables can be read back, so outside changes are detected.

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_project_variable.region shop/REGION
```
