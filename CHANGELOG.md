# Changelog

All notable changes to Hercules are documented here. The project follows semantic versioning while the package API remains alpha.

## [Unreleased]

## [0.9.0] - 2026-07-19

### Added

- Immutable, atomically published, last-known-good Metric Snapshots.
- A serialized owned DuckDB Engine with memory, thread, timeout, row, and series budgets.
- Strict versioned configuration and `hercules.dev/v1alpha1` Package validation.
- Dependency-ordered Packages, typed variables, compatibility checks, collision detection, Package-relative Source paths, remote host/digest policy, and a digest CLI.
- Supervised non-overlapping Source refreshes with jitter, backoff, cancellation, joining, and Prometheus telemetry.
- Correct SQL value handling for gauges, counters, raw-observation histograms, and exact snapshot summaries.
- Private Prometheus registry, readiness/liveness endpoints, build/collection/freshness metrics, and bounded HTTP server timeouts.
- Optional TLS 1.2+ serving and constant-time bearer-token authentication for `/metrics`.
- Static and full-preflight CLI commands, effective configuration output with variable redaction, signal-aware shutdown, and reliable exit codes.
- Race, concurrency, lifecycle, semantic, runtime HTTP, CLI, vet, lint, vulnerability, and example-preflight coverage.
- Multi-architecture non-root containers, Linux release archives and checksums, dependency automation, schemas, ADRs, and operator/author documentation.

### Changed

- Configuration now fails closed on unknown fields and invalid or missing required values.
- Remote Packages and DuckDB extensions are denied by default.
- The default listener is loopback, the default database is in-memory, and explicit resource limits are applied.
- Prometheus scrapes no longer execute SQL or mutate global collector registration.
- Go and direct dependencies were upgraded to current supported release lines.

### Removed

- Global registry replacement, scrape-time SQL collection, unsupervised refresh goroutines, hidden database ownership, permissive Viper fallback behavior, and obsolete helper packages.
- Unused pre-0.9 compatibility helpers that bypassed contexts or duplicated the new lifecycle APIs.

See [Migrating from 0.8 to 0.9](docs/migrating-to-0.9.md) for the configuration, Package, metric, CLI, and deployment changes required by this alpha release.

[Unreleased]: https://github.com/jakthom/hercules/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/jakthom/hercules/compare/v0.8.0...v0.9.0
