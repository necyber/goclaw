# API Examples with cURL

This document provides examples of using the Goclaw API with cURL.

## Health Check Endpoints

### Check Service Health (Liveness)
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok"
}
```

### Check Service Readiness
```bash
curl http://localhost:8080/ready
```

Response:
```json
{
  "ready": true
}
```

### Get Detailed Status
```bash
curl http://localhost:8080/status
```

Response:
```json
{
  "state": "running",
  "uptime": 3600
}
```

## Workflow Endpoints

### Submit a New Workflow
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "data-processing-workflow",
    "description": "Process customer data and generate reports",
    "tasks": [
      {
        "id": "task-1",
        "name": "Fetch data from API",
        "type": "http",
        "timeout": 300,
        "retries": 3
      },
      {
        "id": "task-2",
        "name": "Process data",
        "type": "script",
        "depends_on": ["task-1"],
        "timeout": 600
      },
      {
        "id": "task-3",
        "name": "Generate report",
        "type": "function",
        "depends_on": ["task-2"],
        "timeout": 300
      }
    ],
    "metadata": {
      "environment": "production",
      "team": "data-engineering"
    }
  }'
```

Response:
```json
{
  "id": "wf-123e4567-e89b-12d3-a456-426614174000",
  "name": "data-processing-workflow",
  "status": "pending",
  "created_at": "2026-02-25T10:00:00Z",
  "message": "Workflow submitted successfully"
}
```

### Get Workflow Status
```bash
curl http://localhost:8080/api/v1/workflows/wf-123e4567-e89b-12d3-a456-426614174000
```

Response:
```json
{
  "id": "wf-123e4567-e89b-12d3-a456-426614174000",
  "name": "data-processing-workflow",
  "status": "running",
  "created_at": "2026-02-25T10:00:00Z",
  "started_at": "2026-02-25T10:00:01Z",
  "tasks": [
    {
      "id": "task-1",
      "name": "Fetch data from API",
      "status": "completed",
      "started_at": "2026-02-25T10:00:01Z",
      "completed_at": "2026-02-25T10:00:05Z"
    },
    {
      "id": "task-2",
      "name": "Process data",
      "status": "running",
      "started_at": "2026-02-25T10:00:05Z"
    },
    {
      "id": "task-3",
      "name": "Generate report",
      "status": "pending"
    }
  ],
  "metadata": {
    "environment": "production",
    "team": "data-engineering"
  }
}
```

### List All Workflows
```bash
# List all workflows
curl http://localhost:8080/api/v1/workflows

# List with pagination
curl "http://localhost:8080/api/v1/workflows?limit=10&offset=0"

# Filter by status
curl "http://localhost:8080/api/v1/workflows?status=running"
```

Response:
```json
{
  "workflows": [
    {
      "id": "wf-123e4567-e89b-12d3-a456-426614174000",
      "name": "data-processing-workflow",
      "status": "running",
      "created_at": "2026-02-25T10:00:00Z",
      "task_count": 3
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

### Cancel a Workflow
```bash
curl -X POST http://localhost:8080/api/v1/workflows/wf-123e4567-e89b-12d3-a456-426614174000/cancel
```

Response:
```json
{
  "message": "Workflow cancelled successfully"
}
```

### Get Task Result
```bash
curl http://localhost:8080/api/v1/workflows/wf-123e4567-e89b-12d3-a456-426614174000/tasks/task-1/result
```

Response:
```json
{
  "workflow_id": "wf-123e4567-e89b-12d3-a456-426614174000",
  "task_id": "task-1",
  "status": "completed",
  "result": {
    "data": "processed data here"
  },
  "completed_at": "2026-02-25T10:00:05Z"
}
```

## Error Responses

### 400 Bad Request
```json
{
  "error": {
    "code": "BAD_REQUEST",
    "message": "Invalid request body",
    "request_id": "req-123456"
  }
}
```

### 404 Not Found
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Workflow not found",
    "request_id": "req-123456"
  }
}
```

### 409 Conflict
```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Workflow cannot be cancelled",
    "request_id": "req-123456"
  }
}
```

### 500 Internal Server Error
```json
{
  "error": {
    "code": "INTERNAL_SERVER_ERROR",
    "message": "Failed to submit workflow",
    "request_id": "req-123456"
  }
}
```

## Using with Authentication (Future)

When authentication is implemented, include the token in the Authorization header:

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{ ... }'
```

## Swagger UI

Access the interactive API documentation at:
```
http://localhost:8080/swagger/index.html
```

The Swagger UI provides:
- Interactive API exploration
- Request/response examples
- Schema definitions
- Try-it-out functionality
