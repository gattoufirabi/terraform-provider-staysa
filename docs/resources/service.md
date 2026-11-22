---
page_title: "staysa_service Resource - Staysa Cloud"
subcategory: ""
description: |-
  An application (web service or worker) running a container image. Each change is deployed (blue/green) unless deploy is false.
---

# staysa_service (Resource)

An application (web service or worker) running a container image. Each change is deployed (blue/green) unless deploy is false.

## Example Usage

```terraform
resource "staysa_service" "web" {
  project     = "shop"
  id          = "web"
  image       = "nginx:1.27-alpine"
  port        = 80
  health_path = "/"
  cpu_millis  = 250
  memory_mb   = 256
  env = {
    MODE = "production"
  }
  autoscaling = {
    min         = 1
    max         = 4
    cpu_percent = 70
  }
}
```

## Schema

### Required

- `id` (string) Application ID: 2 to 20 lowercase letters, digits and dashes.
- `image` (string) Container image, such as nginx:1.27-alpine or an image of the platform's registry.
- `project` (string) Project ID.

### Optional

- `autoscaling` (Attributes) Scale between min and max instances around a CPU target.
  - `max` (number) Most instances (up to 10).
  - `min` (number) Fewest instances (1 to 10).
  - `cpu_percent` (number) CPU target, 10 to 95 (default 70).
- `cpu_millis` (number) CPU per instance, in millicores (1000 = one core).
- `deploy` (bool) Deploy each change at once (default). False only saves settings for the next deployment.
- `edge_cache` (bool) The proxy caches responses that allow it.
- `env` (map(string)) Environment variables (plain values; use staysa_service_secret for secrets).
- `health_path` (string) HTTP path checked before traffic switches to a new deployment.
- `instances` (number) Number of instances (load balanced). Ignored while autoscaling is set: the autoscaler decides.
- `memory_mb` (number) Memory per instance, in MB.
- `metrics_path` (string) Prometheus endpoint the platform scrapes, such as /metrics; empty for none.
- `metrics_port` (number) Port of the metrics endpoint; 0 for the application's port.
- `port` (number) Port the container listens on (web services).
- `public` (bool) Has a public address; false keeps it on the private network only.
- `public_ports` (set(string)) Raw TCP or UDP ports open to the internet, such as "25565/tcp".
- `volume_path` (string) Folder kept across deployments (persistent volume); empty for none. An application with a volume runs one instance.
- `wait_for_live` (bool) Wait until a deployment is live, and fail if it fails (default true).
- `worker` (bool) A background worker: no port, no address.

### Read-Only

- `internal_host` (string) Name on the project's private network.
- `revision` (number) Latest deployment revision.
- `status` (string) Status of the latest deployment.
- `url` (string) Public address.

## Import

Import is supported using the following syntax (`terraform import` works the same way):

```shell
tofu import staysa_service.web shop/web
```
