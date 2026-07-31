# Expose immutable metric snapshots

Hercules evaluates enabled metric definitions outside the Prometheus scrape path and atomically replaces one last-known-good metric snapshot. This makes concurrent scrapes deterministic and cheap, at the cost of requiring explicit collection intervals, freshness metrics, and stale-snapshot policy.
