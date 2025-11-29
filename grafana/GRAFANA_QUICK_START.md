# WARNING: This was vibe-coded but seems to be mostly accurate

# Grafana Quick Start - Querying Your Go App Metrics

## Current Setup Status

✅ **Prometheus** is scraping `go-app:8000/metrics` successfully ✅ **Grafana**
is available at `http://localhost:3001` ✅ **Metrics endpoint** is working and
exposing Go runtime metrics

## Available Metrics in Your App

Based on your current setup, you have these metrics available:

### Go Runtime Metrics (Gauge - Current Values)

- `go_goroutines` - Number of active goroutines
- `go_memstats_alloc_bytes` - Currently allocated memory (bytes)
- `go_memstats_heap_alloc_bytes` - Heap memory allocated (bytes)
- `go_memstats_sys_bytes` - Total system memory obtained (bytes)
- `go_memstats_heap_idle_bytes` - Idle heap memory (bytes)
- `go_memstats_heap_inuse_bytes` - In-use heap memory (bytes)
- `go_memstats_heap_objects` - Number of heap objects

### Go GC Metrics (Summary - Distribution)

- `go_gc_duration_seconds` - Garbage collection pause duration
- `go_gc_duration_seconds_sum` - Total GC duration
- `go_gc_duration_seconds_count` - Number of GC cycles

### Metric Handler Metrics (Counter)

- `promhttp_metric_handler_requests_total` - Total requests to `/metrics`
  endpoint

## Step-by-Step: Creating Your First Panel

### 1. Access Grafana

- Open `http://localhost:3001`
- Login: `admin` / `admin`

### 2. Create a New Dashboard

1. Click **"+"** → **"Create dashboard"**
2. Click **"Add visualization"**

### 3. Select Prometheus Data Source

- In the query editor, make sure **Prometheus** is selected as the data source
- If you don't see Prometheus, go to **Configuration → Data Sources** and verify
  it's configured

### 4. Enter Your First Query

Copy and paste one of these queries:

#### Query 1: Goroutine Count (Simple Gauge)

```promql
go_goroutines
```

- **Visualization**: Choose **"Stat"** or **"Gauge"**
- **Panel Title**: "Active Goroutines"
- **Unit**: None (or "short")

#### Query 2: Memory Usage in MB (Converted)

```promql
go_memstats_alloc_bytes / 1024 / 1024
```

- **Visualization**: **"Graph"** or **"Stat"**
- **Panel Title**: "Memory Usage (MB)"
- **Unit**: "Mbytes"

#### Query 3: Memory Breakdown (Multiple Metrics)

```promql
go_memstats_heap_alloc_bytes / 1024 / 1024
```

- **Panel Title**: "Heap Memory (MB)"
- **Unit**: "Mbytes"

Add multiple queries to the same panel:

- Query A: `go_memstats_heap_alloc_bytes / 1024 / 1024` → Legend: "Heap
  Allocated"
- Query B: `go_memstats_heap_inuse_bytes / 1024 / 1024` → Legend: "Heap In Use"
- Query C: `go_memstats_heap_idle_bytes / 1024 / 1024` → Legend: "Heap Idle"

#### Query 4: GC Duration (Average)

```promql
rate(go_gc_duration_seconds_sum[5m]) / rate(go_gc_duration_seconds_count[5m])
```

- **Panel Title**: "Average GC Duration"
- **Unit**: "seconds"
- **Visualization**: **"Graph"**

#### Query 5: GC Duration by Quantile

```promql
go_gc_duration_seconds{quantile="0.5"}
```

- Shows median GC duration
- Try different quantiles: `0`, `0.25`, `0.5`, `0.75`, `1` (min, 25th, median,
  75th, max)

## Example Dashboard Panels

### Panel 1: System Overview (Stat Panels)

Create 3 separate stat panels side-by-side:

**Panel 1A: Goroutines**

- Query: `go_goroutines`
- Visualization: Stat
- Color: Blue

**Panel 1B: Memory (MB)**

- Query: `go_memstats_alloc_bytes / 1024 / 1024`
- Visualization: Stat
- Unit: Mbytes
- Color: Green

**Panel 1C: Heap Objects**

- Query: `go_memstats_heap_objects`
- Visualization: Stat
- Color: Orange

### Panel 2: Memory Over Time (Graph)

- **Query**: `go_memstats_alloc_bytes / 1024 / 1024`
- **Visualization**: Graph
- **Panel Title**: "Memory Usage Over Time"
- **Unit**: Mbytes
- **Legend**: "Allocated Memory"

### Panel 3: Memory Breakdown (Time Series)

Add multiple queries to one graph:

- Query A: `go_memstats_heap_alloc_bytes / 1024 / 1024` → Legend: "Heap
  Allocated"
- Query B: `go_memstats_heap_inuse_bytes / 1024 / 1024` → Legend: "Heap In Use"
- Query C: `go_memstats_heap_idle_bytes / 1024 / 1024` → Legend: "Heap Idle"
- **Panel Title**: "Memory Breakdown"
- **Unit**: Mbytes

### Panel 4: GC Performance (Graph)

- **Query**:
  `rate(go_gc_duration_seconds_sum[5m]) / rate(go_gc_duration_seconds_count[5m])`
- **Panel Title**: "Average GC Duration"
- **Unit**: seconds

## Quick Reference: PromQL Functions

- `rate(metric[5m])` - Calculate per-second rate over 5 minutes (for counters)
- `[5m]` - Time range (1m, 5m, 15m, 1h)
- `/ 1024 / 1024` - Convert bytes to MB
- `{label="value"}` - Filter by label
- `by (label)` - Group by label

## Testing Your Queries

Before creating panels, test queries in Prometheus UI:

1. Open `http://localhost:9090`
2. Go to **"Graph"** tab
3. Enter your query
4. Click **"Execute"**
5. If it works here, it will work in Grafana!

## Troubleshooting

### Grafana shows "No data"

- Check that Prometheus data source is configured correctly
- Verify Prometheus is scraping: `http://localhost:9090/targets`
- Test query in Prometheus UI first

### Can't see Prometheus data source

- Go to **Configuration → Data Sources**
- Verify Prometheus is configured with URL: `http://prometheus:9090`
- Check if it's marked as "Default"

### Metrics not updating

- Check Prometheus targets: `http://localhost:9090/targets`
- Verify `go-app:8000` target shows "UP"
- Check scrape interval (should be 15s)

## Next Steps

1. **Create your first panel** using one of the queries above
2. **Add more panels** to create a comprehensive dashboard
3. **Set time ranges** in Grafana to see historical data
4. **Save your dashboard** for future use

## Useful Links

- Prometheus UI: `http://localhost:9090`
- Grafana UI: `http://localhost:3001`
- Metrics endpoint: `http://localhost:8000/metrics` (when IS_LOCAL_ACT=true)
