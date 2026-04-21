# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [Unreleased]

## [v0.1.2] - 2026-04-21

### Changed

- Bumped Go toolchain from `1.25.7` to `1.25.9` to address Go stdlib vulnerabilities.
- Updated Docker builder image to `golang:1.25.9-alpine`.
- Updated CI to use Go `1.25.9` and run `govulncheck`.
- Added Dependabot configuration for `gomod`, Docker, and GitHub Actions updates.

## [v0.1.1] - 2026-02-09

### Added

- Open-source project scaffolding (`LICENSE`, contributing/security/community docs).
- GitHub Actions CI and Docker Hub publish workflows.
- Multi-arch Docker build support (`linux/amd64`, `linux/arm64`).
- Kubernetes deployment, service, and config map examples.
- Prometheus `ScrapeConfig` example for Kubernetes service discovery.
- Grafana dashboard example aligned with exported metrics.

## [v0.1.0] - 2026-02-09

### Added

- Initial GeoWebCache exporter implementation.
- Prometheus metrics coverage for GWC home/runtime values, including memcache handling.
