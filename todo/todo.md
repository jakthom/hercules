
# To-Do, Features

## Code stuff
    - Interface/genericize metrics ❌

## DuckDB stuff
    - Support ATTACH-ing s3/gcs-based databases ❌
    - Support duckdb secrets registration ❌

## Sources
    - Materialize as table/view ❌
    - Refresh on each `select` ❌
    - Refresh on http-post (POST collector:9100/sources/$BLAH/refresh) ❌
    - Support view-based sources

## Metrics
    - Handle scalar values well ❌
    - Genericize/interface metrics ❌
    - Metric definition packages
        - Support named path groupings ❌
    - Support push-based OTEL ❌

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
