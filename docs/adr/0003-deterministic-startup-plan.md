# Compile before mutating runtime state

Hercules resolves and strictly validates configuration and packages before opening or mutating DuckDB, then initializes extensions, macros, sources, and metric definitions in dependency order. Startup fails rather than serving a partially initialized exporter because silent partial metrics are more dangerous than an unavailable exporter.
