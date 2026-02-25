# gRPC API Examples

This document provides examples for using the Goclaw gRPC API with various tools and clients.

## Prerequisites

- Goclaw server running with gRPC enabled (default port: 9090)
- `grpcurl` installed for command-line testing
- Protocol buffer definitions available in `api/proto/goclaw/v1/`

## Using grpcurl

### Server Reflection

List available services:

```bash
grpcurl -plaintext localhost:9090 list
```

Describe a service:

```bash
grpcurl -plaintext localhost:9090 describe goclaw.v1.WorkflowService
```

### Workflow Operations

#### Submit Workflow

```bash
grpcurl -plaintext -d '{
  "name": "example-workflow",
  "tasks": [
    {
      "id": "task1",
      "name": "First Task",
      "dependencies": []
    },
    {
      "id": "task2",
      "name": "Second Task",
      "dependencies": ["task1"]
    }
  ]
}' localhost:9090 goclaw.v1.WorkflowService/SubmitWorkflow
```

#### List Workflows

```bash
grpcurl -plaintext -d '{
  "pagination": {
    "page_size": 10
  }
}' localhost:9090 goclaw.v1.WorkflowService/ListWorkflows
```

With status filter:

```bash
grpcurl -plaintext -d '{
  "status_filter": "WORKFLOW_STATUS_RUNNING",
  "pagination": {
    "page_size": 20
  }
}' localhost:9090 goclaw.v1.WorkflowService/ListWorkflows
```

#### Get Workflow Status

```bash
grpcurl -plaintext -d '{
  "workflow_id": "wf-123456"
}' localhost:9090 goclaw.v1.WorkflowService/GetWorkflowStatus
```

#### Cancel Workflow

```bash
grpcurl -plaintext -d '{
  "workflow_id": "wf-123456",
  "force": false
}' localhost:9090 goclaw.v1.WorkflowService/CancelWorkflow
```

#### Get Task Result

```bash
grpcurl -plaintext -d '{
  "workflow_id": "wf-123456",
  "task_id": "task1"
}' localhost:9090 goclaw.v1.WorkflowService/GetTaskResult
```

### Streaming Operations

#### Watch Workflow

```bash
grpcurl -plaintext -d '{
  "workflow_id": "wf-123456"
}' localhost:9090 goclaw.v1.StreamingService/WatchWorkflow
```

Resume from sequence number:

```bash
grpcurl -plaintext -d '{
  "workflow_id": "wf-123456",
  "resume_from_sequence": 5
}' localhost:9090 goclaw.v1.StreamingService/WatchWorkflow
```

#### Watch Tasks

Watch all tasks:

```bash
grpcurl -plaintext -d '{
  "workflow_id": "wf-123456"
}' localhost:9090 goclaw.v1.StreamingService/WatchTasks
```

Watch specific tasks:

```bash
grpcurl -plaintext -d '{
  "workflow_id": "wf-123456",
  "task_ids": ["task1", "task2"]
}' localhost:9090 goclaw.v1.StreamingService/WatchTasks
```

Terminal events only:

```bash
grpcurl -plaintext -d '{
  "workflow_id": "wf-123456",
  "terminal_only": true
}' localhost:9090 goclaw.v1.StreamingService/WatchTasks
```

### Batch Operations

#### Submit Multiple Workflows

```bash
grpcurl -plaintext -d '{
  "requests": [
    {
      "name": "workflow-1",
      "tasks": [{"id": "t1", "name": "Task 1"}]
    },
    {
      "name": "workflow-2",
      "tasks": [{"id": "t2", "name": "Task 2"}]
    }
  ]
}' localhost:9090 goclaw.v1.BatchService/SubmitWorkflows
```

Atomic mode (all-or-nothing):

```bash
grpcurl -plaintext -d '{
  "requests": [
    {"name": "workflow-1", "tasks": [{"id": "t1", "name": "Task 1"}]},
    {"name": "workflow-2", "tasks": [{"id": "t2", "name": "Task 2"}]}
  ],
  "atomic": true
}' localhost:9090 goclaw.v1.BatchService/SubmitWorkflows
```

With idempotency key:

```bash
grpcurl -plaintext -d '{
  "requests": [
    {"name": "workflow-1", "tasks": [{"id": "t1", "name": "Task 1"}]}
  ],
  "idempotency_key": "unique-key-123"
}' localhost:9090 goclaw.v1.BatchService/SubmitWorkflows
```

#### Get Multiple Workflow Statuses

```bash
grpcurl -plaintext -d '{
  "workflow_ids": ["wf-123", "wf-456", "wf-789"]
}' localhost:9090 goclaw.v1.BatchService/GetWorkflowStatuses
```

#### Cancel Multiple Workflows

```bash
grpcurl -plaintext -d '{
  "workflow_ids": ["wf-123", "wf-456"],
  "force": false
}' localhost:9090 goclaw.v1.BatchService/CancelWorkflows
```

#### Get Multiple Task Results

```bash
grpcurl -plaintext -d '{
  "workflow_id": "wf-123",
  "task_ids": ["task1", "task2", "task3"]
}' localhost:9090 goclaw.v1.BatchService/GetTaskResults
```

### Admin Operations

#### Get Engine Status

```bash
grpcurl -plaintext -d '{}' localhost:9090 goclaw.v1.AdminService/GetEngineStatus
```

#### Update Configuration

```bash
grpcurl -plaintext -d '{
  "config_updates": {
    "log.level": "debug",
    "orchestration.max_agents": "2000"
  },
  "persist": false
}' localhost:9090 goclaw.v1.AdminService/UpdateConfig
```

Dry run mode:

```bash
grpcurl -plaintext -d '{
  "config_updates": {
    "log.level": "debug"
  },
  "dry_run": true
}' localhost:9090 goclaw.v1.AdminService/UpdateConfig
```

#### Manage Cluster

List nodes:

```bash
grpcurl -plaintext -d '{
  "operation": "CLUSTER_OPERATION_LIST"
}' localhost:9090 goclaw.v1.AdminService/ManageCluster
```

Add node:

```bash
grpcurl -plaintext -d '{
  "operation": "CLUSTER_OPERATION_ADD",
  "node_id": "node-2",
  "node_address": "192.168.1.100:9090"
}' localhost:9090 goclaw.v1.AdminService/ManageCluster
```

Remove node:

```bash
grpcurl -plaintext -d '{
  "operation": "CLUSTER_OPERATION_REMOVE",
  "node_id": "node-2",
  "confirmation": true
}' localhost:9090 goclaw.v1.AdminService/ManageCluster
```

#### Pause Workflows

```bash
grpcurl -plaintext -d '{
  "confirmation": true
}' localhost:9090 goclaw.v1.AdminService/PauseWorkflows
```

#### Resume Workflows

```bash
grpcurl -plaintext -d '{}' localhost:9090 goclaw.v1.AdminService/ResumeWorkflows
```

#### Purge Old Workflows

Dry run:

```bash
grpcurl -plaintext -d '{
  "age_threshold_hours": 168,
  "dry_run": true
}' localhost:9090 goclaw.v1.AdminService/PurgeWorkflows
```

Actual purge:

```bash
grpcurl -plaintext -d '{
  "age_threshold_hours": 168,
  "confirmation": true
}' localhost:9090 goclaw.v1.AdminService/PurgeWorkflows
```

#### Get Lane Statistics

All lanes:

```bash
grpcurl -plaintext -d '{}' localhost:9090 goclaw.v1.AdminService/GetLaneStats
```

Specific lane:

```bash
grpcurl -plaintext -d '{
  "lane_name": "default"
}' localhost:9090 goclaw.v1.AdminService/GetLaneStats
```

#### Export Metrics

JSON format:

```bash
grpcurl -plaintext -d '{
  "format": "METRICS_FORMAT_JSON"
}' localhost:9090 goclaw.v1.AdminService/ExportMetrics
```

Prometheus format:

```bash
grpcurl -plaintext -d '{
  "format": "METRICS_FORMAT_PROMETHEUS"
}' localhost:9090 goclaw.v1.AdminService/ExportMetrics
```

With prefix filter:

```bash
grpcurl -plaintext -d '{
  "format": "METRICS_FORMAT_JSON",
  "prefix_filter": "goclaw_workflow"
}' localhost:9090 goclaw.v1.AdminService/ExportMetrics
```

#### Get Debug Information

Goroutine profile:

```bash
grpcurl -plaintext -d '{
  "type": "DEBUG_INFO_TYPE_GOROUTINE"
}' localhost:9090 goclaw.v1.AdminService/GetDebugInfo
```

Heap profile:

```bash
grpcurl -plaintext -d '{
  "type": "DEBUG_INFO_TYPE_HEAP"
}' localhost:9090 goclaw.v1.AdminService/GetDebugInfo
```

CPU profile:

```bash
grpcurl -plaintext -d '{
  "type": "DEBUG_INFO_TYPE_CPU",
  "duration_seconds": 30
}' localhost:9090 goclaw.v1.AdminService/GetDebugInfo
```

### Health Check

```bash
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

Watch health status:

```bash
grpcurl -plaintext -d '{"service": "goclaw.v1.WorkflowService"}' \
  localhost:9090 grpc.health.v1.Health/Watch
```

## Using with TLS/mTLS

### TLS (Server Authentication)

```bash
grpcurl -cacert ./certs/ca.crt \
  -d '{"workflow_id": "wf-123"}' \
  localhost:9090 goclaw.v1.WorkflowService/GetWorkflowStatus
```

### mTLS (Mutual Authentication)

```bash
grpcurl -cacert ./certs/ca.crt \
  -cert ./certs/client.crt \
  -key ./certs/client.key \
  -d '{"workflow_id": "wf-123"}' \
  localhost:9090 goclaw.v1.WorkflowService/GetWorkflowStatus
```

## Authentication

### Bearer Token

```bash
grpcurl -plaintext \
  -H "authorization: Bearer your-token-here" \
  -d '{"workflow_id": "wf-123"}' \
  localhost:9090 goclaw.v1.WorkflowService/GetWorkflowStatus
```

## Error Handling

gRPC uses standard status codes. Common codes:

- `OK` (0): Success
- `CANCELLED` (1): Operation cancelled
- `INVALID_ARGUMENT` (3): Invalid request parameters
- `NOT_FOUND` (5): Resource not found
- `ALREADY_EXISTS` (6): Resource already exists
- `PERMISSION_DENIED` (7): Insufficient permissions
- `RESOURCE_EXHAUSTED` (8): Rate limit exceeded
- `FAILED_PRECONDITION` (9): Operation rejected (e.g., missing confirmation)
- `UNAVAILABLE` (14): Service unavailable
- `UNAUTHENTICATED` (16): Authentication required

Example error response:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "workflow not found: wf-123"
  }
}
```

## Performance Tips

1. **Use streaming for real-time updates** instead of polling
2. **Batch operations** for multiple workflows/tasks
3. **Enable connection pooling** in clients
4. **Use compression** for large payloads
5. **Set appropriate timeouts** based on operation type
6. **Implement retry logic** with exponential backoff

## Debugging

### Enable verbose logging

```bash
GRPC_GO_LOG_VERBOSITY_LEVEL=99 GRPC_GO_LOG_SEVERITY_LEVEL=info \
  grpcurl -plaintext localhost:9090 list
```

### Inspect metadata

```bash
grpcurl -plaintext -v \
  -d '{"workflow_id": "wf-123"}' \
  localhost:9090 goclaw.v1.WorkflowService/GetWorkflowStatus
```

### Test connectivity

```bash
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

## See Also

- [Client SDK Examples](./client-sdk-examples.md)
- [TLS/mTLS Setup](./tls-setup.md)
- [API Reference](../api/grpc-api.md)
