<h1 align="center">
  <img src="./assets/hercules.png" alt="Hercules" width="20%" style="border-radius: 25%">
  <br>
  Hercules
  <br>
</h1>

<h4 align="center"> Write SQL. Get Prometheus Metrics.</h4>

<div align="center">

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jakthom/hercules)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![test](https://github.com/jakthom/hercules/actions/workflows/test.yml/badge.svg)](https://github.com/jakthom/hercules/actions/workflows/test.yml)
[![lint](https://github.com/jakthom/hercules/actions/workflows/lint.yml/badge.svg)](https://github.com/jakthom/hercules/actions/workflows/lint.yml)
</div>



# ⚡ Quickstart

Launching Hercules in a Codespace is the easiest way to get started.

A `make run` from inside your codespace will get things spinning.

[![Launch GitHub Codespace](https://github.com/codespaces/badge.svg)](https://github.com/codespaces/new?hide_repo_select=true&ref=main&repo=873715049)




# Sources

Hercules generates Prometheus metrics by querying:

- **Local files** (parquet, json, csv, xlsx, etc)
- **Object storage** (GCS, S3, Azure Blob)
- **HTTP endpoints**
- **Databases** (PostgreSQL, MySQL, SQLite)
- **Data lakes** (Iceberg, Delta)
- **Data warehouses** (BigQuery)
- **Arrow IPC buffers**

Sources can be cached and periodically refreshed, or act as views to the underlying data.

Metrics from multiple sources can be materialized using a single exporter.


# Metrics

### Definition

Metric definitions are `yml` and use `sql` in a number of supported dialects to aggregate, enrich, and materialize metric values.


Hercules supports Prometheus **gauges, counters, summaries, and histograms.**

### Enrichment

Sources and metrics can be *externally enriched*, leading to more ***thorough***, ***accurate*** (or is it precise?), ***properly-labeled*** metrics.

Integrate, calculate, enrich, and label on the edge.



# Macros

Metric definitions can be kept DRY using SQL macros.

Macros are useful for:

- Parsing log lines
- Reading useragent strings
- Schematizing unstructured data
- Enriching results with third-party tooling
- Tokenizing attributes


#  Labels

Hercules propagates global labels to all configured metrics. So you don't have to guess where a metric came from.

Labels are propagated from configuration or sourced from environment variables.


# Packages

Hercules extensions, sources, metrics, and macros can be logically grouped and distributed by the use of **packages**.

Examples can be found in the [hercules-packages](/hercules-packages/) directory.


# Getting Started Locally

This guide will help you get Hercules up and running on your local machine, including instructions for configuration, testing, and loading sample data.

## Prerequisites

- Go 1.22+ installed on your system
- Git to clone the repository
- Basic understanding of SQL and Prometheus metrics

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/jakthom/hercules.git
   cd hercules
   ```

2. Build the project:
   ```
   make build
   ```

## Running Hercules

### Basic Execution

Start Hercules with default settings:
```
make run
```

This command starts Hercules with the following defaults:
- Configuration file: `hercules.yml` in the project root
- Port: 9999 (customizable in config)
- SQLite database: `h.db` in the project root

### Debug Mode

To run Hercules with additional debug logging:
```
make debug
```

### Custom Configuration

Hercules uses a YAML configuration file. By default, it looks for `hercules.yml` in the project root. You can specify a custom configuration path using the environment variable:
```
HERCULES_CONFIG_PATH=/path/to/config.yml make run
```

## Configuration File Structure

The `hercules.yml` configuration file defines your server settings and packages:

```yaml
version: 1

name: your-instance-name
debug: false
port: 9100

globalLabels:
  - env: $ENV  # Inject prometheus labels from environment variables
  - region: us-west-1

packages:
  - package: hercules-packages/sample-package/1.0.yml
```

## Testing

Run the test suite to ensure everything is working correctly:
```
make test
```

For more detailed test coverage information:
```
make test-cover-pkg
```

## Working with Sample Data

Hercules includes several example packages in the `/hercules-packages/` directory:

1. **NYC Taxi Data**: 
   ```yaml
   packages:
     - package: hercules-packages/nyc-taxi/1.0.yml
   ```

2. **Snowflake Metrics**:
   ```yaml
   packages:
     - package: hercules-packages/snowflake/1.0.yml
   ```

3. **TPCH Benchmarks**:
   ```yaml
   packages:
     - package: hercules-packages/tpch/1.0.yml
   ```

4. **Bluesky Analytics**:
   ```yaml
   packages:
     - package: hercules-packages/bluesky/1.0.yml
   ```

To use any of these packages, add them to your `hercules.yml` file.

## Accessing Metrics

Once Hercules is running, metrics are available at:
- http://localhost:9100/metrics (if using default port 9100)

You can connect Prometheus or any compatible metrics collector to this endpoint.

## Environment Variables

Hercules supports several environment variables:
- `HERCULES_CONFIG_PATH`: Path to configuration file
- `DEBUG`: Set to enable debug logging
- `TRACE`: Set to enable trace logging
- `ENV`: Used for labeling metrics (defaults to current username)

## Creating Custom Packages

To create your own package:

1. Create a new YAML file in a directory of your choice:
   ```yaml
   name: my-package
   version: 1.0
   
   extensions:
     # SQLite extensions, if needed
   
   macros:
     # SQL macros
   
   sources:
     # Data sources
   
   metrics:
     # Metric definitions
   ```

2. Reference your package in the main configuration:
   ```yaml
   packages:
     - package: path/to/your-package.yml
   ```

# Bonus

- Calculate prometheus-compatible metrics from geospatial data
- Coerce unwieldy files to useful statistics using full-text search
- Use modern [pipe sql syntax](https://research.google/pubs/sql-has-problems-we-can-fix-them-pipe-syntax-in-sql/) or [prql](https://prql-lang.org/) for defining and transforming your metrics
- You don't need to [start queries with `select`](https://jvns.ca/blog/2019/10/03/sql-queries-don-t-start-with-select/).


# Further Resources

More to come.
