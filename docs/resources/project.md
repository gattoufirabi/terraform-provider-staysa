---
page_title: "staysa_project Resource - Staysa Cloud"
subcategory: ""
description: |-
  A project: a group of applications and databases sharing a private network. Needs an admin token.
---

# staysa_project (Resource)

A project: a group of applications and databases sharing a private network. Needs an admin token.

## Example Usage

```terraform
resource "staysa_project" "shop" {
  id   = "shop"
  name = "Shop"
}
```

## Schema

### Required

- `id` (string) Project ID: 2 to 20 lowercase letters, digits and dashes. Changing it replaces the project.

### Optional

- `force_delete` (bool) Delete the project even if it still holds applications or databases not managed here. Default false.
- `name` (string) Display name (defaults to the ID).

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_project.shop shop
```
