# syntax=docker/dockerfile:1.12@sha256:93bfd3b68c109427185cd78b4779fc82b484b0b7618e36d0f104d4d801e66d25
FROM golang:1.26.5-bookworm@sha256:1ecb7edf62a0408027bd5729dfd6b1b8766e578e8df93995b225dfd0944eb651 AS build

ARG VERSION=dev
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/hercules ./cmd/hercules
RUN mkdir -p /out/home/.duckdb /out/state

FROM gcr.io/distroless/cc-debian12:nonroot@sha256:66aa873a4a14fb164aa01296058efd8253744606d72715e45acface073359faa

LABEL org.opencontainers.image.title="Hercules" \
      org.opencontainers.image.description="Expose DuckDB SQL results as immutable Prometheus metric snapshots" \
      org.opencontainers.image.source="https://github.com/jakthom/hercules" \
      org.opencontainers.image.licenses="MIT"

ENV HOME=/home/nonroot
COPY --from=build --chown=nonroot:nonroot /out/home /home/nonroot
COPY --from=build --chown=nonroot:nonroot /out/state /var/lib/hercules
WORKDIR /var/lib/hercules
COPY --from=build --chown=nonroot:nonroot /out/hercules /usr/local/bin/hercules

EXPOSE 9999
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/hercules"]
CMD ["run", "--config", "/etc/hercules/hercules.yml"]
