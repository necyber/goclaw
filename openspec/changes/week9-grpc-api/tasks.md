## 1. Protocol Buffer Definitions

- [x] 1.1 Create `api/proto/goclaw/v1/` directory structure
- [x] 1.2 Define common message types in `common.proto` (pagination, error, timestamps)
- [x] 1.3 Define workflow messages and WorkflowService in `workflow.proto`
- [x] 1.4 Define streaming messages and StreamingService in `streaming.proto`
- [x] 1.5 Define batch messages and BatchService in `batch.proto`
- [x] 1.6 Define admin messages and AdminService in `admin.proto`
- [x] 1.7 Add `buf.yaml` configuration for buf build tool
- [x] 1.8 Add Makefile targets for proto code generation (`make proto`)
- [ ] 1.9 Generate Go code and verify compilation

## 2. gRPC Server Infrastructure

- [x] 2.1 Create `pkg/grpc/server.go` with server initialization and lifecycle management
- [x] 2.2 Add TLS/mTLS configuration support in server options
- [x] 2.3 Implement graceful shutdown with request draining
- [x] 2.4 Add server reflection support (configurable)
- [x] 2.5 Implement gRPC health check service (grpc.health.v1)
- [x] 2.6 Add connection management (limits, idle timeout, keepalive)
- [x] 2.7 Create `pkg/grpc/config.go` for gRPC-specific configuration
- [x] 2.8 Update `config/config.go` to include gRPC server settings
- [ ] 2.9 Modify `cmd/goclaw/main.go` to start gRPC server alongside HTTP

## 3. Interceptors

- [ ] 3.1 Create `pkg/grpc/interceptors/` package structure
- [ ] 3.2 Implement recovery interceptor with panic logging
- [ ] 3.3 Implement request ID interceptor (generate/propagate)
- [ ] 3.4 Implement authentication interceptor (token + mTLS)
- [ ] 3.5 Implement authorization interceptor (role-based)
- [ ] 3.6 Implement rate limiting interceptor
- [ ] 3.7 Implement validation interceptor for request payloads
- [ ] 3.8 Implement logging interceptor (request/response logging)
- [ ] 3.9 Implement metrics interceptor (counters, histograms, gauges)
- [ ] 3.10 Implement tracing interceptor (trace context propagation)
- [ ] 3.11 Create interceptor chain builder with correct ordering
- [ ] 3.12 Add unit tests for each interceptor

## 4. Workflow Service Handlers

- [ ] 4.1 Create `pkg/grpc/handlers/` package structure
- [ ] 4.2 Implement WorkflowService.SubmitWorkflow handler
- [ ] 4.3 Implement WorkflowService.ListWorkflows handler with pagination
- [ ] 4.4 Implement WorkflowService.GetWorkflowStatus handler
- [ ] 4.5 Implement WorkflowService.CancelWorkflow handler
- [ ] 4.6 Implement WorkflowService.GetTaskResult handler
- [ ] 4.7 Add request validation for all workflow handlers
- [ ] 4.8 Add proper gRPC status code mapping for errors
- [ ] 4.9 Add unit tests for workflow handlers

## 5. Streaming Support

- [ ] 5.1 Add observer pattern to `pkg/engine/` for state change notifications
- [ ] 5.2 Create `pkg/grpc/streaming/` package for stream management
- [ ] 5.3 Implement subscriber registry for workflow watchers
- [ ] 5.4 Implement StreamingService.WatchWorkflow (server streaming)
- [ ] 5.5 Implement StreamingService.WatchTasks (server streaming)
- [ ] 5.6 Implement StreamingService.StreamLogs (bidirectional streaming)
- [ ] 5.7 Add backpressure handling (buffer limits, slow consumer detection)
- [ ] 5.8 Add stream lifecycle management (subscribe, unsubscribe, cleanup)
- [ ] 5.9 Implement stream reconnection with sequence numbers
- [ ] 5.10 Add unit tests for streaming handlers

## 6. Batch Operations

- [ ] 6.1 Implement BatchService.SubmitWorkflows handler
- [ ] 6.2 Implement BatchService.GetWorkflowStatuses handler
- [ ] 6.3 Implement BatchService.CancelWorkflows handler
- [ ] 6.4 Implement BatchService.GetTaskResults handler
- [ ] 6.5 Add parallel processing with worker pools
- [ ] 6.6 Implement atomic batch mode (all-or-nothing)
- [ ] 6.7 Add idempotency key support for batch submissions
- [ ] 6.8 Implement batch size limits and validation
- [ ] 6.9 Add pagination for large batch responses
- [ ] 6.10 Add unit tests for batch handlers

## 7. Admin API

- [ ] 7.1 Implement AdminService.GetEngineStatus handler
- [ ] 7.2 Implement AdminService.UpdateConfig handler
- [ ] 7.3 Implement AdminService.ManageCluster handler (list, add, remove nodes)
- [ ] 7.4 Implement AdminService.PauseWorkflows handler
- [ ] 7.5 Implement AdminService.ResumeWorkflows handler
- [ ] 7.6 Implement AdminService.PurgeWorkflows handler
- [ ] 7.7 Implement AdminService.GetLaneStats handler
- [ ] 7.8 Implement AdminService.ExportMetrics handler
- [ ] 7.9 Implement AdminService.GetDebugInfo handler (goroutines, heap, CPU)
- [ ] 7.10 Add admin role verification for all admin handlers
- [ ] 7.11 Add audit logging for admin operations
- [ ] 7.12 Add confirmation flag for destructive operations
- [ ] 7.13 Add unit tests for admin handlers

## 8. Go Client SDK

- [ ] 8.1 Create `pkg/grpc/client/` package structure
- [ ] 8.2 Implement client initialization with connection options
- [ ] 8.3 Add TLS/mTLS connection support
- [ ] 8.4 Implement connection pooling and health checking
- [ ] 8.5 Implement automatic retry with exponential backoff
- [ ] 8.6 Add workflow operation methods (submit, list, status, cancel, result)
- [ ] 8.7 Add streaming operation methods (watch workflow, watch tasks)
- [ ] 8.8 Add batch operation methods (submit, statuses, cancel)
- [ ] 8.9 Add admin operation methods (status, config, cluster)
- [ ] 8.10 Implement context support (cancellation, deadlines, metadata)
- [ ] 8.11 Add typed error handling
- [ ] 8.12 Add unit tests for client SDK

## 9. Integration Testing

- [ ] 9.1 Create `pkg/grpc/integration_test.go` for end-to-end tests
- [ ] 9.2 Add integration tests for workflow operations
- [ ] 9.3 Add integration tests for streaming operations
- [ ] 9.4 Add integration tests for batch operations
- [ ] 9.5 Add integration tests for admin operations
- [ ] 9.6 Add integration tests for authentication/authorization
- [ ] 9.7 Add integration tests for TLS/mTLS connections
- [ ] 9.8 Add performance benchmarks for gRPC endpoints

## 10. Documentation and Examples

- [ ] 10.1 Add gRPC configuration section to `config/config.example.yaml`
- [ ] 10.2 Create `docs/examples/grpc-examples.md` with usage examples
- [ ] 10.3 Create `docs/examples/client-sdk-examples.md` with SDK usage
- [ ] 10.4 Update README.md with gRPC API section
- [ ] 10.5 Add grpcurl examples for debugging
- [ ] 10.6 Document TLS/mTLS certificate setup
