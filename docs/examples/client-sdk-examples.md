# Go Client SDK Examples

This document provides examples for using the Goclaw Go client SDK.

## Installation

```bash
go get github.com/goclaw/goclaw/pkg/grpc/client
```

## Basic Usage

### Creating a Client

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/goclaw/goclaw/pkg/grpc/client"
)

func main() {
    // Create client with default options
    c, err := client.NewClient("localhost:9090")
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer c.Close()

    // Use the client...
}
```

### Client Options

```go
// With custom timeout
c, err := client.NewClient("localhost:9090",
    client.WithTimeout(30*time.Second),
)

// With TLS
c, err := client.NewClient("localhost:9090",
    client.WithTLS("./certs/ca.crt", "", ""),
)

// With mTLS
c, err := client.NewClient("localhost:9090",
    client.WithTLS("./certs/ca.crt", "./certs/client.crt", "./certs/client.key"),
)

// With authentication token
c, err := client.NewClient("localhost:9090",
    client.WithToken("your-bearer-token"),
)

// With retry configuration
c, err := client.NewClient("localhost:9090",
    client.WithRetry(3, 100*time.Millisecond, 5*time.Second),
)

// With connection pool
c, err := client.NewClient("localhost:9090",
    client.WithConnectionPool(10),
)

// Combine multiple options
c, err := client.NewClient("localhost:9090",
    client.WithTimeout(30*time.Second),
    client.WithTLS("./certs/ca.crt", "", ""),
    client.WithToken("your-token"),
    client.WithRetry(3, 100*time.Millisecond, 5*time.Second),
)
```

## Workflow Operations

### Submit Workflow

```go
ctx := context.Background()

tasks := []*client.TaskDefinition{
    {
        ID:           "task1",
        Name:         "First Task",
        Dependencies: []string{},
        Metadata: map[string]string{
            "type": "compute",
        },
    },
    {
        ID:           "task2",
        Name:         "Second Task",
        Dependencies: []string{"task1"},
        Metadata: map[string]string{
            "type": "io",
        },
    },
}

workflowID, err := c.SubmitWorkflow(ctx, "my-workflow", tasks)
if err != nil {
    log.Fatalf("Failed to submit workflow: %v", err)
}

log.Printf("Workflow submitted: %s", workflowID)
```

### List Workflows

```go
// List all workflows
workflows, nextToken, err := c.ListWorkflows(ctx, "", 50, "")
if err != nil {
    log.Fatalf("Failed to list workflows: %v", err)
}

for _, wf := range workflows {
    log.Printf("Workflow: %s, Status: %s", wf.WorkflowID, wf.Status)
}

// Pagination
if nextToken != "" {
    moreWorkflows, _, err := c.ListWorkflows(ctx, "", 50, nextToken)
    // Process more workflows...
}

// Filter by status
runningWorkflows, _, err := c.ListWorkflows(ctx, "WORKFLOW_STATUS_RUNNING", 50, "")
```

### Get Workflow Status

```go
status, err := c.GetWorkflowStatus(ctx, workflowID)
if err != nil {
    log.Fatalf("Failed to get workflow status: %v", err)
}

log.Printf("Workflow: %s", status.WorkflowID)
log.Printf("Status: %s", status.Status)
log.Printf("Created: %v", status.CreatedAt)
log.Printf("Updated: %v", status.UpdatedAt)

for _, task := range status.Tasks {
    log.Printf("  Task %s: %s", task.TaskID, task.Status)
    if task.ErrorMessage != "" {
        log.Printf("    Error: %s", task.ErrorMessage)
    }
}
```

### Cancel Workflow

```go
// Graceful cancel
err := c.CancelWorkflow(ctx, workflowID, false)
if err != nil {
    log.Fatalf("Failed to cancel workflow: %v", err)
}

// Force cancel
err = c.CancelWorkflow(ctx, workflowID, true)
```

### Get Task Result

```go
result, err := c.GetTaskResult(ctx, workflowID, "task1")
if err != nil {
    log.Fatalf("Failed to get task result: %v", err)
}

log.Printf("Task: %s", result.TaskID)
log.Printf("Status: %s", result.Status)
log.Printf("Result: %s", string(result.ResultData))
```

## Streaming Operations

### Watch Workflow

```go
// Watch workflow events
eventChan, errChan, err := c.WatchWorkflow(ctx, workflowID, 0)
if err != nil {
    log.Fatalf("Failed to watch workflow: %v", err)
}

for {
    select {
    case event := <-eventChan:
        log.Printf("Workflow event: %s at seq %d", event.Status, event.SequenceNumber)
        if event.ErrorMessage != "" {
            log.Printf("  Error: %s", event.ErrorMessage)
        }
    case err := <-errChan:
        if err != nil {
            log.Printf("Watch error: %v", err)
        }
        return
    case <-ctx.Done():
        return
    }
}
```

### Watch Tasks

```go
// Watch all tasks
taskEventChan, errChan, err := c.WatchTasks(ctx, workflowID, nil, false, 0)
if err != nil {
    log.Fatalf("Failed to watch tasks: %v", err)
}

for {
    select {
    case event := <-taskEventChan:
        log.Printf("Task %s: %s", event.TaskID, event.Status)
        if event.Progress > 0 {
            log.Printf("  Progress: %.1f%%", event.Progress)
        }
    case err := <-errChan:
        if err != nil {
            log.Printf("Watch error: %v", err)
        }
        return
    case <-ctx.Done():
        return
    }
}

// Watch specific tasks only
taskIDs := []string{"task1", "task2"}
taskEventChan, errChan, err = c.WatchTasks(ctx, workflowID, taskIDs, false, 0)

// Watch terminal events only
taskEventChan, errChan, err = c.WatchTasks(ctx, workflowID, nil, true, 0)
```

### Stream Logs

```go
// Stream logs
logChan, errChan, err := c.StreamLogs(ctx, workflowID, "", "")
if err != nil {
    log.Fatalf("Failed to stream logs: %v", err)
}

for {
    select {
    case logEntry := <-logChan:
        log.Printf("[%s] %s: %s", logEntry.Level, logEntry.TaskID, logEntry.Message)
    case err := <-errChan:
        if err != nil {
            log.Printf("Stream error: %v", err)
        }
        return
    case <-ctx.Done():
        return
    }
}

// Filter by task
logChan, errChan, err = c.StreamLogs(ctx, workflowID, "task1", "")

// Filter by log level
logChan, errChan, err = c.StreamLogs(ctx, workflowID, "", "ERROR")
```

## Batch Operations

### Submit Multiple Workflows

```go
requests := []*client.SubmitWorkflowRequest{
    {
        Name: "workflow-1",
        Tasks: []*client.TaskDefinition{
            {ID: "t1", Name: "Task 1"},
        },
    },
    {
        Name: "workflow-2",
        Tasks: []*client.TaskDefinition{
            {ID: "t2", Name: "Task 2"},
        },
    },
}

// Submit in parallel
responses, err := c.SubmitWorkflows(ctx, requests, false, "", false)
if err != nil {
    log.Fatalf("Failed to submit workflows: %v", err)
}

for _, resp := range responses {
    if resp.Error != nil {
        log.Printf("Failed: %s", resp.Error.Message)
    } else {
        log.Printf("Submitted: %s", resp.WorkflowID)
    }
}

// Atomic mode (all-or-nothing)
responses, err = c.SubmitWorkflows(ctx, requests, true, "", false)

// With idempotency key
responses, err = c.SubmitWorkflows(ctx, requests, false, "unique-key-123", false)

// Ordered submission
responses, err = c.SubmitWorkflows(ctx, requests, false, "", true)
```

### Get Multiple Workflow Statuses

```go
workflowIDs := []string{"wf-123", "wf-456", "wf-789"}

statuses, err := c.GetWorkflowStatuses(ctx, workflowIDs, 100, "")
if err != nil {
    log.Fatalf("Failed to get statuses: %v", err)
}

for _, status := range statuses {
    if status.Error != nil {
        log.Printf("Error for %s: %s", status.WorkflowID, status.Error.Message)
    } else {
        log.Printf("%s: %s", status.WorkflowID, status.Status)
    }
}
```

### Cancel Multiple Workflows

```go
workflowIDs := []string{"wf-123", "wf-456"}

results, err := c.CancelWorkflows(ctx, workflowIDs, false, 30*time.Second)
if err != nil {
    log.Fatalf("Failed to cancel workflows: %v", err)
}

for _, result := range results {
    if result.Error != nil {
        log.Printf("Failed to cancel %s: %s", result.WorkflowID, result.Error.Message)
    } else {
        log.Printf("Cancelled: %s", result.WorkflowID)
    }
}
```

### Get Multiple Task Results

```go
taskIDs := []string{"task1", "task2", "task3"}

results, err := c.GetTaskResults(ctx, workflowID, taskIDs, 100, "")
if err != nil {
    log.Fatalf("Failed to get task results: %v", err)
}

for _, result := range results {
    if result.Error != nil {
        log.Printf("Error for %s: %s", result.TaskID, result.Error.Message)
    } else {
        log.Printf("%s: %s", result.TaskID, result.Status)
    }
}
```

## Admin Operations

### Get Engine Status

```go
status, err := c.GetEngineStatus(ctx)
if err != nil {
    log.Fatalf("Failed to get engine status: %v", err)
}

log.Printf("State: %s", status.State)
log.Printf("Healthy: %v", status.Healthy)
log.Printf("Active workflows: %d", status.Metrics.ActiveWorkflows)
log.Printf("Completed workflows: %d", status.Metrics.CompletedWorkflows)
log.Printf("Running tasks: %d", status.Metrics.RunningTasks)
log.Printf("Memory usage: %d MB", status.Metrics.MemoryUsageBytes/1024/1024)
log.Printf("Goroutines: %d", status.Metrics.GoroutineCount)
log.Printf("CPU usage: %.2f%%", status.Metrics.CPUUsagePercent)
```

### Update Configuration

```go
updates := map[string]string{
    "log.level":                "debug",
    "orchestration.max_agents": "2000",
}

// Dry run
applied, err := c.UpdateConfig(ctx, updates, false, true)
if err != nil {
    log.Fatalf("Config validation failed: %v", err)
}

// Apply changes
applied, err = c.UpdateConfig(ctx, updates, true, false)
if err != nil {
    log.Fatalf("Failed to update config: %v", err)
}

for key, value := range applied {
    log.Printf("Updated %s = %s", key, value)
}
```

### Manage Cluster

```go
// List nodes
nodes, err := c.ListClusterNodes(ctx)
if err != nil {
    log.Fatalf("Failed to list nodes: %v", err)
}

for _, node := range nodes {
    log.Printf("Node: %s (%s) - %s, Healthy: %v",
        node.NodeID, node.Address, node.Role, node.Healthy)
}

// Add node
err = c.AddClusterNode(ctx, "node-2", "192.168.1.100:9090")
if err != nil {
    log.Fatalf("Failed to add node: %v", err)
}

// Remove node
err = c.RemoveClusterNode(ctx, "node-2", true)
if err != nil {
    log.Fatalf("Failed to remove node: %v", err)
}
```

### Pause/Resume Workflows

```go
// Pause all workflows
count, err := c.PauseWorkflows(ctx, true)
if err != nil {
    log.Fatalf("Failed to pause workflows: %v", err)
}
log.Printf("Paused %d workflows", count)

// Resume all workflows
count, err = c.ResumeWorkflows(ctx)
if err != nil {
    log.Fatalf("Failed to resume workflows: %v", err)
}
log.Printf("Resumed %d workflows", count)
```

### Purge Old Workflows

```go
// Dry run - see what would be purged
count, err := c.PurgeWorkflows(ctx, 168, false, true) // 7 days
if err != nil {
    log.Fatalf("Failed to check purge: %v", err)
}
log.Printf("Would purge %d workflows", count)

// Actually purge
count, err = c.PurgeWorkflows(ctx, 168, true, false)
if err != nil {
    log.Fatalf("Failed to purge workflows: %v", err)
}
log.Printf("Purged %d workflows", count)
```

### Get Lane Statistics

```go
// All lanes
stats, err := c.GetLaneStats(ctx, "")
if err != nil {
    log.Fatalf("Failed to get lane stats: %v", err)
}

for _, lane := range stats {
    log.Printf("Lane: %s", lane.LaneName)
    log.Printf("  Queue depth: %d", lane.QueueDepth)
    log.Printf("  Workers: %d", lane.WorkerCount)
    log.Printf("  Throughput: %.2f/s", lane.ThroughputPerSec)
    log.Printf("  Error rate: %.2f%%", lane.ErrorRate*100)
}

// Specific lane
stats, err = c.GetLaneStats(ctx, "default")
```

### Export Metrics

```go
// JSON format
metricsJSON, err := c.ExportMetrics(ctx, "json", "")
if err != nil {
    log.Fatalf("Failed to export metrics: %v", err)
}
log.Printf("Metrics: %s", metricsJSON)

// Prometheus format
metricsPrometheus, err := c.ExportMetrics(ctx, "prometheus", "")

// With prefix filter
metricsFiltered, err := c.ExportMetrics(ctx, "json", "goclaw_workflow")
```

### Get Debug Information

```go
// Goroutine profile
goroutineData, err := c.GetDebugInfo(ctx, "goroutine", 0)
if err != nil {
    log.Fatalf("Failed to get goroutine info: %v", err)
}
log.Printf("Goroutine profile: %d bytes", len(goroutineData))

// Heap profile
heapData, err := c.GetDebugInfo(ctx, "heap", 0)

// CPU profile (30 seconds)
cpuData, err := c.GetDebugInfo(ctx, "cpu", 30)
```

## Advanced Usage

### Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

workflowID, err := c.SubmitWorkflow(ctx, "my-workflow", tasks)
```

### Context with Cancellation

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    // Cancel after some condition
    time.Sleep(5 * time.Second)
    cancel()
}()

err := c.WatchWorkflow(ctx, workflowID, 0)
```

### Custom Metadata

```go
ctx := context.Background()
ctx = metadata.AppendToOutgoingContext(ctx,
    "x-request-id", "req-123",
    "x-user-id", "user-456",
)

workflowID, err := c.SubmitWorkflow(ctx, "my-workflow", tasks)
```

### Error Handling

```go
workflowID, err := c.SubmitWorkflow(ctx, "my-workflow", tasks)
if err != nil {
    if grpcErr, ok := status.FromError(err); ok {
        switch grpcErr.Code() {
        case codes.InvalidArgument:
            log.Printf("Invalid request: %s", grpcErr.Message())
        case codes.NotFound:
            log.Printf("Resource not found: %s", grpcErr.Message())
        case codes.Unauthenticated:
            log.Printf("Authentication required: %s", grpcErr.Message())
        case codes.PermissionDenied:
            log.Printf("Permission denied: %s", grpcErr.Message())
        case codes.ResourceExhausted:
            log.Printf("Rate limit exceeded: %s", grpcErr.Message())
        case codes.Unavailable:
            log.Printf("Service unavailable: %s", grpcErr.Message())
        default:
            log.Printf("gRPC error: %s", grpcErr.Message())
        }
    } else {
        log.Printf("Error: %v", err)
    }
}
```

### Connection Health Check

```go
healthy, err := c.HealthCheck(ctx)
if err != nil {
    log.Fatalf("Health check failed: %v", err)
}

if healthy {
    log.Println("Server is healthy")
} else {
    log.Println("Server is unhealthy")
}
```

## Complete Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/goclaw/goclaw/pkg/grpc/client"
)

func main() {
    // Create client
    c, err := client.NewClient("localhost:9090",
        client.WithTimeout(30*time.Second),
        client.WithRetry(3, 100*time.Millisecond, 5*time.Second),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer c.Close()

    ctx := context.Background()

    // Submit workflow
    tasks := []*client.TaskDefinition{
        {ID: "task1", Name: "First Task"},
        {ID: "task2", Name: "Second Task", Dependencies: []string{"task1"}},
    }

    workflowID, err := c.SubmitWorkflow(ctx, "example-workflow", tasks)
    if err != nil {
        log.Fatalf("Failed to submit workflow: %v", err)
    }
    log.Printf("Submitted workflow: %s", workflowID)

    // Watch workflow progress
    eventChan, errChan, err := c.WatchWorkflow(ctx, workflowID, 0)
    if err != nil {
        log.Fatalf("Failed to watch workflow: %v", err)
    }

    for {
        select {
        case event := <-eventChan:
            log.Printf("Workflow status: %s", event.Status)
            if event.Status == "WORKFLOW_STATUS_COMPLETED" ||
               event.Status == "WORKFLOW_STATUS_FAILED" {
                goto done
            }
        case err := <-errChan:
            if err != nil {
                log.Fatalf("Watch error: %v", err)
            }
            goto done
        }
    }

done:
    // Get final status
    status, err := c.GetWorkflowStatus(ctx, workflowID)
    if err != nil {
        log.Fatalf("Failed to get status: %v", err)
    }

    log.Printf("Final status: %s", status.Status)
    for _, task := range status.Tasks {
        log.Printf("  Task %s: %s", task.TaskID, task.Status)
    }
}
```

## See Also

- [gRPC API Examples](./grpc-examples.md)
- [TLS/mTLS Setup](./tls-setup.md)
- [API Reference](../api/grpc-api.md)
