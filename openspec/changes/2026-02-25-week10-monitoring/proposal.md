## Why

Goclaw currently lacks production-grade observability features. Without metrics and monitoring, operators cannot:
- Track system performance and resource usage
- Identify bottlenecks in workflow execution
- Set up alerts for anomalies or failures
- Make data-driven decisions for capacity planning

Adding Prometheus metrics will enable production monitoring, alerting, and performance analysis.

## What Changes

- Add Prometheus metrics instrumentation to core components
- Expose metrics endpoint for Prometheus scraping
- Implement key performance indicators (KPIs) for workflows, tasks, and lanes
- Add metrics middleware for HTTP API
- Provide Grafana dashboard templates for visualization

## Capabilities

### New Capabilities
- `prometheus-metrics`: Core metrics collection and exposition
- `workflow-metrics`: Workflow-level metrics (submissions, completions, durations)
- `task-metrics`: Task-level metrics (executions, failures, retries)
- `lane-metrics`: Lane queue metrics (depth, wait time, throughput)
- `http-metrics`: HTTP API metrics (request rate, latency, errors)

### Modified Capabilities
- `http-server-core`: Add metrics middleware to request pipeline
- `engine-core`: Add metrics collection hooks in workflow execution

## Impact

**Affected Code:**
- `pkg/metrics/` - New package for metrics collection
- `pkg/api/middleware/metrics.go` - HTTP metrics middleware
- `pkg/engine/engine.go` - Add metrics hooks
- `pkg/lane/lane.go` - Add queue metrics
- `cmd/goclaw/main.go` - Initialize metrics server

**New Dependencies:**
- `github.com/prometheus/client_golang` - Prometheus client library

**Configuration:**
- New `metrics` section in config.yaml
- Metrics server port (default: 9091)
- Metrics path (default: /metrics)
- Enable/disable metrics collection

**Deployment:**
- Metrics endpoint exposed on separate port
- Prometheus scrape configuration required
- Optional Grafana dashboard deployment

**Performance:**
- Minimal overhead (<1% CPU, <10MB memory)
- Metrics collection is non-blocking
- Configurable metric cardinality limits

**Breaking Changes:**
- None - metrics are additive and optional
