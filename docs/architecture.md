# Architecture

## Purpose

Hercules is a single-process SQL metrics exporter. One Instance owns one DuckDB engine, compiles operator configuration and trusted Packages into a deterministic startup plan, initializes Sources, evaluates Metric Definitions in the background, and exposes exactly one immutable last-known-good Metric Snapshot.

The design keeps DuckDB work out of the Prometheus scrape path. This is the central architectural seam: collection can be slow or fail without making concurrent scrapes contend for query execution or observe half of a Metric Snapshot update.

## Module map

| Module | Responsibility | Does not own |
| --- | --- | --- |
| `cmd/hercules` | CLI parsing, signals, logging, exit codes | DuckDB, HTTP internals |
| `internal/runtime` | Composition root, startup phases, HTTP server, lifecycle | SQL result semantics |
| `pkg/config` | Strict top-level decoding, defaults, validation, environment labels | Package resolution |
| `pkg/package` | Trust-aware package resolution, variables, compatibility, dependency ordering, collision checks | Runtime execution |
| `pkg/engine` | One serialized DuckDB connection and its resource settings | Scheduling |
| `pkg/db` | Context-aware query conversion, extension and macro initialization | Connection ownership |
| `pkg/source` | Source SQL generation, ordered initialization, refresh supervision and status metrics | Metric collection |
| `pkg/metric` | Metric schema, type validation, naming, metadata, variable expansion | Prometheus collectors |
| `pkg/exporter` | Query evaluation, result validation, type conversion, immutable snapshots, collection telemetry | HTTP server |

The composition root imports the implementations and wires their narrow interfaces. Lower packages do not import `internal/runtime`, and `db.Executor` is the primary seam used by the source and exporter implementations in tests.

## Startup sequence

```text
configuration file
       |
       v
strict decode + defaults + validation
       |
       v
resolve package references and variables
       |
       v
compatibility + dependencies + catalog collision checks
       |
       v
open one DuckDB engine and apply resource limits
       |
       v
extensions -> macros -> sources -> initial metric snapshot
       |
       v
private Prometheus registry + HTTP server
       |
       v
ready
```

Compilation finishes before DuckDB is opened. After the engine exists, initialization is deliberately ordered because later phases may depend on objects from earlier phases. Any failure aborts startup, closes the engine, and leaves no partially ready HTTP endpoint.

`check-config` stops after package compilation. `check` runs the entire sequence through the initial snapshot but never listens on a network socket.

## Metric collection and scrapes

The exporter evaluates enabled definitions sequentially through the serialized engine. Each SQL result becomes const Prometheus metrics in memory. Row count and represented series are accumulated against global limits, and each definition's projected series cost is checked before its Prometheus metric objects are allocated. Only after every definition succeeds is the new `Snapshot` atomically published.

```text
collection ticker -> all SQL definitions -> validate/convert -> complete Snapshot
                                                              |
                                                        atomic replace
                                                              |
Prometheus scrapes ---------------------------> current Snapshot + self-metrics
```

Scrapes only load an atomic pointer and send immutable metric objects. A failed collection updates error telemetry and retains the old pointer. This provides whole-snapshot consistency, cheap concurrent scrapes, and a precise freshness signal. It also means a stale snapshot remains available by design; operators must alert on `hercules_snapshot_age_seconds` and collection failures.

The collector uses a private `prometheus.Registry`. It never unregisters or replaces collectors in the global registry, so unrelated libraries cannot interfere with Hercules and concurrent gathers cannot observe missing families.

## Source lifecycle

Sources are initialized in package dependency and declaration order. A non-materialized Source is a DuckDB view and cannot be scheduled. A materialized Source is a DuckDB table and may set `refreshIntervalSeconds`.

The source Manager owns one goroutine per scheduled Source. Each goroutine runs one refresh synchronously, so a Source cannot overlap with itself. Intervals receive stable name-derived jitter to avoid synchronized load. Failures use bounded exponential backoff, contexts enforce timeouts, and shutdown cancels in-flight work and waits for all workers.

All source refreshes and metric queries still pass through the Engine mutex. This sacrifices query parallelism in favor of deterministic DuckDB connection ownership and eliminates connection-close and DDL/query races. A future multi-connection engine must preserve snapshot consistency and source replacement atomicity before relaxing this invariant.

## Package compiler

Packages are declarative distribution units but are treated as trusted code because they can contain arbitrary SQL and request native DuckDB extensions. The compiler provides structure and policy, not a sandbox.

Compilation performs:

1. Local path or opt-in remote resolution with size, scheme, host, redirect, and digest policy.
2. Strict YAML decoding against `hercules.dev/v1alpha1` and `kind: Package`.
3. Variable default/override/environment resolution and `{{name}}` expansion.
4. Package-relative Source path resolution.
5. Package, Source, Metric Definition, and compatibility validation.
6. Dependency topological sorting with missing-dependency and cycle errors.
7. Global Source, macro, and rendered metric name collision checks.

Remote package content is not implicitly trusted merely because it is valid YAML. Operators must enable remote loading, name allowed hosts, and normally pin the exact SHA-256.

## Metric result model

`db.Table` separates result columns from rows. The exporter resolves one value column, validates declared or inferred labels, converts the value with finite-number checks, converts labels explicitly, merges static Instance labels, sorts label names, and creates Prometheus const metrics.

Gauges require unique label sets. Counters aggregate non-negative rows. Histograms and summaries treat rows as raw observations grouped by label set. This makes their output exact for the current SQL result, but the SQL author remains responsible for cumulative counter/histogram semantics across snapshots.

The entire Metric Snapshot has one collection timestamp. Per-definition timestamps are intentionally absent: exposing a family from one cycle beside a family from a different cycle would weaken the whole-snapshot invariant.

## Ownership and shutdown

`Runtime` owns every long-lived resource created after compilation:

- worker context and wait group;
- source Manager;
- metric exporter loop;
- private Prometheus registry;
- HTTP server;
- DuckDB Engine;
- optional ephemeral database file.

Shutdown first marks readiness false, cancels worker contexts, and begins graceful HTTP shutdown. It then joins workers before closing DuckDB and removing an explicitly configured ephemeral file. `Close` is idempotent, concurrent calls observe the same shutdown, and a caller's deadline bounds how long that caller waits. If a worker does not stop before the deadline, owned cleanup continues and a later `Close` call can wait for its completion.

## Failure model

| Failure | Behavior |
| --- | --- |
| Invalid config/package/dependency/catalog | Process does not open DuckDB |
| Extension, macro, or initial Source failure | Startup fails and engine closes |
| Initial metric collection failure | Startup fails; readiness is never exposed |
| Later metric collection failure | Previous complete snapshot remains; error counter increments |
| Source refresh failure | Existing Source remains available where DuckDB statement atomicity permits; worker backs off |
| HTTP listen failure | `Run` returns non-zero through the CLI |
| Shutdown deadline | Error is joined and process exits non-zero |

## Security boundaries

Hercules controls accidental exposure and resource exhaustion through safe network defaults, policy-gated packages/extensions, resource budgets, strict schemas, timeouts, a non-root container, and vulnerability scanning. It does not interpret SQL to prove safety. Package SQL can access whatever DuckDB and the process identity can access.

See [SECURITY.md](../SECURITY.md) for the threat model and deployment requirements.

## Extension seams

The deliberate seams are:

- `db.Executor` for an alternative engine or test double;
- package resolution policy for future signed catalogs or content-addressed caches;
- Source `Type` for additional safe reader adapters;
- metric `Type` for new result encodings;
- Runtime registry composition for more self-collectors.

New abstractions should be introduced only when they deepen one of these modules. A generic plugin interface spanning SQL, scheduling, and Prometheus would erase the existing ownership seams.

## Known trade-offs

- One connection is simple and race-free but serializes long source refreshes and metric collection.
- Whole-snapshot publication maximizes consistency but one bad definition prevents all definitions from advancing.
- Exact summary quantiles require retaining all observations returned for that definition within the configured row budget.
- Raw-observation histograms are easy to author but inefficient for very large cumulative datasets.
- Process restart is the current configuration reload mechanism; zero-downtime reload belongs in an external supervisor or orchestrator.
- Packages are powerful and portable, but their SQL and extensions make them unsuitable for untrusted multi-tenant submission.
