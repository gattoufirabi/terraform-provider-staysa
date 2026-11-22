---
page_title: "staysa_bucket Resource - Staysa Cloud"
subcategory: ""
description: |-
  An S3-compatible storage bucket of the organization. Deleting it needs an admin token and deletes its files.
---

# staysa_bucket (Resource)

An S3-compatible storage bucket of the organization. Deleting it needs an admin token and deletes its files.

## Example Usage

```terraform
resource "staysa_bucket" "uploads" {
  name     = "uploads"
  quota_mb = 1024
}
```

## Schema

### Required

- `name` (string) 3 to 40 lowercase letters, digits and dashes.

### Optional

- `quota_mb` (number) Space reserved, in MB (default 1024 or what the plan has left). Change it in the console.

### Read-Only

- `endpoint` (string) S3 endpoint.
- `region` (string) S3 region.
- `s3_name` (string) Bucket name in S3 requests.

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_bucket.uploads uploads
```
