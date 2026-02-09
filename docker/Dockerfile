FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/gwc-exporter.go ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/gwc-exporter gwc-exporter.go

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/gwc-exporter /gwc-exporter

ENV GWC_TARGET_URL=http://127.0.0.1:8080/geowebcache
ENV GWC_WEB_LISTEN_ADDRESS=:9109
ENV GWC_WEB_TELEMETRY_PATH=/metrics
ENV GWC_SCRAPE_TIMEOUT=5s

ENTRYPOINT ["/gwc-exporter"]
