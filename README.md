# Terraform / OpenTofu provider for Staysa Cloud

Manage projects, applications (with autoscaling), managed databases, custom domains, hosted
DNS and object storage on a [Staysa Cloud](https://github.com/gattoufirabi/cloud-experiment)
platform as code. Works with OpenTofu and Terraform 1.0+.

```hcl
terraform {
  required_providers {
    staysa = {
      source = "registry.opentofu.org/gattoufirabi/staysa"
    }
  }
}

provider "staysa" {} # STAYSA_ENDPOINT and STAYSA_TOKEN from the environment

resource "staysa_project" "shop" {
  id = "shop"
}

resource "staysa_service" "web" {
  project = staysa_project.shop.id
  id      = "web"
  image   = "nginx:1.27-alpine"
  port    = 80
}
```

| Resources | Data sources |
|---|---|
| `staysa_project`, `staysa_service`, `staysa_service_secret`, `staysa_project_variable`, `staysa_database`, `staysa_domain`, `staysa_dns_zone`, `staysa_dns_record`, `staysa_bucket`, `staysa_service_source`, `staysa_server_builds` | `staysa_project`, `staysa_service`, `staysa_database` |

Documentation for every attribute and import ID is in [`docs/`](docs/) and on the registry.
A complete example (database, autoscaled application, secret) is in
[`examples/complete`](examples/complete/main.tf).

## Authentication

Create an API token in the console under **Access > API tokens**. The **admin** role is needed
to create and delete projects, to delete databases, zones and buckets, and for
`staysa_server_builds`; **developer** is
enough for applications, secrets and domains. A token limited to one project manages only that
project.

```
export STAYSA_ENDPOINT=https://console.example.com
export STAYSA_TOKEN=pla_...
```

## Develop

```
go build -o terraform-provider-staysa .
```

Point OpenTofu (`~/.tofurc`, Windows `%APPDATA%\tofu.rc`) or Terraform (`~/.terraformrc`) at the
folder holding the binary, then run `plan` / `apply` without `init`:

```hcl
provider_installation {
  dev_overrides {
    "registry.opentofu.org/gattoufirabi/staysa" = "/path/to/terraform-provider-staysa"
  }
  direct {}
}
```

## Releases

Releases are built by GoReleaser and signed with the key in
[`release-signing-key.asc`](release-signing-key.asc)
(fingerprint `51353DC9411B4E07B5EE0730B3517F9E206ED882`). Push a `vX.Y.Z` tag; the `release`
workflow publishes the zips, `SHA256SUMS` and its signature.

## License

[MPL-2.0](LICENSE)
