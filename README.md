# Hercules

Hercules is a DuckDB-powered Prometheus exporter that makes metrics harvesting go vroom.



**Generate** prometheus-compatible metrics from parquet, csv files, json logs, data lakes, databases, http endpoints, and much more.

**Enrich and label** your metrics properly at the source, not in your metrics database.

**Supercharge** your metrics harvesting with Prometheus-compatible scrape targets that can run [TPC-H benchmarks](https://www.tpc.org/information/benchmarks5.asp).


# Getting Started



### Prerequisites

You'll need `go >=1.22` on your machine.

### Clone and Run Hercules Locally

```
git clone git@github.com:dbecorp/hercules.git && cd hercules

make run
```

### Get Prometheus Metrics

[localhost:9100/metrics](localhost:9100/metric])


# Features

### Sources

Hercules materializes metrics from data sources such as:
- Local files (parquet, json, csv, xlsx, etc)
- Object storage (GCS, S3, Azure Blob)
- Data lakes (Iceberg, Delta)
- Databases (PostgreSQL, MySQL, SQLite)
- HTTP endpoints
- Arrow IPC buffers


Hercules sources are represented as either `views` or `tables` - depending on desired performance and specified latency requirements.

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

Each Hercules metric is defined using **sql** in a number of supported dialects.

Metric definitions require two fields to be returned in the resultset: a `struct` field of `tags` and a `value` column corresponding to the metric value.

#### Metric Types

Hercules supports the following Prometheus metric types:

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

### Enrichment

Hercules **sources** and **metrics** can both be joined with *external enrichments*, leading to more ***thorough***, ***accurate*** (or is it precise?) metrics.

By enriching your metrics on the edge you no longer have to run *expensive, time-consuming operations on your centralized Prometheus data store.*