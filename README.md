# Hercules

![heracles](assets/heracles.png)

Hercules is a DuckDB-powered Prometheus exporter that supercharges metrics.


**Generate prometheus-compatible metrics** from parquet, csv files, json logs, data lakes, databases, http endpoints, and much more.

**Generate enriched, labeled** metrics properly from the source; don't relabel using your favorite metrics database.

**Embrace** the pantheon of metrics harvesting with Prometheus-compatible scrape targets that easily tame [TPC-H benchmarks](https://www.tpc.org/information/benchmarks5.asp).


# Getting Started


### Prerequisites

You'll need `go >=1.22` on your machine.

### Clone and Run Hercules Locally

```
git clone git@github.com:dbecorp/hercules.git && cd hercules

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


**Example Gauge Metric Definition**

```
metrics:
  gauge:
    - name: nyc_pickup_location_fare_total // Prometheus metric name
      help: Total NYC fares for the month of August by pickup location // Prometheus help tag
      sql: select struct_pack(pickupLocation := PULocationID::text), sum(fare_amount) as val from nyc_yellow_taxi_june_2024 group by 1 // SQL-based metric definition
      labels: // Struct key(s) to be parsed and injected into metric labels
        - pickupLocation
```


**Example Histogram Metric Definition**
```
metrics:
  histogram:
    - name: query_duration_seconds
      help: Histogram of Snowflake virtual warehouse query duration seconds
      sql: select struct_pack(user :=  user_name, warehouse := warehouse_name) as labels, total_elapsed_time as value from snowflake_query_history;
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


**Example Summary Metric Definition**
```
  summary:
    - name: virtual_warehouse_query_duration_seconds
      help: Summary of Snowflake virtual warehouse query duration seconds
      sql: select struct_pack(user :=  user_name, warehouse := warehouse_name) as labels, total_elapsed_time as value from snowflake_query_history;
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


**Example Counter Metric Definition**
```
  counter:
    - name: queries_executed_count
      help: The count of Snowflake queries executed by user and warehouse
      sql: select struct_pack(user :=  user_name, warehouse := warehouse_name) as labels, 1 as value from snowflake_query_history;
      labels:
        - user
        - warehouse
```


### Metric Enrichment

Hercules **sources** and **metrics** can be *externally enriched*, leading to more ***thorough***, ***accurate*** (or is it precise?), properly-labeled metrics.

Don't run expensive, time-consuming relabel and join operations on your centralized Prometheus data store. Integrate, calculate, enrich, and label on the edge.

```
  - name: user_signups
    type: sql
    source: select s.timestamp, s.userId, u.name from signups s join users u on s.userId = u.id
    materialize: true
    refreshIntervalSeconds: 5
```


### Macros

Keep metric definitions DRY using Hercules macros.

Macros are automatically ensured on startup and are useful for common activities such as:

- Parsing log lines
- Reading useragent strings
- Schematizing unstructured data


**Example Macro Definition**

```
macros:
  - sql: create or replace macro parse_tomcat_log(logLine) AS ( $PARSING_LOGIC );
```


### Global Labels

Hercules allows global labels to be propagated to all configured metrics. So you don't have to guess where the metric came from.

**Example label definition**
```
globalLabels:
  - cell: ausw1
  - env: dev
```


### Embedded Analytics


### Other Hercules Niceties

- Calculate prometheus-compatible metrics from geospatial data
- Coerce unweildy files to useful statistics using full-text search
- Enhance metric labeling using vector similarity search
- [Don't start queries with `select`](https://jvns.ca/blog/2019/10/03/sql-queries-don-t-start-with-select/) if you don't want to
- Use modern [pipe sql syntax](https://research.google/pubs/sql-has-problems-we-can-fix-them-pipe-syntax-in-sql/) or [prql](https://prql-lang.org/) for defining and transforming your metrics


# Further Resources
