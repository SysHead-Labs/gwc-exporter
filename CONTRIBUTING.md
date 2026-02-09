# Contributing

## Development Setup

```bash
cd src
go mod tidy
go build ./...
```

## Local Run

```bash
cd src
go run gwc-exporter.go -target.url "http://geowebcache:8080/geowebcache"
```

## Pull Request Guidelines

- Keep changes focused and small.
- Update `README.md` when behavior, flags, or deployment flow changes.
- Update `grafana/gwc-exporter-dashboard.json` and/or `prometheus/scrape-gwc-example.yaml` if relevant.
- Add or update tests when parser logic changes.
- Ensure code is formatted (`gofmt`) and builds cleanly.

## Commit Messages

- Use clear, imperative messages.
- Mention the main affected area (e.g. `exporter:`, `docker:`, `grafana:`).
