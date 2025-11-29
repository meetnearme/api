# WARNING: This was vibe-coded but seems to be mostly accurate

# Grafana Queries for Prometheus - Quick Guide

## Prometheus Data Structure

Prometheus stores data as **time series**. Each time series has:

1. **Metric Name**: What you're measuring (e.g., `http_requests_total`,
   `go_memstats_alloc_bytes`)
2. **Labels**: Key-value pairs that provide dimensions (e.g., `method="GET"`,
   `path="/api/events"`, `status="200"`)
3. **Values**: Float64 numbers with timestamps

### Metric Types

- **Counter**: Only increases (e.g., total requests, total errors)
- **Gauge**: Can go up or down (e.g., memory usage, active connections)
- **Histogram**: Distribution of values in buckets (e.g., request duration)
- **Summary**: Similar to histogram but with quantiles

## Default Metrics from `promhttp.Handler()`

When you use `promhttp.Handler()`, it automatically exposes Go's built-in
metrics:

### Go Runtime Metrics (prefix: `go_`)

- `go_memstats_alloc_bytes` - Currently allocated memory
- `go_memstats_sys_bytes` - Total system memory obtained
- `go_memstats_gc_duration_seconds` - GC pause duration
- `go_goroutines` - Number of goroutines
- `go_info` - Go version info
- `go_threads` - Number of OS threads

### HTTP Request Metrics (prefix: `promhttp_`)

- `promhttp_metric_handler_requests_total` - Total requests to `/metrics`
  endpoint
- `promhttp_metric_handler_requests_in_flight` - Current requests in progress

## Common PromQL Queries for Grafana

### 1. Request Rate (Requests per Second)

```promql
rate(promhttp_metric_handler_requests_total[5m])
```

**Use case**: Shows how many requests per second to your `/metrics` endpoint

### 2. Memory Usage

```promql
go_memstats_alloc_bytes
```

**Use case**: Current memory allocation in bytes

**Convert to MB**:

```promql
go_memstats_alloc_bytes / 1024 / 1024
```

### 3. Goroutine Count

```promql
go_goroutines
```

**Use case**: Number of active goroutines (useful for detecting goroutine leaks)

### 4. GC Duration

```promql
go_memstats_gc_duration_seconds
```

**Use case**: Garbage collection pause times

**Average GC duration**:

```promql
rate(go_memstats_gc_duration_seconds_sum[5m]) / rate(go_memstats_gc_duration_seconds_count[5m])
```

### 5. Total System Memory

```promql
go_memstats_sys_bytes / 1024 / 1024
```

**Use case**: Total memory obtained from OS (in MB)

## Creating Custom HTTP Metrics (Optional)

To track your actual application requests (not just the `/metrics` endpoint),
you can add custom metrics:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "HTTP request duration in seconds",
        },
        []string{"method", "path"},
    )
)
```

Then in your middleware:

```go
func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        next.ServeHTTP(w, r)

        duration := time.Since(start).Seconds()
        httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
        httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(statusCode)).Inc()
    })
}
```

### Query Custom Metrics

**Request rate by endpoint**:

```promql
sum(rate(http_requests_total[5m])) by (path)
```

**95th percentile latency**:

```promql
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path))
```

**Error rate**:

```promql
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))
```

## Example Grafana Dashboard Panels

### Panel 1: Memory Usage (Gauge)

- **Query**: `go_memstats_alloc_bytes / 1024 / 1024`
- **Visualization**: Stat or Gauge
- **Unit**: MB

### Panel 2: Goroutines (Graph)

- **Query**: `go_goroutines`
- **Visualization**: Graph
- **Unit**: None

### Panel 3: GC Duration (Graph)

- **Query**:
  `rate(go_memstats_gc_duration_seconds_sum[5m]) / rate(go_memstats_gc_duration_seconds_count[5m])`
- **Visualization**: Graph
- **Unit**: seconds

### Panel 4: Memory Breakdown (Graph)

- **Queries**:
  - Allocated: `go_memstats_alloc_bytes / 1024 / 1024`
  - System: `go_memstats_sys_bytes / 1024 / 1024`
  - Heap: `go_memstats_heap_alloc_bytes / 1024 / 1024`
- **Visualization**: Graph
- **Unit**: MB
- **Legend**: `{{__name__}}`

## Quick Start: Creating Your First Panel

1. **In Grafana**: Click "+" → "Add visualization"
2. **Select Prometheus** as data source
3. **Enter query**: `go_memstats_alloc_bytes / 1024 / 1024`
4. **Choose visualization**: "Stat" or "Graph"
5. **Set unit**: "Mbytes" (or "bytes" if you prefer raw values)
6. **Panel title**: "Memory Usage (MB)"
7. **Save**

## Useful PromQL Functions

- `rate()` - Calculate per-second average rate (for counters)
- `increase()` - Calculate increase over time (for counters)
- `sum()` - Sum values
- `avg()` - Average values
- `max()` / `min()` - Maximum/minimum values
- `by (label)` - Group by label
- `histogram_quantile(0.95, ...)` - Calculate percentile from histogram
- `[5m]` - Time range selector (last 5 minutes)

## Time Ranges

- `[1m]` - Last 1 minute
- `[5m]` - Last 5 minutes
- `[15m]` - Last 15 minutes
- `[1h]` - Last 1 hour

## Next Steps

1. **Explore available metrics**: Visit `http://localhost:8000/metrics` (when
   `IS_LOCAL_ACT=true`)
2. **Use Prometheus UI**: Visit `http://localhost:9090` to test queries
3. **Create panels**: Start with the examples above
4. **Add custom metrics**: Instrument your application for business metrics

## Resources

- [PromQL Documentation](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Prometheus Metric Types](https://prometheus.io/docs/concepts/metric_types/)
- [Grafana Prometheus Data Source](https://grafana.com/docs/grafana/latest/datasources/prometheus/)
