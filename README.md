# Hercules
An OLAP-powered Prometheus Exporter

# To-Do, Features
## Clean up application initialization
    - Database initialization ✅
    - Source initialization ✅
    - Metric materialization ✅

## Code stuff
    - Pass around full metric registry ✅
    - Consolidate metric definitions ✅
    - Interface/genericize metrics ❌
    - clean up code duplication ✅
    - Make DEBUG flag actually work, flip log levels accordingly ❌
    - Make middleware signature actually acceptable ✅

## DuckDB stuff
    - Support ATTACH-ing s3/gcs-based databases ❌
    - Support duckdb secrets registration ❌

## Extensions
    - Support preloading DuckDB core extensions ✅
    - Support preloading DuckDB community extensions ✅

## Sources
    - Refresh on startup ✅
    - Refresh on time interval ✅
    - Refresh on each `select` ❌
    - Refresh on http-post (POST collector:9100/sources/$BLAH/refresh) ❌

## Metrics
    - Support all Prometheus metric types
        - gauges (vec) ✅
        - histograms (vec) ✅
        - counters (vec) ✅
        - summaries (vec) ✅
    - Collector reregistration ✅
    - Support named path groupings ❌
    - Support push-based OTEL ❌
    - Labels
        - Hercules name propagated to labels ✅
        - Propagate global labels from config ✅

## Config/Functionality
    - Macros ✅

## Secrets
    - Support passing env-based secrets ❌

## Operator Niceties
    - Meta metrics
        - Source refresh timing ❌
        - Metric materialization timing ❌
        - Metric endpoint response time ❌

## Developer niceties
    - jsonschema-based config (validation, auto-complete) ❌
    - config validation (run through a local duckdb) ❌

## Distribution stuff
    - Build docker and publish to ghcr.io ❌
    - Build binaries and publish to package registry ❌
    - Tests and badge ❌
    - Lint and badge ❌
    - Codespaces ❌
