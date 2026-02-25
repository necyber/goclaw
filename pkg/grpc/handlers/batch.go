package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// MaxBatchSize is the maximum number of items in a batch request
	MaxBatchSize = 1000
	// DefaultWorkerPoolSize is the default number of workers for parallel processing
	DefaultWorkerPoolSize = 10
)

// BatchServiceServer implements the gRPC BatchService
type BatchServiceServer struct {
	pb.UnimplementedBatchServiceServer
	engine           WorkflowEngine
	workerPoolSize   int
	idempotencyCache *IdempotencyCache
}

// NewBatchServiceServer creates a new batch service server
func NewBatchServiceServer(engine WorkflowEngine) *BatchServiceServer {
	return &BatchServiceServer{
		engine:           engine,
		workerPoolSize:   DefaultWorkerPoolSize,
		idempotencyCache: NewIdempotencyCache(time.Hour), // 1 hour TTL
	}
}

// SetWorkerPoolSize sets the worker pool size for parallel processing
func (s *BatchServiceServer) SetWorkerPoolSize(size int) {
	if size > 0 {
		s.workerPoolSize = size
	}
}

// SubmitWorkflows handles batch workflow submission
func (s *BatchServiceServer) SubmitWorkflows(ctx context.Context, req *pb.SubmitWorkflowsRequest) (*pb.SubmitWorkflowsResponse, error) {
	if req == nil || len(req.Workflows) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one workflow is required")
	}

	// Validate batch size
	if len(req.Workflows) > MaxBatchSize {
		return nil, status.Errorf(codes.InvalidArgument, "batch size exceeds maximum of %d", MaxBatchSize)
	}

	// Check idempotency key
	if req.IdempotencyKey != "" {
		if cachedResp := s.idempotencyCache.Get(req.IdempotencyKey); cachedResp != nil {
			return cachedResp.(*pb.SubmitWorkflowsResponse), nil
		}
	}

	// Atomic mode: all-or-nothing
	if req.Atomic {
		return s.submitWorkflowsAtomic(ctx, req)
	}

	// Non-atomic mode: best-effort with parallel processing
	return s.submitWorkflowsParallel(ctx, req)
}

// submitWorkflowsAtomic submits workflows in atomic mode (all-or-nothing)
func (s *BatchServiceServer) submitWorkflowsAtomic(ctx context.Context, req *pb.SubmitWorkflowsRequest) (*pb.SubmitWorkflowsResponse, error) {
	results := make([]*pb.WorkflowSubmissionResult, len(req.Workflows))
	submittedIDs := make([]string, 0, len(req.Workflows))

	// First pass: validate all workflows
	for i, wf := range req.Workflows {
		if wf.Name == "" {
			return &pb.SubmitWorkflowsResponse{
				Error: &pb.Error{
					Code:    "VALIDATION_FAILED",
					Message: fmt.Sprintf("workflow %d: name is required", i),
				},
			}, nil
		}
		if len(wf.Tasks) == 0 {
			return &pb.SubmitWorkflowsResponse{
				Error: &pb.Error{
					Code:    "VALIDATION_FAILED",
					Message: fmt.Sprintf("workflow %d: at least one task is required", i),
				},
			}, nil
		}
	}

	// Second pass: submit all workflows
	for i, wf := range req.Workflows {
		tasks := make([]WorkflowTask, len(wf.Tasks))
		for j, t := range wf.Tasks {
			tasks[j] = WorkflowTask{
				ID:           t.Id,
				Name:         t.Name,
				Dependencies: t.Dependencies,
				Metadata:     t.Metadata,
			}
		}

		workflowID, err := s.engine.SubmitWorkflow(ctx, wf.Name, tasks)
		if err != nil {
			// Rollback: cancel all previously submitted workflows
			for _, id := range submittedIDs {
				s.engine.CancelWorkflow(context.Background(), id, true)
			}

			return &pb.SubmitWorkflowsResponse{
				Error: &pb.Error{
					Code:    "ATOMIC_SUBMISSION_FAILED",
					Message: fmt.Sprintf("workflow %d failed: %v (rolled back all submissions)", i, err),
				},
			}, nil
		}

		submittedIDs = append(submittedIDs, workflowID)
		results[i] = &pb.WorkflowSubmissionResult{
			Index:      int32(i),
			Success:    true,
			WorkflowId: workflowID,
		}
	}

	resp := &pb.SubmitWorkflowsResponse{
		Results: results,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(len(results)),
		},
	}

	// Cache response for idempotency
	if req.IdempotencyKey != "" {
		s.idempotencyCache.Set(req.IdempotencyKey, resp)
	}

	return resp, nil
}

// submitWorkflowsParallel submits workflows in parallel (best-effort)
func (s *BatchServiceServer) submitWorkflowsParallel(ctx context.Context, req *pb.SubmitWorkflowsRequest) (*pb.SubmitWorkflowsResponse, error) {
	results := make([]*pb.WorkflowSubmissionResult, len(req.Workflows))

	// Ordered mode: submit sequentially
	if req.Ordered {
		for i, wf := range req.Workflows {
			results[i] = s.submitSingleWorkflow(ctx, wf, i)
		}
	} else {
		// Parallel mode: use worker pool
		var wg sync.WaitGroup
		workChan := make(chan int, len(req.Workflows))
		resultChan := make(chan struct {
			index  int
			result *pb.WorkflowSubmissionResult
		}, len(req.Workflows))

		// Start workers
		for w := 0; w < s.workerPoolSize && w < len(req.Workflows); w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range workChan {
					result := s.submitSingleWorkflow(ctx, req.Workflows[i], i)
					resultChan <- struct {
						index  int
						result *pb.WorkflowSubmissionResult
					}{i, result}
				}
			}()
		}

		// Send work
		for i := range req.Workflows {
			workChan <- i
		}
		close(workChan)

		// Wait for completion
		go func() {
			wg.Wait()
			close(resultChan)
		}()

		// Collect results
		for r := range resultChan {
			results[r.index] = r.result
		}
	}

	resp := &pb.SubmitWorkflowsResponse{
		Results: results,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(len(results)),
		},
	}

	// Cache response for idempotency
	if req.IdempotencyKey != "" {
		s.idempotencyCache.Set(req.IdempotencyKey, resp)
	}

	return resp, nil
}

// submitSingleWorkflow submits a single workflow and returns the result
func (s *BatchServiceServer) submitSingleWorkflow(ctx context.Context, wf *pb.SubmitWorkflowRequest, index int) *pb.WorkflowSubmissionResult {
	if wf.Name == "" {
		return &pb.WorkflowSubmissionResult{
			Index:   int32(index),
			Success: false,
			Error: &pb.Error{
				Code:    "VALIDATION_FAILED",
				Message: "workflow name is required",
			},
		}
	}

	if len(wf.Tasks) == 0 {
		return &pb.WorkflowSubmissionResult{
			Index:   int32(index),
			Success: false,
			Error: &pb.Error{
				Code:    "VALIDATION_FAILED",
				Message: "at least one task is required",
			},
		}
	}

	tasks := make([]WorkflowTask, len(wf.Tasks))
	for j, t := range wf.Tasks {
		tasks[j] = WorkflowTask{
			ID:           t.Id,
			Name:         t.Name,
			Dependencies: t.Dependencies,
			Metadata:     t.Metadata,
		}
	}

	workflowID, err := s.engine.SubmitWorkflow(ctx, wf.Name, tasks)
	if err != nil {
		return &pb.WorkflowSubmissionResult{
			Index:   int32(index),
			Success: false,
			Error: &pb.Error{
				Code:    "SUBMISSION_FAILED",
				Message: err.Error(),
			},
		}
	}

	return &pb.WorkflowSubmissionResult{
		Index:      int32(index),
		Success:    true,
		WorkflowId: workflowID,
	}
}

// GetWorkflowStatuses handles batch workflow status retrieval
func (s *BatchServiceServer) GetWorkflowStatuses(ctx context.Context, req *pb.GetWorkflowStatusesRequest) (*pb.GetWorkflowStatusesResponse, error) {
	if req == nil || len(req.WorkflowIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one workflow ID is required")
	}

	// Validate batch size
	if len(req.WorkflowIds) > MaxBatchSize {
		return nil, status.Errorf(codes.InvalidArgument, "batch size exceeds maximum of %d", MaxBatchSize)
	}

	// Apply pagination
	startIdx := 0
	endIdx := len(req.WorkflowIds)
	if req.Pagination != nil && req.Pagination.PageSize > 0 {
		pageSize := int(req.Pagination.PageSize)
		if pageSize > MaxBatchSize {
			pageSize = MaxBatchSize
		}
		// Simple offset-based pagination for batch operations
		if req.Pagination.PageToken != "" {
			// Parse page token as offset
			var offset int
			fmt.Sscanf(req.Pagination.PageToken, "%d", &offset)
			startIdx = offset
		}
		endIdx = startIdx + pageSize
		if endIdx > len(req.WorkflowIds) {
			endIdx = len(req.WorkflowIds)
		}
	}

	workflowIDs := req.WorkflowIds[startIdx:endIdx]
	results := make([]*pb.WorkflowStatusResult, len(workflowIDs))

	// Parallel processing with worker pool
	var wg sync.WaitGroup
	workChan := make(chan int, len(workflowIDs))
	resultChan := make(chan struct {
		index  int
		result *pb.WorkflowStatusResult
	}, len(workflowIDs))

	// Start workers
	for w := 0; w < s.workerPoolSize && w < len(workflowIDs); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range workChan {
				result := s.getSingleWorkflowStatus(ctx, workflowIDs[i])
				resultChan <- struct {
					index  int
					result *pb.WorkflowStatusResult
				}{i, result}
			}
		}()
	}

	// Send work
	for i := range workflowIDs {
		workChan <- i
	}
	close(workChan)

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for r := range resultChan {
		results[r.index] = r.result
	}

	// Generate next page token
	nextPageToken := ""
	if endIdx < len(req.WorkflowIds) {
		nextPageToken = fmt.Sprintf("%d", endIdx)
	}

	return &pb.GetWorkflowStatusesResponse{
		Results: results,
		Pagination: &pb.PaginationResponse{
			NextPageToken: nextPageToken,
			TotalCount:    int32(len(results)),
		},
	}, nil
}

// getSingleWorkflowStatus retrieves status for a single workflow
func (s *BatchServiceServer) getSingleWorkflowStatus(ctx context.Context, workflowID string) *pb.WorkflowStatusResult {
	ws, err := s.engine.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		return &pb.WorkflowStatusResult{
			WorkflowId: workflowID,
			Found:      false,
			Error: &pb.Error{
				Code:    "NOT_FOUND",
				Message: err.Error(),
			},
		}
	}

	// Convert tasks
	pbTasks := make([]*pb.TaskStatusDetail, len(ws.Tasks))
	for i, t := range ws.Tasks {
		pbTasks[i] = &pb.TaskStatusDetail{
			TaskId:       t.TaskID,
			Name:         t.Name,
			Status:       convertToProtoTaskStatus(t.Status),
			StartedAt:    timestampFromUnix(t.StartedAt),
			CompletedAt:  timestampFromUnix(t.CompletedAt),
			ErrorMessage: t.ErrorMsg,
		}
	}

	return &pb.WorkflowStatusResult{
		WorkflowId: workflowID,
		Found:      true,
		Status: &pb.GetWorkflowStatusResponse{
			WorkflowId: ws.WorkflowID,
			Name:       ws.Name,
			Status:     convertToProtoStatus(ws.Status),
			Tasks:      pbTasks,
			CreatedAt:  timestampFromUnix(ws.CreatedAt),
			UpdatedAt:  timestampFromUnix(ws.UpdatedAt),
		},
	}
}

// CancelWorkflows handles batch workflow cancellation
func (s *BatchServiceServer) CancelWorkflows(ctx context.Context, req *pb.CancelWorkflowsRequest) (*pb.CancelWorkflowsResponse, error) {
	if req == nil || len(req.WorkflowIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one workflow ID is required")
	}

	// Validate batch size
	if len(req.WorkflowIds) > MaxBatchSize {
		return nil, status.Errorf(codes.InvalidArgument, "batch size exceeds maximum of %d", MaxBatchSize)
	}

	results := make([]*pb.WorkflowCancellationResult, len(req.WorkflowIds))

	// Apply timeout if specified
	cancelCtx := ctx
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		cancelCtx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	// Parallel processing with worker pool
	var wg sync.WaitGroup
	workChan := make(chan int, len(req.WorkflowIds))
	resultChan := make(chan struct {
		index  int
		result *pb.WorkflowCancellationResult
	}, len(req.WorkflowIds))

	// Start workers
	for w := 0; w < s.workerPoolSize && w < len(req.WorkflowIds); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range workChan {
				result := s.cancelSingleWorkflow(cancelCtx, req.WorkflowIds[i], req.Force)
				resultChan <- struct {
					index  int
					result *pb.WorkflowCancellationResult
				}{i, result}
			}
		}()
	}

	// Send work
	for i := range req.WorkflowIds {
		workChan <- i
	}
	close(workChan)

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for r := range resultChan {
		results[r.index] = r.result
	}

	return &pb.CancelWorkflowsResponse{
		Results: results,
	}, nil
}

// cancelSingleWorkflow cancels a single workflow
func (s *BatchServiceServer) cancelSingleWorkflow(ctx context.Context, workflowID string, force bool) *pb.WorkflowCancellationResult {
	err := s.engine.CancelWorkflow(ctx, workflowID, force)
	if err != nil {
		// Check if workflow is already in terminal state
		ws, statusErr := s.engine.GetWorkflowStatus(ctx, workflowID)
		if statusErr == nil && isTerminalStatus(ws.Status) {
			return &pb.WorkflowCancellationResult{
				WorkflowId:       workflowID,
				Success:          true,
				AlreadyTerminal:  true,
			}
		}

		return &pb.WorkflowCancellationResult{
			WorkflowId: workflowID,
			Success:    false,
			Error: &pb.Error{
				Code:    "CANCEL_FAILED",
				Message: err.Error(),
			},
		}
	}

	return &pb.WorkflowCancellationResult{
		WorkflowId: workflowID,
		Success:    true,
	}
}

// GetTaskResults handles batch task result retrieval
func (s *BatchServiceServer) GetTaskResults(ctx context.Context, req *pb.GetTaskResultsRequest) (*pb.GetTaskResultsResponse, error) {
	if req == nil || req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id is required")
	}

	if len(req.TaskIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one task ID is required")
	}

	// Validate batch size
	if len(req.TaskIds) > MaxBatchSize {
		return nil, status.Errorf(codes.InvalidArgument, "batch size exceeds maximum of %d", MaxBatchSize)
	}

	// Apply pagination
	startIdx := 0
	endIdx := len(req.TaskIds)
	if req.Pagination != nil && req.Pagination.PageSize > 0 {
		pageSize := int(req.Pagination.PageSize)
		if pageSize > MaxBatchSize {
			pageSize = MaxBatchSize
		}
		if req.Pagination.PageToken != "" {
			var offset int
			fmt.Sscanf(req.Pagination.PageToken, "%d", &offset)
			startIdx = offset
		}
		endIdx = startIdx + pageSize
		if endIdx > len(req.TaskIds) {
			endIdx = len(req.TaskIds)
		}
	}

	taskIDs := req.TaskIds[startIdx:endIdx]
	results := make([]*pb.TaskResultDetail, len(taskIDs))

	// Parallel processing with worker pool
	var wg sync.WaitGroup
	workChan := make(chan int, len(taskIDs))
	resultChan := make(chan struct {
		index  int
		result *pb.TaskResultDetail
	}, len(taskIDs))

	// Start workers
	for w := 0; w < s.workerPoolSize && w < len(taskIDs); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range workChan {
				result := s.getSingleTaskResult(ctx, req.WorkflowId, taskIDs[i])
				resultChan <- struct {
					index  int
					result *pb.TaskResultDetail
				}{i, result}
			}
		}()
	}

	// Send work
	for i := range taskIDs {
		workChan <- i
	}
	close(workChan)

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for r := range resultChan {
		results[r.index] = r.result
	}

	// Generate next page token
	nextPageToken := ""
	if endIdx < len(req.TaskIds) {
		nextPageToken = fmt.Sprintf("%d", endIdx)
	}

	return &pb.GetTaskResultsResponse{
		Results: results,
		Pagination: &pb.PaginationResponse{
			NextPageToken: nextPageToken,
			TotalCount:    int32(len(results)),
		},
	}, nil
}

// getSingleTaskResult retrieves result for a single task
func (s *BatchServiceServer) getSingleTaskResult(ctx context.Context, workflowID, taskID string) *pb.TaskResultDetail {
	result, err := s.engine.GetTaskResult(ctx, workflowID, taskID)
	if err != nil {
		return &pb.TaskResultDetail{
			TaskId: taskID,
			Found:  false,
			Error: &pb.Error{
				Code:    "NOT_FOUND",
				Message: err.Error(),
			},
		}
	}

	return &pb.TaskResultDetail{
		TaskId: taskID,
		Found:  true,
		Result: &pb.GetTaskResultResponse{
			TaskId:       result.TaskID,
			Status:       convertToProtoTaskStatus(result.Status),
			ResultData:   result.ResultData,
			ErrorMessage: result.ErrorMsg,
		},
	}
}

// isTerminalStatus checks if a workflow status is terminal
func isTerminalStatus(status string) bool {
	return status == "COMPLETED" || status == "FAILED" || status == "CANCELLED"
}

// IdempotencyCache provides simple in-memory caching for idempotency
type IdempotencyCache struct {
	mu    sync.RWMutex
	cache map[string]*cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// NewIdempotencyCache creates a new idempotency cache
func NewIdempotencyCache(ttl time.Duration) *IdempotencyCache {
	cache := &IdempotencyCache{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}
	// Start cleanup goroutine
	go cache.cleanup()
	return cache
}

// Get retrieves a value from the cache
func (c *IdempotencyCache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.value
}

// Set stores a value in the cache
func (c *IdempotencyCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// cleanup removes expired entries periodically
func (c *IdempotencyCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.cache {
			if now.After(entry.expiresAt) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}
