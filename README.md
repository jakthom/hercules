# ducktheus_exporter
A DuckDB-powered Prometheus Exporter


# To-Do, Features
## Clean up application initialization
    - Database initialization ✅
    - Source initialization ✅
    - Metric materialization ✅
    - Support preloading DuckDB extensions ❌

## Code stuff
    - Consolidate metric definitions and metric registry ❌
    - Interface the metrics, clean up code duplication ❌

## DuckDB stuff
    - Support ATTACH-ing s3/gcs-based databases ❌
    - Support duckdb secrets registration ❌

## Sources
    - Refresh on startup ✅
    - Refresh on time interval ✅
    - Refresh on each `select` ❌
    - Refresh on http-post (POST collector:9100/sources/$BLAH/refresh) ❌

## Metrics
    - Support all Prometheus metric types
        - gauges (vec) ✅
        - gauges (no labels) ❌
        - histograms (vec) ✅
        - histograms (no labels) ❌
        - counters (vec) 🚨 NOTE!!!!! Counters continously auto-increment. They either need to be reset (unregistered/registered) on each request, or explicitly unsupported for now. TBD. They work, but not properly.
        - counters (no labels) ❌
        - summaries (vec) ✅
        - summaries (no labels) ❌
    - Support named path groupings ❌
    - Support push-based OTEL ❌
    - Support propagating labels from config ❌

## Config/Functionality
    - Macros ✅

## Secrets
    - Support passing env-based secrets ❌

## Developer niceties
    - jsonschema-based config (validation, auto-complete) ❌
    - config validation (run through a local duckdb) ❌

## Distribution stuff
    - Build docker and publish to ghcr.io ❌
    - Build binaries and publish to package registry ❌
    - Tests and badge ❌
    - Lint and badge ❌
    - Codespaces ❌
