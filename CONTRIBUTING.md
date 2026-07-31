# Contributing

Hercules is experimental, but changes should preserve the invariants in [`CONTEXT.md`](CONTEXT.md) and [`docs/architecture.md`](docs/architecture.md). Read the architecture decisions before changing package trust, startup ordering, snapshot publication, or connection ownership.

## Development setup

Install Go 1.26.5, a C compiler, `make`, and golangci-lint 2.12.2. DuckDB's Go bindings use CGO.

```sh
make mod
make quality
make build
make check
```

`make quality` runs formatting verification, strict lint, vet, race-enabled tests, and the vulnerability scan. New behavior needs focused tests. Concurrency or lifecycle changes need a race-enabled regression test. Package/config changes must update their JSON Schema, examples, and author documentation in the same change.

## Design guidelines

- Keep CLI policy in `cmd/hercules` and resource ownership in `internal/runtime`.
- Depend on the narrow `db.Executor` seam where SQL execution is all a module needs.
- Keep DuckDB queries out of Prometheus `Collect`.
- Publish a complete Metric Snapshot atomically; never advance a subset of families.
- Make goroutine ownership, cancellation, and joining explicit.
- Fail strict configuration and Package compilation before runtime mutation.
- Treat Package SQL and extensions as trusted code and keep new capabilities policy-gated.
- Prefer deep modules with small interfaces over generic plugin abstractions.

## Pull requests

Explain the operator-visible behavior, failure modes, compatibility impact, and validation performed. Add an ADR under `docs/adr/` when changing an architectural invariant or introducing a consequential trade-off. Keep commits reviewable and do not include generated binaries, databases, coverage files, or secrets.

## Releases

See [`docs/releasing.md`](docs/releasing.md). Do not manually create release assets outside the documented tag workflow.
