# Package authoring

## Trust model

A Package can execute arbitrary DuckDB SQL, read process-visible files and network locations, and request native extensions. Treat a Package exactly like executable code: review it, version it, pin remote content, and grant the Hercules process only the credentials it needs.

Digest-pinned remote Package bytes may be stored in the content-addressed cache configured by the Instance. Online loads atomically populate it after verification; offline mode reads it without rewriting. This cache covers the Package document only, not remote Source data referenced by that document.

## Header and identity

```yaml
apiVersion: hercules.dev/v1alpha1
kind: Package
name: orders
version: "1.0"
minHerculesVersion: "0.9.0"
dependencies: [shared]
```

`apiVersion`, `kind`, `name`, and `version` are required. `minHerculesVersion` uses semantic versioning. Dependencies name other configured Packages; they control initialization order but do not download packages. Package names must be unique.

The rendered metric family is:

```text
<reference metricPrefix><package name with hyphens replaced by underscores>_<definition name>
```

## Variables

```yaml
variables:
  table: events

metrics:
  gauge:
    - name: rows
      help: Event rows.
      sql: select count(*) as value from {{table}}
```

Package defaults are overlaid by reference variables from the Instance configuration. Reference values beginning with `$` resolve environment variables. Expansion applies to macro SQL, Source locations/SQL, metric names, help, and SQL. Undefined variables fail compilation.

Variables are textual substitution, not SQL parameters. Never pass untrusted end-user input into them.

## Extensions

```yaml
extensions:
  core:
    - name: httpfs
  community:
    - name: example
```

Extensions initialize before macros and Sources. Both lists require a top-level security allowlist; community extensions require the separate community opt-in. Extension installation can contact DuckDB repositories and loads native code into the process.

## Macros

```yaml
macros:
  - name: normalize_state
    sql: normalize_state(value) AS (lower(trim(value)));
```

The `sql` value is the body following DuckDB's `CREATE OR REPLACE MACRO`. `name` is required, must match the identifier declared at the start of `sql` (case-insensitively), and is used for diagnostics and catalog collision checks. Macros initialize in package order and declaration order.

## Sources

```yaml
sources:
  - name: events
    type: parquet
    source: ./data/events.parquet
    materialize: true
    refreshIntervalSeconds: 300
```

Supported Source types are:

| Type | Generated query |
| --- | --- |
| `parquet` | `read_parquet(...)` |
| `csv` | `read_csv_auto(...)` |
| `json` | `read_json_auto(...)` |
| `sql` | Exact Package SQL |

Local relative paths resolve from the Package directory. A materialized Source becomes a table. A non-materialized Source becomes a view and cannot have a refresh interval. Zero disables recurring refresh. Source names are global DuckDB catalog names and must not collide across Packages.

Dependencies between Sources are expressed through declaration and Package dependency order. Hercules does not parse SQL to infer a dependency graph.

## Metrics

Every metric requires `name`, `help`, and `sql`; `enabled` defaults to true. Declare `labels` explicitly when possible:

```yaml
metrics:
  gauge:
    - name: queue_depth
      help: Current queue depth.
      labels: [queue]
      sql: select queue, count(*) as value from events group by queue

  counter:
    - name: events_processed_total
      help: Cumulative processed events.
      labels: [queue]
      sql: select queue, max(sequence) as value from events group by queue

  histogram:
    - name: duration_seconds
      help: Event processing duration.
      labels: [queue]
      sql: select queue, duration_ms / 1000.0 as value from events
      histogramMode: observations
      buckets: [0.01, 0.05, 0.1, 0.5, 1, 5]

  summary:
    - name: payload_bytes
      help: Payload size quantiles.
      labels: [queue]
      sql: select queue, payload_bytes as value from events
      objectives:
        - quantile: 0.5
          error: 0.05
        - quantile: 0.9
          error: 0.01
        - quantile: 0.99
          error: 0.001
```

Metric SQL must return exactly one numeric value column plus the declared labels. Static Instance labels are merged later. A collision between a query label and a static label is an error. Null numeric values, non-finite values, invalid label types, excessive rows/series, and duplicate gauge label sets fail the whole collection.

Counters, histogram buckets, and the count/sum portions of summaries are modeled as cumulative Prometheus series. Author SQL against cumulative source data; do not query a shrinking window and label it as a counter.

Counter names must end in `_total`. Histogram and summary base names must not end in `_bucket`, `_sum`, `_count`, or `_total`; those suffixes are emitted by Prometheus. The `hercules_`, `go_`, and `process_` prefixes are reserved for Instance telemetry. A Metric Definition supports at most 20 dynamic labels, a histogram at most 100 finite strictly increasing buckets, and a summary at most 20 objectives.

## Dependencies and collisions

Hercules topologically orders configured Packages. Missing dependencies and cycles fail. After ordering, it rejects duplicate Source names, named macro collisions, and duplicate fully rendered metric names. Namespacing is explicit through Package names and optional reference prefixes rather than hidden DuckDB schemas.

## Validation workflow

```sh
hercules package-digest packages/orders/1.0.yml
hercules check-config --config hercules.yml
hercules check --config hercules.yml
```

The first command pins content, the second checks schemas/dependencies without executing SQL, and the third validates real DuckDB objects and the initial snapshot. Package files can use [`schemas/package.json`](../schemas/package.json) in editors and CI.

## Compatibility policy

The current API is `v1alpha1`. Additive fields may arrive within the alpha. Breaking schema or semantic changes require a new `apiVersion`; the loader does not silently reinterpret an unknown version. Set `minHerculesVersion` when relying on behavior introduced after 0.9.
