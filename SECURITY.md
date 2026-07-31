# Security policy

## Supported versions

Hercules is pre-1.0. Security fixes are applied to the latest released minor version. Older alpha versions should be upgraded rather than operated indefinitely.

## Reporting a vulnerability

Use GitHub's private security advisory flow for `jakthom/hercules`. Do not open a public issue containing exploit details, credentials, or sensitive deployment information. Include the affected version, deployment model, reproduction, impact, and any proposed mitigation. Maintainers should acknowledge a report within seven days and coordinate disclosure after a fix or mitigation is available.

## Trust model

Hercules Packages are trusted code. Package SQL can read any file or network resource accessible to DuckDB and can consume substantial compute. DuckDB extensions load native code into the Hercules process. Strict YAML and an allowlist prevent accidental activation; they do not sandbox malicious SQL or native extensions.

Only run reviewed Packages. Pin remote Packages by SHA-256, use exact allowed hostnames, leave community extensions disabled unless required, and grant the process least-privilege filesystem, network, object-store, and database credentials.

## Deployment requirements

- Keep the default loopback listener unless remote scraping is intentional.
- Use the built-in TLS and `/metrics` bearer token for simple direct exposure, or place the Instance behind an identity-aware TLS proxy. Always apply network policy.
- Run as an unprivileged operating-system user. The release container already uses a non-root identity.
- Mount Package and Source files read-only where possible.
- Set an outer process/container memory and CPU limit in addition to DuckDB's internal limits.
- Keep `collectionTimeoutSeconds`, `maxRowsPerMetric`, and `maxSeries` bounded.
- Do not expose a writable Package directory to less-trusted users.
- Avoid secrets in Package files and SQL. Use environment-backed reference variables, protect process environments, and remember that downstream Source systems may log queries.
- Monitor snapshot age and collection/refresh errors; last-known-good output can be stale.

## Remote Package controls

Remote loading is off by default. When enabled, Hercules enforces allowed hostnames and HTTPS before following every redirect hop, successful status codes, response-size limits, and—by default—an exact SHA-256. The digest provides content integrity, not publisher identity. Distribute expected digests through a separately trusted configuration/release channel.

When configured, verified Package bytes are written to a content-addressed cache with restrictive permissions and atomic replacement. Offline mode requires a digest and reads only this cache. Protect the cache directory from less-trusted writers; content is reverified before compilation, but availability can still be disrupted.

## Out of scope guarantees

Hercules is not a multi-tenant SQL service, package sandbox, secret manager, network proxy, or user/role policy engine. Its bearer token is one shared secret protecting `/metrics`, not a general authorization system. It does not prove that a query is read-only, stop a Package from calling DuckDB functions with side effects, or isolate one Package's catalog and credentials from another Package in the same Instance.

## Supply chain

CI runs race tests, vet, a strict lint profile, a reachable-vulnerability scan, a real example preflight, and a container build. Releases are tag-driven, checksum Linux amd64/arm64 archives, and publish multi-architecture non-root images with provenance and SBOM attestations. Dependabot tracks Go, GitHub Actions, and container dependencies.
