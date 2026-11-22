---
page_title: "staysa_service_secret Resource - Staysa Cloud"
subcategory: ""
description: |-
  An encrypted secret given to an application as an environment variable. The platform never shows the value again, so changes made outside Terraform are not detected.
---

# staysa_service_secret (Resource)

An encrypted secret given to an application as an environment variable. The platform never shows the value again, so changes made outside Terraform are not detected.

## Example Usage

```terraform
resource "staysa_service_secret" "db" {
  project = "shop"
  service = "web"
  name    = "DATABASE_URL"
  value   = staysa_database.main.url
}
```

## Schema

### Required

- `name` (string) Variable name: uppercase letters, digits and underscores.
- `project` (string)
- `service` (string) Application ID.
- `value` (string, Sensitive) Value (up to 8192 characters).

### Optional

- `redeploy` (bool) Redeploy the application after a change so it gets the new value (default true).

