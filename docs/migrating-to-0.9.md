# Migrating from 0.8 to 0.9

Hercules 0.9 deliberately tightens its alpha interfaces. Run `hercules check-config` before opening DuckDB, then `hercules check` in the production filesystem and network environment.

## Instance configuration

- Add the required integer `version: 1`.
- Change `port` from a quoted string to an integer.
- Set `listenAddress` explicitly when remote scraping is intended; the default is now `127.0.0.1`.
- Set `db` explicitly when persistence is intended; the default is now `:memory:`. File-backed databases are retained unless `ephemeralDB: true` is explicit.
- Prefer the canonical `globalLabels` object. The old list-of-objects form is decoded for migration, but new configurations should use `globalLabels: {env: production}`.
- Add DuckDB, collection, row, and series limits appropriate to the deployment. Defaults are documented in [Configuration](configuration.md).
- Allowlist every requested DuckDB extension under `security.allowedExtensions`. Remote Packages and community extensions now require explicit security policy.

Unknown keys, missing environment variables, malformed values, and label collisions now fail instead of being ignored or replaced with partial defaults.

## Packages

Every Package document now requires:

```yaml
apiVersion: hercules.dev/v1alpha1
kind: Package
name: example
version: "1.0"
```

Package variables and reference overrides must be string-to-string maps. Relative non-SQL Source paths resolve from the Package directory. Add `dependencies` when one Package needs another Package's macros or Sources; initialization is no longer concurrent or accidental.

Remote Package references are disabled by default. If enabled, configure an exact hostname allowlist and normally add `sha256`; use `hercules package-digest PACKAGE.yml` to produce the value.

## Metric Definitions

- Omit `enabled` for an enabled Metric Definition; set `enabled: false` only to disable it. This corrects the old zero-value behavior.
- Counter names must end in `_total` and SQL rows for the same label set are summed.
- Histogram and summary SQL rows are raw observations for the current Metric Snapshot. Histograms require finite, strictly increasing `buckets`; summaries require `quantile`/`error` objective objects. Numeric objective shorthand remains accepted during migration.
- Prefer a numeric result column named `value` and declare `labels` explicitly. Hercules now rejects ambiguous columns, undeclared columns, duplicate gauge label sets, invalid numeric values, and static/dynamic label collisions.
- Package and optional reference prefixes are part of the emitted name. Cross-Package collisions, reserved Hercules/Go/process prefixes, and type-generated suffix collisions fail compilation.

Prometheus scrapes now serve the last complete Metric Snapshot; they no longer run SQL. Alert on snapshot age and collection errors instead of treating scrape success alone as freshness.

## Deployment and CLI

The supported entrypoint is `hercules run --config PATH` (or the equivalent default command). Replace ad hoc startup validation with:

```sh
hercules check-config --config PATH
hercules check --config PATH
hercules run --config PATH
```

The release container is non-root and multi-architecture. Ensure mounted database, cache, and temporary directories have suitable ownership; mount configuration, Package documents, and local Source files read-only where possible.
