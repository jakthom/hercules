
# To-Do, Features

## Code stuff
    - Interface/genericize metrics ❌

## Database stuff
    - Support ATTACH-ing s3/gcs-based databases ❌
    - Support duckdb-based secrets registration ❌
    - Namespace all packages using a database so they don't collide ❌

## Sources
    - Materialize as table/view ❌
    - Refresh on each `select` ❌
    - Refresh on http-post (POST collector:9100/sources/$BLAH/refresh) ❌
    - Support view-based sources ✅

## Registries
    - Make a registry for each package so they can be reloaded independently @ some point ✅
    - Reload registries via http ❌

## Metrics
    - Handle scalar values well ✅
    - Continue materializing metrics if a single metric cannot be materialized instead of blowing up ✅
    - Prefix metric names with the package name ✅
    - Support custom prefixes via configuration ✅
    - Genericize/interface metrics ❌
    - Metric definition packages
        - Support named path groupings ❌
        - Support package
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

## Tests dude
    - Write them ❌

## Distribution stuff
    - Build docker and publish to ghcr.io ❌
    - Build binaries and publish to package registry ❌
    - Tests and badge ❌
    - Lint and badge ❌
    - Codespaces ❌

## Package distribution stuff
    - Turn dashes to underscores in package names so the metrics get properly registered ❌
    - Pull from public/s3-backed, nicely-named package registry ❌
    - CLI for authoring, linting, and publishing packages? Maybe? idk, not yet. ❌

## Outstanding questions
    -
