# Configuration reference

Hercules reads one strict YAML configuration. Unknown keys, invalid values, missing environment references, invalid packages, and catalog collisions are errors. Relative paths resolve from the directory containing the configuration file.

## Top-level fields

| Field | Default | Meaning |
| --- | --- | --- |
| `version` | required | Must be `1`. |
| `name` | `hercules` | Instance name added as the reserved `hercules` label. |
| `debug` | `false` | Enable debug logging; `DEBUG`/`TRACE` environment flags take precedence. |
| `listenAddress` | `127.0.0.1` | IP address or `localhost`. Use `0.0.0.0` only when exposure is intended. |
| `port` | `9999` | HTTP port, 1–65535. |
| `db` | `:memory:` | DuckDB path; a relative file is configuration-relative. |
| `ephemeralDB` | `false` | Remove a non-memory database file during orderly shutdown. |
| `duckdb` | bounded defaults | DuckDB memory, worker thread, and temporary directory settings. |
| `http` | no TLS/authentication | Optional TLS certificate/key and `/metrics` bearer token files. |
| `collectionIntervalSeconds` | `15` | Period between complete Metric Snapshot collections. |
| `collectionTimeoutSeconds` | `10` | Deadline for initialization, each collection, and each Source refresh. |
| `maxRowsPerMetric` | `100000` | Maximum rows one Metric Definition may return. |
| `maxSeries` | `100000` | Maximum Prometheus series represented by one complete snapshot. |
| `globalLabels` | `{}` | Static string labels merged into every configured metric. |
| `packages` | `[]` | Package references compiled before the inline core configuration. |
| `extensions` | empty | DuckDB extensions owned by the implicit core Package. |
| `macros` | empty | DuckDB macros owned by the implicit core Package. |
| `sources` | empty | Sources owned by the implicit core Package. |
| `metrics` | empty | Metric Definitions owned by the implicit core Package. |
| `security` | deny by default | Remote Package and DuckDB extension trust policy. |

## DuckDB resources

```yaml
duckdb:
  memoryLimit: 1GB
  threads: 4
  tempDirectory: ./tmp
```

`memoryLimit` accepts a positive size with `B`, `KB`, `MB`, `GB`, `TB`, or IEC equivalents. `threads` must be positive. A relative `tempDirectory` is configuration-relative. Ensure it exists and is writable by the process. These are per-Instance budgets; Source readers may also consume operating-system page cache and network buffers.

## Labels and environment values

```yaml
globalLabels:
  environment: production
  region: $REGION
```

Label names use Prometheus identifier syntax. Values beginning with `$` resolve the remainder as an environment variable; a missing variable is an error. The `hercules` label is reserved. The legacy list-of-maps form is still decoded for migration, but the object form is canonical.

## HTTP transport and authentication

```yaml
http:
  tlsCertificateFile: secrets/tls.crt
  tlsKeyFile: secrets/tls.key
  bearerTokenFile: secrets/metrics.token
```

Certificate and key must be configured together and parse as an X.509 key pair during full preflight/startup. TLS uses a minimum of TLS 1.2. The token file must contain one non-empty whitespace-free token of at most 8 KiB. Hercules stores only a SHA-256 comparison value after startup and accepts `Authorization: Bearer <token>` on `/metrics`. Health and readiness are intentionally unauthenticated. All paths are configuration-relative; rotate them with a controlled Instance restart.

## Package references

```yaml
packages:
  - package: ./packages/orders/1.0.yml
    metricPrefix: edge_
    variables:
      table: orders
      token: $ORDERS_TOKEN
    sha256: sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

`metricPrefix` is prepended before the rendered package and definition name. Package variable values are sensitive and are redacted by `print-effective-config`. `sha256` may omit the `sha256:` prefix. It is optional for local packages and required by default for remote packages.

## Security policy

```yaml
security:
  allowRemotePackages: false
  requireRemotePackageSHA256: true
  allowedPackageHosts: []
  packageCacheDirectory: ""
  offline: false
  allowCommunityExtensions: false
  allowedExtensions: []
  maxRemotePackageBytes: 2097152
```

- Remote Packages require `allowRemotePackages`, an exact hostname in `allowedPackageHosts`, HTTPS, a bounded successful response, and normally a digest.
- A non-empty `packageCacheDirectory` stores verified remote Package bytes by digest using atomic replacement. `offline: true` requires that directory and resolves remote references only from it.
- Every DuckDB extension, including a core extension, must be listed in `allowedExtensions`.
- Community extensions additionally require `allowCommunityExtensions`.
- Redirect destinations are checked again for HTTPS and allowlisted hostname.

## Inline core package

Top-level `extensions`, `macros`, `sources`, and `metrics` form an implicit Package named `core`. It participates in catalog collision checking and renders metric names with `core_`. Relative paths used by inline non-SQL Sources resolve from the configuration directory. Inline configuration is convenient for one-off definitions; reusable definitions should live in a versioned Package.

## Example

See the repository [`hercules.yml`](../hercules.yml), the JSON Schema at [`schemas/config.json`](../schemas/config.json), and [Package authoring](packages.md).
