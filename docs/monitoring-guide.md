# Monitoring Guide

Goclaw provides production-grade monitoring with Prometheus metrics, Grafana dashboards, and pre-configured alert rules.

## Quick Start

```bash
# Start the full monitoring stack
docker-compose up -d

# Access services
# Goclaw API:  http://localhost:8080
# Metrics:     http://localhost:9091/metrics
# Prometheus:  http://localhost:9092
# Grafana:     http://localhost:3000 (admin/admin)
```

## Metrics Reference

### Workflow Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `workflow_submissions_total` | Counter | `status` | Total workflow submissions (submitted, completed, failed, cancelled) |
| `workflow_duration_seconds` | Histogram | `status` | Workflow execution duration |
| `workflow_active_count` | Gauge | `status` | Current active workflows by status |

Histogram buckets: 0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300 seconds

### Task Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `task_executions_total` | Counter | `status` | Total task executions (completed, failed) |
| `task_duration_seconds` | Histogram | `status` | Task execution duration |
| `task_retries_total` | Counter | — | Total task retry attempts |

Histogram buckets: 0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30 seconds

### Lane Queue Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `lane_queue_depth` | Gauge | `lane_name` | Current queue depth per lane |
| `lane_wait_duration_seconds` | Histogram | `lane_name` | Task wait time in queue |
| `lane_throughput_total` | Counter | `lane_name` | Total tasks processed per lane |

Histogram buckets: 0.001, 0.01, 0.1, 0.5, 1, 5, 10, 30 seconds

### HTTP API Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `http_requests_total` | Counter | `method`, `path`, `status` | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | `method`, `path` | HTTP request latency |
| `http_active_connections` | Gauge | — | Current active HTTP connections |

Histogram buckets: 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5 seconds

### System Metrics (Auto-collected)

| Metric | Description |
|--------|-------------|
| `go_goroutines` | Number of goroutines |
| `go_memstats_alloc_bytes` | Bytes allocated and in use |
| `go_memstats_sys_bytes` | Bytes obtained from system |
| `go_gc_duration_seconds` | GC pause duration |
| `process_cpu_seconds_total` | Total CPU time |
| `process_resident_memory_bytes` | Resident memory size |
| `process_open_fds` | Open file descriptors |

## Configuration

```yaml
metrics:
  enabled: true
  port: 9091        # Metrics server port (separate from API)
  path: /metrics    # Metrics endpoint path
```

Environment variables:
```bash
export GOCLAW_METRICS_ENABLED=true
export GOCLAW_METRICS_PORT=9091
```

When `enabled: false`, a no-op metrics manager is used with zero overhead.

## Prometheus Configuration

The included `config/prometheus.yml` is pre-configured:

```yaml
global:
  scrape_interval: 15s
  scrape_timeout: 10s

scrape_configs:
  - job_name: 'goclaw'
    static_configs:
      - targets: ['localhost:9091']
    scrape_interval: 15s

rule_files:
  - 'alerts.yml'
```

To use with an external Prometheus, add this scrape config:

```yaml
scrape_configs:
  - job_name: 'goclaw'
    static_configs:
      - targets: ['<goclaw-host>:9091']
    metrics_path: /metrics
    scrape_interval: 15s
```

## Grafana Dashboard

Import the pre-built dashboard from `config/grafana/goclaw-dashboard.json`:

1. Open Grafana at http://localhost:3000
2. Go to Dashboards → Import
3. Upload `config/grafana/goclaw-dashboard.json`
4. Select your Prometheus data source

The dashboard includes 10 panels:
- Workflow submission rate and success rate
- Active workflows by status
- Workflow duration P95
- Lane queue depth and throughput
- HTTP request rate and latency P95
- Memory usage and goroutine count

Dashboard auto-refreshes every 10 seconds with a 1-hour default time range.

## Alert Rules

Pre-configured alerts in `config/prometheus/alerts.yml`:

| Alert | Condition | Duration | Severity |
|-------|-----------|----------|----------|
| HighWorkflowFailureRate | Failure rate > 10% | 5m | warning |
| LaneQueueBacklog | Queue depth > 100 | 10m | warning |
| HighAPILatency | P95 latency > 1s | 5m | warning |
| HighHTTPErrorRate | 5xx error rate > 5% | 5m | critical |
| GoclawServiceDown | Service unreachable | 1m | critical |
| HighMemoryUsage | Memory > 2GB | 10m | warning |
| HighGoroutineCount | Goroutines > 10,000 | 10m | warning |

### Customizing Alert Thresholds

Edit `config/prometheus/alerts.yml` and adjust the `expr` values:

```yaml
# Example: lower the failure rate threshold to 5%
- alert: HighWorkflowFailureRate
  expr: |
    (sum(rate(workflow_submissions_total{status="failed"}[5m]))
    / sum(rate(workflow_submissions_total[5m]))) > 0.05
  for: 5m
```

### Alertmanager Integration

To route alerts to Slack, PagerDuty, or email, configure Alertmanager:

```yaml
# alertmanager.yml
route:
  receiver: 'slack'
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty'

receivers:
  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/...'
        channel: '#alerts'
  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: '<key>'
```

## Troubleshooting

### Metrics endpoint not responding

1. Verify metrics are enabled: `grep -A3 "metrics:" config.yaml`
2. Check the metrics port isn't in use: `lsof -i :9091`
3. Check logs for startup errors: look for "metrics server started" log line
4. Test directly: `curl http://localhost:9091/metrics`

### Prometheus not scraping

1. Check Prometheus targets page: http://localhost:9092/targets
2. Verify the target is listed and state is "UP"
3. If "DOWN", check network connectivity between Prometheus and Goclaw
4. Verify `scrape_configs` in prometheus.yml matches your deployment

### Grafana dashboard shows "No data"

1. Verify Prometheus data source is configured correctly in Grafana
2. Check that Prometheus is scraping successfully (targets page)
3. Verify the time range in Grafana covers when Goclaw was running
4. Try querying `up{job="goclaw"}` directly in Grafana Explore

### High metrics cardinality

If Prometheus memory usage is high:
- Check for high-cardinality labels (many unique values)
- The `path` label on HTTP metrics can cause cardinality issues with dynamic URLs
- Consider reducing `scrape_interval` if storage is a concern

### Metrics collection overhead

Metrics collection is designed for < 1% CPU overhead:
- When disabled, a no-op manager is used (zero overhead)
- Prometheus counters and gauges use atomic operations
- Histogram observations are lock-free
- The metrics HTTP server runs on a separate port to avoid impacting the API
