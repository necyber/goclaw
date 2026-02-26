## Why

As Goclaw evolves toward distributed multi-agent orchestration, service-to-service communication requires a more efficient protocol than HTTP/JSON. gRPC provides type-safe contracts, bidirectional streaming for real-time updates, and better performance for internal cluster communication. This enables future distributed mode (Phase 2) and prepares the foundation for cluster coordination.

## What Changes

- Add gRPC server on port 9090 for internal service-to-service communication
- Implement full workflow management API via gRPC (submit, list, status, cancel, task results)
- Add streaming endpoints for real-time workflow status updates, task progress, and log streaming
- Implement batch operations for bulk workflow submission, status queries, and cancellation
- Add admin operations for engine control, configuration updates, and cluster management
- Provide authentication/authorization via mTLS and token-based auth
- Implement gRPC interceptors for request logging, metrics collection, and distributed tracing
- Add gRPC health check protocol and server reflection for debugging
- Create Go client SDK with connection pooling, retry logic, and usage examples
- Keep existing HTTP API unchanged for external client compatibility

## Capabilities

### New Capabilities
- `proto-definitions`: Protocol Buffer definitions for workflow, task, and admin services
- `grpc-server`: gRPC server implementation with service handlers and lifecycle management
- `grpc-client`: Go client SDK with connection management and helper methods
- `streaming-support`: Bidirectional streaming for real-time workflow and task updates
- `interceptors`: Authentication, logging, metrics, and tracing interceptors
- `batch-operations`: Bulk workflow submission, status queries, and cancellation endpoints
- `admin-api`: Engine control, configuration management, and cluster coordination endpoints

### Modified Capabilities
<!-- No existing capabilities require requirement changes -->

## Impact

- **New package**: `pkg/grpc/` for gRPC server, handlers, and interceptors
- **New package**: `pkg/grpc/client/` for Go client SDK
- **New directory**: `api/proto/` for Protocol Buffer definitions
- **Modified**: `cmd/goclaw/main.go` to start gRPC server alongside HTTP server
- **Modified**: `config/` to add gRPC server configuration (port, TLS, auth)
- **Modified**: `pkg/engine/` to support streaming status updates and batch operations
- **Dependencies**: Add `google.golang.org/grpc`, `google.golang.org/protobuf`, `grpc-ecosystem/go-grpc-middleware`
- **Build**: Add protoc code generation to Makefile
- **Documentation**: Add gRPC API examples and client SDK usage guide
