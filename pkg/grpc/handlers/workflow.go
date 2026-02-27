package handlers

import (
	"context"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WorkflowEngine defines the interface for workflow operations
type WorkflowEngine interface {
	SubmitWorkflow(ctx context.Context, name string, tasks []WorkflowTask) (string, error)
	GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error)
	ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*WorkflowSummary, string, error)
	CancelWorkflow(ctx context.Context, workflowID string, force bool) error
	GetTaskResult(ctx context.Context, workflowID, taskID string) (*TaskResult, error)
}

// WorkflowTask represents a task definition
type WorkflowTask struct {
	ID           string
	Name         string
	Dependencies []string
	Metadata     map[string]string
}

// WorkflowStatus represents workflow execution status
type WorkflowStatus struct {
	WorkflowID string
	Name       string
	Status     string
	Tasks      []*TaskStatus
	CreatedAt  int64
	UpdatedAt  int64
}

// TaskStatus represents task execution status
type TaskStatus struct {
	TaskID      string
	Name        string
	Status      string
	StartedAt   int64
	CompletedAt int64
	ErrorMsg    string
}

// WorkflowSummary represents a workflow summary
type WorkflowSummary struct {
	WorkflowID string
	Name       string
	Status     string
	CreatedAt  int64
	UpdatedAt  int64
}

// WorkflowFilter represents workflow list filter
type WorkflowFilter struct {
	StatusFilter string
	PageSize     int32
	PageToken    string
}

// TaskResult represents task execution result
type TaskResult struct {
	TaskID     string
	Status     string
	ResultData []byte
	ErrorMsg   string
}

// WorkflowServiceServer implements the gRPC WorkflowService
type WorkflowServiceServer struct {
	pb.UnimplementedWorkflowServiceServer
	engine WorkflowEngine
}

// NewWorkflowServiceServer creates a new workflow service server
func NewWorkflowServiceServer(engine WorkflowEngine) *WorkflowServiceServer {
	return &WorkflowServiceServer{
		engine: engine,
	}
}

// SubmitWorkflow handles workflow submission
func (s *WorkflowServiceServer) SubmitWorkflow(ctx context.Context, req *pb.SubmitWorkflowRequest) (*pb.SubmitWorkflowResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow name is required")
	}

	if len(req.Tasks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one task is required")
	}

	// Convert proto tasks to engine tasks
	tasks := make([]WorkflowTask, len(req.Tasks))
	for i, t := range req.Tasks {
		if t.Id == "" {
			return nil, status.Errorf(codes.InvalidArgument, "task %d: id is required", i)
		}
		tasks[i] = WorkflowTask{
			ID:           t.Id,
			Name:         t.Name,
			Dependencies: t.Dependencies,
			Metadata:     t.Metadata,
		}
	}

	// Submit workflow to engine
	workflowID, err := s.engine.SubmitWorkflow(ctx, req.Name, tasks)
	if err != nil {
		return &pb.SubmitWorkflowResponse{
			Error: &pb.Error{
				Code:    "SUBMISSION_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &pb.SubmitWorkflowResponse{
		WorkflowId: workflowID,
	}, nil
}

// ListWorkflows handles workflow listing with pagination
func (s *WorkflowServiceServer) ListWorkflows(ctx context.Context, req *pb.ListWorkflowsRequest) (*pb.ListWorkflowsResponse, error) {
	if req == nil {
		req = &pb.ListWorkflowsRequest{}
	}

	// Set default page size
	pageSize := int32(50)
	if req.Pagination != nil && req.Pagination.PageSize > 0 {
		pageSize = req.Pagination.PageSize
		if pageSize > 1000 {
			pageSize = 1000 // Max page size
		}
	}

	pageToken := ""
	if req.Pagination != nil {
		pageToken = req.Pagination.PageToken
	}

	// Convert status filter
	statusFilter := ""
	if req.StatusFilter != pb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED {
		statusFilter = normalizeWorkflowFilterStatus(req.StatusFilter.String())
	}

	filter := WorkflowFilter{
		StatusFilter: statusFilter,
		PageSize:     pageSize,
		PageToken:    pageToken,
	}

	// Get workflows from engine
	workflows, nextToken, err := s.engine.ListWorkflows(ctx, filter)
	if err != nil {
		return &pb.ListWorkflowsResponse{
			Error: &pb.Error{
				Code:    "LIST_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	// Convert to proto format
	pbWorkflows := make([]*pb.WorkflowSummary, len(workflows))
	for i, w := range workflows {
		pbWorkflows[i] = &pb.WorkflowSummary{
			WorkflowId: w.WorkflowID,
			Name:       w.Name,
			Status:     convertToProtoStatus(w.Status),
			CreatedAt:  timestampFromUnix(w.CreatedAt),
			UpdatedAt:  timestampFromUnix(w.UpdatedAt),
		}
	}

	return &pb.ListWorkflowsResponse{
		Workflows: pbWorkflows,
		Pagination: &pb.PaginationResponse{
			NextPageToken: nextToken,
			TotalCount:    int32(len(pbWorkflows)),
		},
	}, nil
}

// GetWorkflowStatus handles workflow status retrieval
func (s *WorkflowServiceServer) GetWorkflowStatus(ctx context.Context, req *pb.GetWorkflowStatusRequest) (*pb.GetWorkflowStatusResponse, error) {
	if req == nil || req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id is required")
	}

	// Get status from engine
	ws, err := s.engine.GetWorkflowStatus(ctx, req.WorkflowId)
	if err != nil {
		if IsNotFoundError(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
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

	return &pb.GetWorkflowStatusResponse{
		WorkflowId: ws.WorkflowID,
		Name:       ws.Name,
		Status:     convertToProtoStatus(ws.Status),
		Tasks:      pbTasks,
		CreatedAt:  timestampFromUnix(ws.CreatedAt),
		UpdatedAt:  timestampFromUnix(ws.UpdatedAt),
	}, nil
}

// CancelWorkflow handles workflow cancellation
func (s *WorkflowServiceServer) CancelWorkflow(ctx context.Context, req *pb.CancelWorkflowRequest) (*pb.CancelWorkflowResponse, error) {
	if req == nil || req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id is required")
	}

	// Cancel workflow in engine
	err := s.engine.CancelWorkflow(ctx, req.WorkflowId, req.Force)
	if err != nil {
		if IsNotFoundError(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	return &pb.CancelWorkflowResponse{
		Success: true,
	}, nil
}

// GetTaskResult handles task result retrieval
func (s *WorkflowServiceServer) GetTaskResult(ctx context.Context, req *pb.GetTaskResultRequest) (*pb.GetTaskResultResponse, error) {
	if req == nil || req.WorkflowId == "" || req.TaskId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id and task_id are required")
	}

	// Get task result from engine
	result, err := s.engine.GetTaskResult(ctx, req.WorkflowId, req.TaskId)
	if err != nil {
		if IsNotFoundError(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetTaskResultResponse{
		TaskId:       result.TaskID,
		Status:       convertToProtoTaskStatus(result.Status),
		ResultData:   result.ResultData,
		ErrorMessage: result.ErrorMsg,
	}, nil
}
