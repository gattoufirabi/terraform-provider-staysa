---
page_title: "staysa_server_builds Resource - Staysa Cloud"
subcategory: ""
description: |-
  Makes one of the organization's own servers run builds, like a self-hosted CI agent: it takes the builds of applications whose staysa_service_source has build_on = servers and whose build_labels it has. Destroying it stops the server taking builds. Needs an organization-wide token with the admin role.
---

# staysa_server_builds (Resource)

Makes one of the organization's own servers run builds, like a self-hosted CI agent: it takes the builds of applications whose staysa_service_source has build_on = servers and whose build_labels it has. Destroying it stops the server taking builds. Needs an organization-wide token with the admin role.

## Example Usage

```terraform
# Let one of the organization's servers run up to two builds at once.
resource "staysa_server_builds" "builder1" {
  server     = "c-k3x9q2m7wd"
  labels     = ["large"]
  max_builds = 2
}
```

## Schema

### Required

- `server` (string) Server ID, as shown on the Your servers page.

### Optional

- `clean` (bool) Start every build from an empty cache and keep nothing after it, like a throwaway agent (slower builds).
- `labels` (set of string) Labels matched against an application's build_labels (up to 10; lowercase letters, digits, dots, dashes, underscores).
- `max_builds` (number) Builds the server runs at once, 1 to 8 (default 1).

### Read-Only

- `auto_labels` (set of string) Labels the server has automatically: its operating system and architecture.
- `name` (string) Server name.

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_server_builds.builder1 c-k3x9q2m7wd
```
