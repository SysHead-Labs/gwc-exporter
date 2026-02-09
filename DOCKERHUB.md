# GeoWebCache Exporter

Prometheus exporter for GeoWebCache runtime / home page cache statistics.

Source code (GitHub):
- https://github.com/SysHead-Labs/gwc-exporter

Docker image:
- `syshead/gwc-exporter`

Supported platforms:
- `linux/amd64`
- `linux/arm64`

## Quick start (Docker)

```bash
docker pull syshead/gwc-exporter:latest

docker run --rm -p 9109:9109 \
  -e GWC_TARGET_URL="http://geowebcache:8080/geowebcache" \
  syshead/gwc-exporter:latest
```

Metrics endpoint:
- `http://127.0.0.1:9109/metrics`

## Configuration (environment variables)

- `GWC_TARGET_URL`
  - Default: `http://127.0.0.1:8080/geowebcache`
- `GWC_WEB_LISTEN_ADDRESS`
  - Default: `:9109`
- `GWC_WEB_TELEMETRY_PATH`
  - Default: `/metrics`
- `GWC_SCRAPE_TIMEOUT`
  - Default: `5s`

Example: custom listen address

```bash
docker run --rm -p 9200:9200 \
  -e GWC_TARGET_URL="http://geowebcache:8080/geowebcache" \
  -e GWC_WEB_LISTEN_ADDRESS=":9200" \
  syshead/gwc-exporter:latest
```

## Kubernetes

Manifests are in the GitHub repository:
- ConfigMap: https://github.com/SysHead-Labs/gwc-exporter/blob/main/kubernetes/configmap.yaml
- Deployment: https://github.com/SysHead-Labs/gwc-exporter/blob/main/kubernetes/deployment.yaml
- Service: https://github.com/SysHead-Labs/gwc-exporter/blob/main/kubernetes/service.yaml

Apply (example):

```bash
kubectl apply -f https://github.com/SysHead-Labs/gwc-exporter/raw/main/kubernetes/configmap.yaml
kubectl apply -f https://github.com/SysHead-Labs/gwc-exporter/raw/main/kubernetes/deployment.yaml
kubectl apply -f https://github.com/SysHead-Labs/gwc-exporter/raw/main/kubernetes/service.yaml
```

Prometheus Operator ScrapeConfig example:
- `https://github.com/SysHead-Labs/gwc-exporter/blob/main/prometheus/scrape-gwc-example.yaml`

Grafana Dashboard example:
- `https://github.com/SysHead-Labs/gwc-exporter/blob/main/grafana/gwc-exporter-dashboard.json`

## Tags

- `latest`: latest published release
- `X.Y.Z`: immutable release tags (published from Git tags `vX.Y.Z`)