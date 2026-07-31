# Hercules

Hercules turns operator-authored SQL over heterogeneous data into Prometheus metric snapshots using an embedded DuckDB engine.

## Language

**Instance**:
A running Hercules process configured by exactly one top-level configuration.
_Avoid_: Server, daemon

**Package**:
A versioned, trusted YAML bundle of extensions, macros, sources, and metric definitions.
_Avoid_: Plugin, bundle

**Source**:
A named DuckDB table or view that makes external or derived data available to metric queries.
_Avoid_: Input, dataset

**Metric Definition**:
A named SQL query plus Prometheus type, labels, help text, and type-specific settings.
_Avoid_: Metric, query

**Metric Snapshot**:
An immutable, last-known-good set of Prometheus metrics produced by evaluating all enabled metric definitions.
_Avoid_: Registry, cache

**Refresh**:
A scheduled replacement of a materialized source from its configured source query.
_Avoid_: Sync, reload

## Relationships

- An **Instance** loads zero or more **Packages** and one implicit core **Package**.
- A **Package** owns zero or more **Sources** and **Metric Definitions**.
- A **Metric Definition** may query one or more **Sources**.
- An **Instance** exposes exactly one current **Metric Snapshot**.
- A materialized **Source** may receive recurring **Refreshes**.

## Example dialogue

> **Dev:** “Does a Prometheus scrape execute every **Metric Definition**?”
> **Domain expert:** “No. Collection evaluates definitions into a **Metric Snapshot**; scrapes only expose the latest successful snapshot.”

## Flagged ambiguities

- “plugin” previously meant both a declarative **Package** and a native DuckDB extension. Hercules now reserves **Package** for its YAML format and “DuckDB extension” for native executable extensions.
- “materialize” previously meant both refreshing a **Source** and evaluating a **Metric Definition**. Hercules uses **Refresh** for sources and “collection” for producing a **Metric Snapshot**.
