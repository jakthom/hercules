<h1 align="center">
  <img src="./assets/hercules.png" alt="Hercules" width="20%">
  <br>
  Hercules
</h1>

<p align="center"><strong>Write DuckDB SQL. Export stable Prometheus metrics.</strong></p>

<p align="center">
  <a href="https://github.com/jakthom/hercules/actions/workflows/test.yml"><img src="https://github.com/jakthom/hercules/actions/workflows/test.yml/badge.svg" alt="CI"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT license"></a>
</p>

Hercules embeds DuckDB, turns local files, object-store objects, HTTP data, database scans, and SQL transformations into named sources, then evaluates SQL metric definitions into an immutable Prometheus snapshot. Prometheus scrapes never execute DuckDB queries.

Hercules 0.9 is an alpha release. Its configuration and package APIs are versioned, but `hercules.dev/v1alpha1` may still change before 1.0. Packages are trusted code, not sandboxed plugins.

## Why Hercules

- Use DuckDB's query engine and readers at the edge instead of writing a custom exporter for every external data source.
- Join, normalize, enrich, and aggregate data in SQL before it reaches Prometheus.
- Group reusable extensions, macros, sources, and metrics into dependency-aware packages.
- Serve concurrent scrapes from one last-known-good snapshot with no registry mutation or scrape-time query latency.
- Put explicit limits around query time, DuckDB memory and threads, rows, series, label sizes, remote packages, and native extensions.

## Quick start

Prerequisites are Go 1.26.5, a C toolchain, and `make`.

```sh
git clone https://github.com/jakthom/hercules.git
cd hercules
make build
./hercules check --config hercules.yml
./hercules run --config hercules.yml
```

The repository example listens on `0.0.0.0:9100`. Open:

- `http://localhost:9100/metrics` for Prometheus/OpenMetrics output
- `http://localhost:9100/healthz` for process liveness
- `http://localhost:9100/readyz` for snapshot readiness

The safe default listen address is `127.0.0.1`; the example explicitly opts into all interfaces for container use.

## Minimal configuration

```yaml
version: 1
name: local
listenAddress: 127.0.0.1
port: 9999

globalLabels:
  environment: development

metrics:
  gauge:
    - name: answer
      help: A minimal scalar metric.
      sql: select 42 as value
```

Validate it without opening DuckDB:

```sh
hercules check-config --config hercules.yml
```

Run a full preflight that opens DuckDB, initializes every extension, macro, and source, and evaluates the initial snapshot:

```sh
hercules check --config hercules.yml
```

## Metric query contract

Every enabled metric definition executes during background collection. A result must contain one numeric value column and zero or more label columns.

```yaml
metrics:
  gauge:
    - name: queued_jobs
      help: Jobs currently queued by tenant.
      labels: [tenant]
      sql: select tenant, count(*) as value from jobs where state = 'queued' group by tenant
```

Prefer a column named `value` and declare `labels` explicitly. `val`, `v`, a sole column, and a single function-like column are accepted for compatibility. Hercules rejects missing, duplicate, invalid, colliding, or undeclared columns and non-finite values.

Metric types have these snapshot semantics:

- Gauge: one row per unique label set. Duplicate label sets fail the collection.
- Counter: non-negative rows with the same label set are summed. SQL must represent a cumulative value that does not decrease over the life of the observed system.
- Histogram: each row is a raw observation. Configured bucket counts, sum, and count are computed exactly for the snapshot.
- Summary: each row is a raw observation. Configured quantiles, sum, and count are computed exactly for the snapshot. The `error` field documents the intended objective and preserves Prometheus-style configuration, but exact snapshot calculation does not use an approximation error.

If any query or conversion fails, Hercules keeps serving the entire previous snapshot and increments `hercules_collection_errors_total`. It never publishes a partially updated snapshot.

## Packages

A Package is a versioned, trusted YAML document:

```yaml
apiVersion: hercules.dev/v1alpha1
kind: Package
name: example
version: "1.0"

variables:
  table: events

sources:
  - name: events
    type: parquet
    source: ./events.parquet
    materialize: true

metrics:
  gauge:
    - name: rows
      help: Rows in the event source.
      sql: select count(*) as value from {{table}}
```

Package references may override variables, add a metric prefix, and pin content with SHA-256. Relative source paths resolve from the package file, not the process working directory. Dependencies are topologically sorted; missing dependencies, cycles, and cross-package source, macro, or metric name collisions fail startup.

Remote packages are disabled by default. When enabled, hosts must be allowlisted, HTTPS is mandatory, response size is bounded, and SHA-256 is required by default. Generate a digest with:

```sh
hercules package-digest path/to/package.yml
```

An explicit `security.packageCacheDirectory` stores verified remote Package bytes by digest. `security.offline: true` serves those Package references only from the cache, which may be mounted read-only; Source URLs inside a Package retain their own network behavior.

See [Package authoring](docs/packages.md) for the complete contract.

## Container

```sh
make docker-build
make docker-run
```

Release images are published for `linux/amd64` and `linux/arm64` at `ghcr.io/jakthom/hercules`. The image runs as a non-root user and expects `/etc/hercules/hercules.yml`; mount packages and data beneath `/etc/hercules` so relative paths remain stable.

For direct production exposure, configure a certificate/key pair and a bearer token file. File paths are configuration-relative:

```yaml
http:
  tlsCertificateFile: secrets/tls.crt
  tlsKeyFile: secrets/tls.key
  bearerTokenFile: secrets/metrics.token
```

Bearer authentication protects `/metrics`; health and readiness remain available for orchestrator probes. A reverse proxy or service mesh is still preferred when centralized identity, authorization, certificate rotation, or audit policy is required.

## CLI

```text
hercules [run] [--config PATH]
hercules check-config [--config PATH]
hercules check [--config PATH]
hercules print-effective-config [--config PATH]
hercules package-digest PATH
hercules version
hercules help
```

`print-effective-config` renders defaults and redacts package variable values. `DEBUG=true` and `TRACE=true` adjust log verbosity. Labels or package variables whose value starts with `$` read the named environment variable and fail closed if it is missing.

## Documentation

- [Architecture](docs/architecture.md)
- [Configuration reference](docs/configuration.md)
- [Package authoring](docs/packages.md)
- [Operations](docs/operations.md)
- [Migrating from 0.8 to 0.9](docs/migrating-to-0.9.md)
- [Security policy and trust model](SECURITY.md)
- [Contributing](CONTRIBUTING.md)
- [Architecture decisions](docs/adr/)

JSON Schemas live in [`schemas/`](schemas/) and are wired into the repository's VS Code settings.

## Current boundaries

Hercules intentionally does not provide a Package sandbox, SQL authorization layer, multi-tenant isolation, or user/role authorization system. It offers optional TLS and one bearer token for `/metrics`; use an external identity-aware proxy for richer policy. Run it with least-privilege filesystem/network credentials. Configuration replacement currently uses a controlled process restart so a candidate Instance can complete its initial Metric Snapshot before traffic moves.

## License

Hercules is available under the [MIT License](LICENSE).
