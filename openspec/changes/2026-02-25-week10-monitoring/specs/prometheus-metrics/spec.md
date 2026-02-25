# Prometheus Metrics Specification

## Overview

This specification defines the Prometheus metrics instrumentation for Goclaw, enabling production-grade observability and monitoring.

## Metrics Endpoint

- **Path**: `/metrics`
- **Port**: `9091` (configurable)
- **Format**: Prometheus text exposition format
- **Authentication**: Optional (configurable)

## Metric Categories

### 1. Workflow Metrics

#### workflow_submissions_total
- **Type**: Counter
- **Description**: Total number of workflow submissions
- **Labels**:
  - `status`: Workflow status (pending, running, completed, failed, cancelled)
- **Example**:
  ```
  workflow_submissions_total{status="completed"} 1234
  workflow_submissions_total{status="failed"} 56
  ```

#### workflow_duration_seconds
- **Type**: Histogram
- **Description**: Workflow execution duration in seconds
- **Labels**:
  - `status`: Final status (completed, failed)
- **Buckets**: [0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300]
- **Example**:
  ```
  workflow_duration_seconds_bucket{status="completed",le="1"} 100
  workflow_duration_seconds_bucket{status="completed",le="5"} 450
  workflow_duration_seconds_sum{status="completed"} 12345.67
  workflow_duration_seconds_count{status="completed"} 1000
  ```

#### workflow_active_count
- **Type**: Gauge
- **Description**: Current number of active workflows
- **Labels**:
  - `status`: Current status (pending, running)
- **Example**:
  ```
  workflow_active_count{status="pending"} 5
  workflow_active_count{status="running"} 12
  ```

### 2. Task Metrics

#### task_executions_total
- **Type**: Counter
- **Description**: Total number of task executions
- **Labels**:
  - `status`: Execution status (completed, failed)
  - `task_type`: Type of task (optional)
- **Example**:
  ```
  task_executions_total{status="completed",task_type="http"} 5678
  task_executions_total{status="failed",task_type="script"} 123
  ```

#### task_duration_seconds
- **Type**: Histogram
- **Description**: Task execution duration in seconds
- **Labels**:
  - `task_type`: Type of task (optional)
- **Buckets**: [0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30]
- **Example**:
  ```
  task_duration_seconds_bucket{task_type="http",le="0.1"} 1000
  task_duration_seconds_bucket{task_type="http",le="1"} 4500
  task_duration_seconds_sum{task_type="http"} 2345.67
  task_duration_seconds_count{task_type="http"} 5000
  ```

#### task_retries_total
- **Type**: Counter
- **Description**: Total number of task retries
- **Labels**:
  - `task_type`: Type of task (optional)
- **Example**:
  ```
  task_retries_total{task_type="http"} 234
  ```

### 3. Lane Queue Metrics

#### lane_queue_depth
- **Type**: Gauge
- **Description**: Current depth of lane queue
- **Labels**:
  - `lane_name`: Name of the lane
- **Example**:
  ```
  lane_queue_depth{lane_name="default"} 42
  lane_queue_depth{lane_name="high_priority"} 5
  ```

#### lane_wait_duration_seconds
- **Type**: Histogram
- **Description**: Time tasks spend waiting in queue
- **Labels**:
  - `lane_name`: Name of the lane
- **Buckets**: [0.001, 0.01, 0.1, 0.5, 1, 5, 10, 30]
- **Example**:
  ```
  lane_wait_duration_seconds_bucket{lane_name="default",le="0.1"} 500
  lane_wait_duration_seconds_bucket{lane_name="default",le="1"} 900
  lane_wait_duration_seconds_sum{lane_name="default"} 456.78
  lane_wait_duration_seconds_count{lane_name="default"} 1000
  ```

#### lane_throughput_total
- **Type**: Counter
- **Description**: Total number of tasks processed by lane
- **Labels**:
  - `lane_name`: Name of the lane
- **Example**:
  ```
  lane_throughput_total{lane_name="default"} 10000
  ```

### 4. HTTP API Metrics

#### http_requests_total
- **Type**: Counter
- **Description**: Total number of HTTP requests
- **Labels**:
  - `method`: HTTP method (GET, POST, etc.)
  - `path`: Request path (normalized)
  - `status`: HTTP status code
- **Example**:
  ```
  http_requests_total{method="POST",path="/api/v1/workflows",status="201"} 1234
  http_requests_total{method="GET",path="/api/v1/workflows",status="200"} 5678
  http_requests_total{method="GET",path="/api/v1/workflows/:id",status="404"} 12
  ```

#### http_request_duration_seconds
- **Type**: Histogram
- **Description**: HTTP request duration in seconds
- **Labels**:
  - `method`: HTTP method
  - `path`: Request path (normalized)
- **Buckets**: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5]
- **Example**:
  ```
  http_request_duration_seconds_bucket{method="POST",path="/api/v1/workflows",le="0.01"} 800
  http_request_duration_seconds_bucket{method="POST",path="/api/v1/workflows",le="0.1"} 1200
  http_request_duration_seconds_sum{method="POST",path="/api/v1/workflows"} 45.67
  http_request_duration_seconds_count{method="POST",path="/api/v1/workflows"} 1234
  ```

#### http_active_connections
- **Type**: Gauge
- **Description**: Current number of active HTTP connections
- **Example**:
  ```
  http_active_connections 23
  ```

### 5. System Metrics

#### go_goroutines
- **Type**: Gauge
- **Description**: Number of goroutines currently running
- **Source**: Prometheus Go collector (built-in)

#### go_memstats_alloc_bytes
- **Type**: Gauge
- **Description**: Number of bytes allocated and in use
- **Source**: Prometheus Go collector (built-in)

#### go_gc_duration_seconds
- **Type**: Summary
- **Description**: GC invocation durations
- **Source**: Prometheus Go collector (built-in)

#### process_cpu_seconds_total
- **Type**: Counter
- **Description**: Total user and system CPU time spent
- **Source**: Prometheus process collector (built-in)

#### process_open_fds
- **Type**: Gauge
- **Description**: Number of open file descriptors
- **Source**: Prometheus process collector (built-in)

## Metric Naming Conventions

1. **Prefix**: All custom metrics use `goclaw_` prefix (optional, configurable)
2. **Units**: Include unit suffix (`_seconds`, `_bytes`, `_total`)
3. **Base units**: Use base units (seconds, not milliseconds)
4. **Naming**: Use snake_case for metric and label names

## Label Guidelines

1. **Cardinality**: Keep label cardinality low (<100 unique values per label)
2. **Avoid**: Do not use high-cardinality labels (user IDs, timestamps, UUIDs)
3. **Normalization**: Normalize paths (e.g., `/api/v1/workflows/:id` not `/api/v1/workflows/abc123`)
4. **Consistency**: Use consistent label names across metrics

## Configuration

```yaml
metrics:
  enabled: true
  port: 9091
  path: /metrics

  # Metric-specific configuration
  workflow:
    enabled: true
    duration_buckets: [0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300]

  task:
    enabled: true
    duration_buckets: [0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30]

  lane:
    enabled: true
    wait_duration_buckets: [0.001, 0.01, 0.1, 0.5, 1, 5, 10, 30]

  http:
    enabled: true
    duration_buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5]
    path_normalization: true
```

## Performance Considerations

1. **Overhead**: Metrics collection adds <1% CPU overhead
2. **Memory**: Metrics storage uses <10MB memory
3. **Non-blocking**: All metric operations are non-blocking
4. **Atomic**: Use atomic operations for counters and gauges

## Security Considerations

1. **Port isolation**: Metrics endpoint on separate port from API
2. **Access control**: Optional authentication for metrics endpoint
3. **Sensitive data**: Never include sensitive data in labels
4. **Rate limiting**: Optional rate limiting for metrics endpoint

## Example Queries

### Workflow Success Rate
```promql
sum(rate(workflow_submissions_total{status="completed"}[5m]))
/
sum(rate(workflow_submissions_total[5m]))
```

### P95 Workflow Duration
```promql
histogram_quantile(0.95,
  rate(workflow_duration_seconds_bucket[5m])
)
```

### Lane Queue Saturation
```promql
lane_queue_depth / lane_capacity
```

### HTTP Error Rate
```promql
sum(rate(http_requests_total{status=~"5.."}[5m]))
/
sum(rate(http_requests_total[5m]))
```

### API P99 Latency by Endpoint
```promql
histogram_quantile(0.99,
  sum by (path, le) (
    rate(http_request_duration_seconds_bucket[5m])
  )
)
```

## Compatibility

- **Prometheus**: Compatible with Prometheus 2.x+
- **Grafana**: Compatible with Grafana 8.x+
- **OpenMetrics**: Follows OpenMetrics specification
- **Client Library**: Uses `prometheus/client_golang` v1.x
