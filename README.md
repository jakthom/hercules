# Hercules

[![License](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![test](https://github.com/jakthom/hercules/actions/workflows/test.yml/badge.svg)](https://github.com/jakthom/hercules/actions/workflows/test.yml)
[![lint](https://github.com/jakthom/hercules/actions/workflows/lint.yml/badge.svg)](https://github.com/jakthom/hercules/actions/workflows/lint.yml)



<img src="assets/heracles.png" width="250" align="right"/>


### Hercules is a Prometheus-compatible exporter that supercharges your metrics.


* **Generate prometheus-compatible metrics** from parquet, csv files, json logs, data lakes, databases, http endpoints, and much more.

* **Generate enriched, labeled** metrics properly from the source; don't relabel using your favorite metrics database.

* **Embrace** the pantheon of metrics harvesting using Prometheus-compatible scrape targets that easily tame [TPC-H benchmarks](https://www.tpc.org/information/benchmarks5.asp).


# Getting Started


### Prerequisites

You'll need `go >=1.22` on your machine.

### Clone and Run Hercules Locally

```
git clone git@github.com:jakthom/hercules.git && cd hercules

make run
```

### Get Prometheus Metrics

[localhost:9100/metrics](http://localhost:9100/metrics)


# Features

### Sources

Hercules materializes metrics from data sources such as:
- **Local files** (parquet, json, csv, xlsx, etc)
- **Object storage** (GCS, S3, Azure Blob)
- **HTTP endpoints**
- **Databases** (PostgreSQL, MySQL, SQLite)
- **Data lakes** (Iceberg, Delta)
- **Data warehouses** (BigQuery)
- **Arrow IPC buffers**


Sources can be represented as `views` or `tables` depending on desired performance and specified latency requirements.

**Example source definition:**

```
sources:
  - name: nyc_yellow_taxi_june_2024
    type: parquet
    source: https://d37ci6vzurychx.cloudfront.net/trip-data/yellow_tripdata_2024-07.parquet
    materialize: true
    refreshIntervalSeconds: 100
```

### Metrics

Metric definitions are `yml` and use `sql` in a number of supported dialects to aggregate, enrich, and materialize metric values.

Metric materialization expects two fields in the query resultset: a `struct` field of `tags` and a `value` column corresponding to the metric value.

#### Prometheus Metric Types

Hercules supports the following metric types:

- Gauge metrics ✅
- Counter metrics ✅
- Summary metrics ✅
- Histogram metrics ✅


**Example Gauge Metric Definition:**

```
metrics:
  gauge:
    - name: nyc_pickup_location_fare_total // Prometheus metric name
      help: Total NYC fares for the month of August by pickup location // Prometheus help tag
      sql: select struct_pack(pickupLocation := PULocationID::text), sum(fare_amount) as val from nyc_yellow_taxi_june_2024 group by 1 // SQL-based metric definition
      labels: // Struct key(s) to be parsed and injected into metric labels
        - pickupLocation
```


**Example Histogram Metric Definition:**
```
metrics:
  histogram:
    - name: query_duration_seconds
      help: Histogram of Snowflake virtual warehouse query duration seconds
      sql: from snowflake_query_history select struct_pack(user :=  user_name, warehouse := warehouse_name) as labels, total_elapsed_time as value;
      labels:
        - user
        - warehouse
      buckets:
        - 0.1
        - 0.5
        - 1
        - 2
        - 4
        - 8
        - 16
```


**Example Summary Metric Definition:**
```
  summary:
    - name: virtual_warehouse_query_duration_seconds
      help: Summary of Snowflake virtual warehouse query duration seconds
      sql: from snowflake_query_history select struct_pack(user :=  user_name, warehouse := warehouse_name) as labels, total_elapsed_time as value;
      labels:
        - user
        - warehouse
      objectives:
        - 0.001
        - 0.05
        - 0.01
        - 0.5
        - 0.9
        - 0.99
```


**Example Counter Metric Definition:**
```
  counter:
    - name: queries_executed_count
      help: The count of Snowflake queries executed by user and warehouse
      sql: from snowflake_query_history select struct_pack(user :=  user_name, warehouse := warehouse_name) as labels, 1 as value;
      labels:
        - user
        - warehouse
```


### Metric Enrichment

Hercules **sources** and **metrics** can be *externally enriched*, leading to more ***thorough***, ***accurate*** (or is it precise?), ***properly-labeled*** metrics.

Integrate, calculate, enrich, and label on the edge.

**Example Enriched Source:**

```
sources:
  - name: user_signups
    type: sql
    source: select s.timestamp, s.userId, u.name from signups s join users u on s.userId = u.id
    materialize: true
    refreshIntervalSeconds: 5
```


### Macros

Metric definitions can be kept DRY using Hercules macros.

Macros are automatically ensured on startup and are useful for common activities such as:

- Parsing log lines
- Reading useragent strings
- Schematizing unstructured data
- Enriching results with third-party tooling
- Tokenizing attributes


**Example Macro Definition:**

```
macros:
  - sql: create or replace macro parse_tomcat_log(logLine) AS ( $PARSING_LOGIC );
```


### Global Labels

Hercules allows global labels to be propagated to all configured metrics. So you don't have to guess where a metric came from.

Labels can also be propagated directly from environment variables.

**Example label definition:**
```
globalLabels:
  - cell: ausw1
  - env: dev
  - region: $REGION # Propagate the value of an environment variable titled `REGION` to prometheus labels
```

### Packages

Hercules includes a yml-based package loader which means extensions, macros, sources, and metrics can be logically grouped and distributed.

Starter packages can be found in the [hercules-packages](/hercules-packages/) directory.

**Example package registration:**

```
packages:
  - package: hercules-packages/snowflake/1.0.yml
    variables:
      yo: yee
    metricPrefix: skt_
```


### Embedded Analytics

A byproduct of Hercules being ridiculously efficient is the capability to **materialize and serve a lot more metrics, from a lot more sources, using a single Prometheus scrape endpoint.**


### Other Hercules Niceties

- Calculate prometheus-compatible metrics from geospatial data
- Coerce unwieldy files to useful statistics using full-text search
- Enhance metric labeling using vector similarity search
- Use modern [pipe sql syntax](https://research.google/pubs/sql-has-problems-we-can-fix-them-pipe-syntax-in-sql/) or [prql](https://prql-lang.org/) for defining and transforming your metrics
- [Don't start queries with `select`](https://jvns.ca/blog/2019/10/03/sql-queries-don-t-start-with-select/) if you don't want to (thanks Jevans!)


# Further Resources

Coming soon.
