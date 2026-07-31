# Own one serialized DuckDB engine

One Runtime owns one DuckDB connection and serializes extension, macro, Source, refresh, and metric work through it. This prioritizes deterministic lifecycle and catalog consistency over parallel query throughput; a future connection pool must define transactionally consistent Source replacement and Metric Snapshot reads before changing this decision.
