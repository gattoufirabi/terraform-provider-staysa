---
page_title: "staysa_service_source Resource - Staysa Cloud"
subcategory: ""
description: |-
  The git repository an application is built from, and where its builds run: on the platform's builder, or on the organization's own build servers (see staysa_server_builds). Destroying it removes the repository from the application.
---

# staysa_service_source (Resource)

The git repository an application is built from, and where its builds run: on the platform's builder, or on the organization's own build servers (see staysa_server_builds). Destroying it removes the repository from the application.

## Example Usage

```terraform
# Build the application from an on-premises GitLab, on the organization's own
# servers that have the "large" label.
resource "staysa_service_source" "web" {
  project      = "shop"
  service      = "web"
  repo         = "https://gitlab.internal.example/shop/web"
  branch       = "main"
  token        = var.gitlab_read_token
  build_on     = "servers"
  build_labels = ["large", "amd64"]
}
```

## Schema

### Required

- `project` (string)
- `repo` (string) HTTPS address of the git repository. Builds on your own servers may also use a private address, such as an on-premises GitLab.
- `service` (string) Application ID.

### Optional

- `branch` (string) Branch built on each push (default main).
- `build_labels` (set of string) With build_on = servers: a build runs only on a server that has every one of these labels. A server's operating system and architecture (linux, amd64, arm64) count as labels.
- `build_on` (string) Where builds run: platform (the platform's builder, billed as build minutes) or servers (the organization's own servers that run builds).
- `dir` (string) Folder inside the repository to build, such as services/web; empty for the root.
- `pr_previews` (bool) Deploy a preview of each pull request.
- `token` (string, Sensitive) Read token of a private repository. The platform never shows it again, so changes made outside Terraform are not detected; removing it from the configuration removes it from the platform.

### Read-Only

- `has_token` (bool) Whether the platform holds a read token.

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_service_source.web shop/web
```
