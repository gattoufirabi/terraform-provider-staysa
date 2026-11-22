---
page_title: "staysa_database Resource - Staysa Cloud"
subcategory: ""
description: |-
  A managed database (PostgreSQL, MySQL, MariaDB, MongoDB, Redis or Kafka) on a project's private network. Deleting it needs an admin token and deletes its data.
---

# staysa_database (Resource)

A managed database (PostgreSQL, MySQL, MariaDB, MongoDB, Redis or Kafka) on a project's private network. Deleting it needs an admin token and deletes its data.

## Example Usage

```terraform
resource "staysa_database" "main" {
  project     = "shop"
  id          = "main"
  engine      = "postgres:17"
  memory_mb   = 512
  backup_hour = 3
}
```

## Schema

### Required

- `engine` (string) postgres, mysql, mariadb, mongodb, redis or kafka, optionally with a version (postgres:16). Changing it replaces the database.
- `id` (string) Database ID: 2 to 20 lowercase letters, digits and dashes.
- `project` (string) Project ID.

### Optional

- `backup_enabled` (bool) Daily backups (on by default; Kafka has none).
- `backup_hour` (number) Hour of the daily backup, 0 to 23 UTC.
- `cpu_millis` (number) CPU in millicores (50 to 4000).
- `memory_mb` (number) Memory in MB (128 to 8192; each engine has a minimum).
- `wait_for_running` (bool) Wait until the database runs after creation (default true).

### Read-Only

- `database` (string) Database name.
- `host` (string) Host name on the project's private network.
- `password` (string, Sensitive) Password (needs a developer or admin token).
- `port` (number) Port.
- `secret_name` (string) Usual secret name for the URL (DATABASE_URL, REDIS_URL...).
- `status` (string) starting, running, failed...
- `url` (string, Sensitive) Connection URL, to give an application as a secret.
- `username` (string) User name.
- `version` (string) Engine version.

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_database.main shop/main
```
