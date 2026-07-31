# Operations

## Deployment model

Run one Hercules Instance per trust and resource boundary. Packages within an Instance share one DuckDB catalog, one serialized connection, process credentials, memory limit, thread budget, and snapshot failure domain. Separate unrelated or differently trusted workloads into separate processes or containers.

The released container runs as non-root and supports Linux amd64 and arm64. Mount configuration, Packages, and local Source data read-only where possible. Its home and working directories are writable for allowlisted DuckDB extensions and temporary state; mount an explicitly owned writable directory for a file-backed database, persistent extension cache, or DuckDB temporary directory.

## Endpoints

| Path | Meaning |
| --- | --- |
| `/metrics` | Current immutable configured snapshot plus Go, process, collection, and Source refresh metrics. |
| `/healthz` | Process and HTTP handler are alive. It does not assert Source freshness. |
| `/readyz` | The initial complete Metric Snapshot exists and shutdown has not begun. |
| `/` | Redirects to `/metrics`. |

Use `/healthz` for liveness and `/readyz` for readiness. Do not restart solely because a later collection fails: the last-known-good snapshot is intentionally still served.

## Essential alerts

Alert on:

- `hercules_ready == 0`;
- growth in `hercules_collection_errors_total`;
- `hercules_snapshot_age_seconds` above a workload-specific multiple of `collectionIntervalSeconds`;
- `hercules_source_refresh_failures_total` growth;
- old `hercules_source_last_success_timestamp_seconds` for Sources that must refresh;
- sustained `hercules_collection_duration_seconds` near `collectionTimeoutSeconds`;
- process memory approaching the container/cgroup limit despite the DuckDB memory budget.

`hercules_last_successful_collection_timestamp_seconds`, `hercules_snapshot_rows`, and `hercules_snapshot_series` are useful for dashboards and capacity planning. The `hercules_build_info{version=...}` gauge identifies deployed versions.

## Startup and rollout

Use this sequence in CI or deployment automation:

```sh
hercules check-config --config /etc/hercules/hercules.yml
hercules check --config /etc/hercules/hercules.yml
hercules run --config /etc/hercules/hercules.yml
```

`check` may read remote Sources, install allowlisted extensions, and execute Package SQL. Run it with the same filesystem, network, environment, and credentials as production.

There is no in-process configuration reload in 0.9. Roll out a replacement process, wait for `/readyz`, then move traffic. This prevents an invalid candidate from destroying the current last-known-good snapshot.

## Shutdown

SIGINT and SIGTERM start a bounded graceful shutdown. Readiness turns off, refresh and collection contexts cancel, the HTTP server drains, workers join, DuckDB closes, and an explicitly ephemeral file database is removed. If the deadline expires, the HTTP server is forced closed and the caller returns an error instead of blocking indefinitely; in-process cleanup continues while the process remains alive. Give the process at least 15 seconds of termination grace plus any platform scheduling delay.

## Capacity and cardinality

Set `maxRowsPerMetric` and `maxSeries` well below the process/container capacity. Histogram series per label set equal the configured finite bucket count plus the implicit `+Inf` bucket, `_sum`, and `_count`; summaries equal objective count plus `_sum` and `_count`. A high-cardinality label can exhaust the snapshot series budget even when the SQL row count looks modest.

Exact summaries retain and sort all observation values for each label group during a collection. Large raw-observation histograms perform a bucket comparison per observation. Pre-aggregate in DuckDB, narrow Source data, or prefer gauges when cumulative histogram/summary semantics are unnecessary.

The DuckDB memory limit is not a complete process limit. Leave headroom for Go, Prometheus metric objects, native bindings, operating-system caching, and loaded extensions.

## Exposure and transport security

The built-in server has bounded headers and read, header, write, and idle timeouts. It can load a configuration-relative certificate/key pair and require a constant-time-compared bearer token on `/metrics`. `/healthz` and `/readyz` remain unauthenticated for orchestrator probes. Keep the loopback default for a local collector, or bind to a private interface and use network policy. Prefer a reverse proxy or service mesh when you need identity-aware authorization, centralized certificate rotation, rate limiting, or access logs.

## Persistent databases

Use `db: :memory:` for a purely derived catalog. For a persistent path, set `ephemeralDB: false`, mount durable storage, and plan backup/recovery according to the Source data that cannot be regenerated. Relative database and temporary-directory paths are configuration-relative.

## Debugging

1. Run `print-effective-config` and verify resolved defaults; package variable values are intentionally redacted.
2. Run `check-config` to isolate schema, variable, dependency, or trust-policy failures.
3. Run `check` to isolate DuckDB extension, macro, Source, or metric query failures.
4. Inspect collection and Source self-metrics for later failures.
5. Set `DEBUG=true`; reserve `TRACE=true` for short diagnostic windows.

Hercules joins related cleanup errors, so the final log can contain more than one shutdown cause. Startup errors are wrapped with the phase and object name that failed.
