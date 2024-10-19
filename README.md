# ducktheus_exporter
A DuckDB-powered Prometheus Exporter


# To-Do, Features
## Clean up application initialization
    - Database initialization ✅
    - Source initialization ✅
    - Metric materialization ✅
    - Support preloading DuckDB extensions ❌

## DuckDB stuff
    - Support ATTACH-ing s3/gcs-based databases ❌

## Sources
    - Refresh on startup ✅
    - Refresh on time interval ✅
    - Refresh on each `select` ❌
    - Refresh on http-post (POST collector:9100/sources/$BLAH/refresh) ❌

## Metrics
    - Support all Prometheus metric types
        - gauges ✅
        - histograms ❌
        - counters ❌
        - summaries ❌
    - Support named path groupings ❌
    - Support push-based OTEL ❌

## Config/Functionality
    - Macros ✅

## Secrets
    - Support passing env-based secrets ❌

## Distribution stuff
    - Build docker and publish to ghcr.io ❌
    - Build binaries and publish to package registry ❌
    - Tests and badge
    - Lint and badge
    - Codespaces ❌
