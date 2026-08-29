# syntax=docker/dockerfile:1
ARG GO_VERSION=1.26.6

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /out && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
      -ldflags="-s -w \
        -X github.com/c3-oss/prosa-webp-widgets/internal/buildinfo.Version=${VERSION} \
        -X github.com/c3-oss/prosa-webp-widgets/internal/buildinfo.Commit=${COMMIT} \
        -X github.com/c3-oss/prosa-webp-widgets/internal/buildinfo.BuildDate=${BUILD_DATE}" \
      -o /out/ ./cmd/...

FROM debian:bookworm-slim AS prosa-webp-widgets
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates chromium fonts-dejavu-core \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/prosa-webp-widgets /usr/local/bin/prosa-webp-widgets
ENTRYPOINT ["/usr/local/bin/prosa-webp-widgets"]
