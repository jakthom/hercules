# Treat packages as trusted code

Hercules packages can execute arbitrary DuckDB SQL, read process-accessible data, and request native DuckDB extensions. Packages are therefore trusted operator-supplied code rather than a multi-tenant sandbox; remote packages and community extensions are disabled by default and require explicit policy opt-in.
