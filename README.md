# GeoWebCache Prometheus Exporter

Exports runtime values from the GeoWebCache home/status page as Prometheus metrics.

## Project Metadata

- License: `MIT` (see `LICENSE`)
- Contributing guide: `CONTRIBUTING.md`
- Code of conduct: `CODE_OF_CONDUCT.md`
- Security policy: `SECURITY.md`
- Changelog: `CHANGELOG.md`

## Build Manually

```bash
cd src
go mod tidy
go build -ldflags="-s -w" -o gwc-exporter gwc-exporter.go
```

## Run Manually

```bash
cd src
./gwc-exporter \
  -target.url "http://geowebcache:8080/geowebcache" \
  -web.listen-address ":9109" \
  -web.telemetry-path "/metrics"
```

Open metrics at:

```text
http://127.0.0.1:9109/metrics
```

## Build Docker Image

```bash
docker build -f docker/Dockerfile -t gwc-exporter:local .
```

## Build Multi-Arch Docker Image (amd64 + arm64)

```bash
docker buildx create --use --name gwc-builder
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f docker/Dockerfile \
  -t syshead/gwc-exporter:latest \
  --push \
  .
```

If you want a local image for only your current platform:

```bash
docker buildx build --load -f docker/Dockerfile -t gwc-exporter:local .
```

## GitHub Actions: Docker Hub Publish

Folder structure:

```text
.github/
  workflows/
    dockerhub-publish.yml
```

Workflow behavior:

- Push to `main`: publishes a multi-arch image and updates `latest`
- Push tag like `v1.2.3`: publishes tag `v1.2.3`
- Manual run (`workflow_dispatch`) is supported

Required GitHub repository secrets:

- `DOCKERHUB_TOKEN` (Docker Hub access token)

Published image name:

```text
syshead/gwc-exporter
```

## Run Docker Image

```bash
docker run --rm -p 9109:9109 \
  -e GWC_TARGET_URL="http://geowebcache:8080/geowebcache" \
  gwc-exporter:local
```

Then scrape:

```text
http://127.0.0.1:9109/metrics
```

## Run From Docker Hub

```bash
docker pull syshead/gwc-exporter:latest
docker run --rm -p 9109:9109 \
  -e GWC_TARGET_URL="http://geowebcache:8080/geowebcache" \
  syshead/gwc-exporter:latest
```

## Environment Variables

These are ideal for Kubernetes ConfigMap/Deployment env injection.

- `GWC_TARGET_URL` default: `http://127.0.0.1:8080/geowebcache`
- `GWC_WEB_LISTEN_ADDRESS` default: `:9109`
- `GWC_WEB_TELEMETRY_PATH` default: `/metrics`
- `GWC_SCRAPE_TIMEOUT` default: `5s`

Flags are still supported and override env vars when explicitly provided.

## Kubernetes ConfigMap Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gwc-exporter-config
data:
  GWC_TARGET_URL: "http://geowebcache:8080/geowebcache"
  GWC_WEB_LISTEN_ADDRESS: ":9109"
  GWC_WEB_TELEMETRY_PATH: "/metrics"
  GWC_SCRAPE_TIMEOUT: "5s"
```

## Kubernetes Deployment Example

Manifests included in this repo:

- `kubernetes/configmap.yaml`
- `kubernetes/deployment.yaml`
- `kubernetes/service.yaml`

Apply:

```bash
kubectl apply -f kubernetes/configmap.yaml
kubectl apply -f kubernetes/deployment.yaml
kubectl apply -f kubernetes/service.yaml
```

These are aligned with:

- namespace: `monitoring`
- Service name: `gwc-exporter`
- Service port name: `metrics`

## Prometheus ScrapeConfig Example

Use:

- `prometheus/scrape-gwc-example.yaml`

This ScrapeConfig discovers the `gwc-exporter` Kubernetes Service endpoints and scrapes `/metrics`.

## Grafana Dashboard Example

Import dashboard JSON:

- `grafana/gwc-exporter-dashboard.json`

Dashboard queries are aligned with all currently exported `gwc_*` metrics.
