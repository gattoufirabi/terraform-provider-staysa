---
page_title: "Provider: Staysa Cloud"
description: |-
  Manage projects, applications, databases, domains, DNS and storage on a Staysa Cloud platform.
---

# Staysa Cloud Provider

Manage projects, applications, databases, domains, DNS and storage on a Staysa Cloud platform.

Authenticate with an API token created in the console under **Access > API tokens**
(admin role to create and delete projects). Set `STAYSA_ENDPOINT` and `STAYSA_TOKEN`
in the environment rather than in files.

Every change goes through the platform's own checks, plan limits and audit trail. Changes
to an application are deployed blue/green, and the provider waits until the new version is
live (or fails with the deployment's message).

## Example Usage

```terraform
provider "staysa" {
  # endpoint = "https://console.example.com" # or STAYSA_ENDPOINT
  # token    = var.staysa_token               # or STAYSA_TOKEN (pla_...)
}
```

A complete project (database, autoscaled application, secret) is in
[examples/complete](https://github.com/gattoufirabi/terraform-provider-staysa/tree/main/examples/complete).

## Schema

### Optional

- `endpoint` (string) Address of the console, such as https://console.example.com. Defaults to the STAYSA_ENDPOINT environment variable.
- `token` (string, Sensitive) API token (pla_...) created under Access > API tokens. Defaults to the STAYSA_TOKEN environment variable.
