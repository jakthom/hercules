<h1 align="center">
  <img src="./assets/hercules.png" alt="Hercules" width="25%" style="border-radius: 25%">
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



# Getting Started

Launching Hercules in a Codespace is the easiest way to get started.

[![Launch GitHub Codespace](https://github.com/codespaces/badge.svg)](https://github.com/codespaces/new?hide_repo_select=true&ref=main&repo=873715049)




# Sources

Hercules materializes Prometheus metrics by querying:

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

Sources and  can be *externally enriched*, leading to more ***thorough***, ***accurate*** (or is it precise?), ***properly-labeled*** metrics.

Integrate, calculate, enrich, and label on the edge.



# Macros

Metric definitions can be kept DRY using SQL macros.

Macros are useful for operations like:

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

Starter packages can be found in the [hercules-packages](/hercules-packages/) directory.



# Bonus

- Calculate prometheus-compatible metrics from geospatial data
- Coerce unwieldy files to useful statistics using full-text search
- Use modern [pipe sql syntax](https://research.google/pubs/sql-has-problems-we-can-fix-them-pipe-syntax-in-sql/) or [prql](https://prql-lang.org/) for defining and transforming your metrics
- You don't need to [start queries with `select`](https://jvns.ca/blog/2019/10/03/sql-queries-don-t-start-with-select/).


# Further Resources

More to come.
