package handlers

import (
	"context"
	"errors"
	"testing"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockWorkflowEngine is a mock implementation of WorkflowEngine for testing
type MockWorkflowEngine struct {
	SubmitWorkflowFunc    func(ctx context.Context, name string, tasks []WorkflowTask) (string, error)
	GetWorkflowStatusFunc func(ctx context.Context, workflowID string) (*WorkflowStatus, error)
	ListWorkflowsFunc     func(ctx context.Context, filter WorkflowFilter) ([]*WorkflowSummary, string, error)
	CancelWorkflowFunc    func(ctx context.Context, workflowID string, force bool) error
	GetTaskResultFunc     func(ctx context.Context, workflowID, taskID string) (*TaskResult, error)
}

func (m *MockWorkflowEngine) SubmitWorkflow(ctx context.Context, name string, tasks []WorkflowTask) (string, error) {
	if m.SubmitWorkflowFunc != nil {
		return m.SubmitWorkflowFunc(ctx, name, tasks)
	}
	return "workflow-123", nil
}

func (m *MockWorkflowEngine) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	if m.GetWorkflowStatusFunc != nil {
		return m.GetWorkflowStatusFunc(ctx, workflowID)
	}
	return &WorkflowStatus{
		WorkflowID: workflowID,
		Name:       "test-workflow",
		Status:     "RUNNING",
		Tasks:      []*TaskStatus{},
		CreatedAt:  1234567890,
		UpdatedAt:  1234567890,
	}, nil
}

func (m *MockWorkflowEngine) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*WorkflowSummary, string, error) {
	if m.ListWorkflowsFunc != nil {
		return m.ListWorkflowsFunc(ctx, filter)
	}
	return []*WorkflowSummary{
		{
			WorkflowID: "workflow-1",
			Name:       "test-1",
			Status:     "RUNNING",
			CreatedAt:  1234567890,
			UpdatedAt:  1234567890,
		},
	}, "", nil
}

func (m *MockWorkflowEngine) CancelWorkflow(ctx context.Context, workflowID string, force bool) error {
	if m.CancelWorkflowFunc != nil {
		return m.CancelWorkflowFunc(ctx, workflowID, force)
	}
	return nil
}

func (m *MockWorkflowEngine) GetTaskResult(ctx context.Context, workflowID, taskID string) (*TaskResult, error) {
	if m.GetTaskResultFunc != nil {
		return m.GetTaskResultFunc(ctx, workflowID, taskID)
	}
	return &TaskResult{
		TaskID:     taskID,
		Status:     "COMPLETED",
		ResultData: []byte("result"),
		ErrorMsg:   "",
	}, nil
}

func TestSubmitWorkflow_Success(t *testing.T) {
	engine := &MockWorkflowEngine{}
	server := NewWorkflowServiceServer(engine)

	req := &pb.SubmitWorkflowRequest{
		Name: "test-workflow",
		Tasks: []*pb.TaskDefinition{
			{Id: "task-1", Name: "Task 1"},
			{Id: "task-2", Name: "Task 2", Dependencies: []string{"task-1"}},
		},
	}

	resp, err := server.SubmitWorkflow(context.Background(), req)
	if err != nil {
		t.Fatalf("SubmitWorkflow failed: %v", err)
	}

	if resp.WorkflowId == "" {
		t.Error("Expected workflow ID, got empty string")
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got: %v", resp.Error)
	}
}

func TestSubmitWorkflow_MissingName(t *testing.T) {
	engine := &MockWorkflowEngine{}
	server := NewWorkflowServiceServer(engine)

	req := &pb.SubmitWorkflowRequest{
		Tasks: []*pb.TaskDefinition{
			{Id: "task-1", Name: "Task 1"},
		},
	}

	_, err := server.SubmitWorkflow(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for missing name")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", st.Code())
	}
}

func TestSubmitWorkflow_NoTasks(t *testing.T) {
	engine := &MockWorkflowEngine{}
	server := NewWorkflowServiceServer(engine)

	req := &pb.SubmitWorkflowRequest{
		Name:  "test-workflow",
		Tasks: []*pb.TaskDefinition{},
	}

	_, err := server.SubmitWorkflow(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for no tasks")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", st.Code())
	}
}

func TestSubmitWorkflow_EngineError(t *testing.T) {
	engine := &MockWorkflowEngine{
		SubmitWorkflowFunc: func(ctx context.Context, name string, tasks []WorkflowTask) (string, error) {
			return "", errors.New("engine error")
		},
	}
	server := NewWorkflowServiceServer(engine)

	req := &pb.SubmitWorkflowRequest{
		Name: "test-workflow",
		Tasks: []*pb.TaskDefinition{
			{Id: "task-1", Name: "Task 1"},
		},
	}

	resp, err := server.SubmitWorkflow(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no gRPC error, got: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	if resp.Error.Code != "SUBMISSION_FAILED" {
		t.Errorf("Expected SUBMISSION_FAILED, got %s", resp.Error.Code)
	}
}

func TestListWorkflows_Success(t *testing.T) {
	engine := &MockWorkflowEngine{}
	server := NewWorkflowServiceServer(engine)

	req := &pb.ListWorkflowsRequest{
		Pagination: &pb.PaginationRequest{
			PageSize: 10,
		},
	}

	resp, err := server.ListWorkflows(context.Background(), req)
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}

	if len(resp.Workflows) == 0 {
		t.Error("Expected workflows, got empty list")
	}

	if resp.Pagination == nil {
		t.Error("Expected pagination response")
	}
}

func TestListWorkflows_DefaultPageSize(t *testing.T) {
	engine := &MockWorkflowEngine{
		ListWorkflowsFunc: func(ctx context.Context, filter WorkflowFilter) ([]*WorkflowSummary, string, error) {
			if filter.PageSize != 50 {
				t.Errorf("Expected default page size 50, got %d", filter.PageSize)
			}
			return []*WorkflowSummary{}, "", nil
		},
	}
	server := NewWorkflowServiceServer(engine)

	_, err := server.ListWorkflows(context.Background(), &pb.ListWorkflowsRequest{})
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}
}

func TestGetWorkflowStatus_Success(t *testing.T) {
	engine := &MockWorkflowEngine{}
	server := NewWorkflowServiceServer(engine)

	req := &pb.GetWorkflowStatusRequest{
		WorkflowId: "workflow-123",
	}

	resp, err := server.GetWorkflowStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("GetWorkflowStatus failed: %v", err)
	}

	if resp.WorkflowId != "workflow-123" {
		t.Errorf("Expected workflow-123, got %s", resp.WorkflowId)
	}

	if resp.Status == pb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED {
		t.Error("Expected valid status")
	}
}

func TestGetWorkflowStatus_MissingID(t *testing.T) {
	engine := &MockWorkflowEngine{}
	server := NewWorkflowServiceServer(engine)

	req := &pb.GetWorkflowStatusRequest{}

	_, err := server.GetWorkflowStatus(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for missing workflow ID")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", st.Code())
	}
}

func TestCancelWorkflow_Success(t *testing.T) {
	engine := &MockWorkflowEngine{}
	server := NewWorkflowServiceServer(engine)

	req := &pb.CancelWorkflowRequest{
		WorkflowId: "workflow-123",
		Force:      false,
	}

	resp, err := server.CancelWorkflow(context.Background(), req)
	if err != nil {
		t.Fatalf("CancelWorkflow failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success")
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got: %v", resp.Error)
	}
}

func TestCancelWorkflow_EngineError(t *testing.T) {
	engine := &MockWorkflowEngine{
		CancelWorkflowFunc: func(ctx context.Context, workflowID string, force bool) error {
			return errors.New("cancel failed")
		},
	}
	server := NewWorkflowServiceServer(engine)

	req := &pb.CancelWorkflowRequest{
		WorkflowId: "workflow-123",
	}

	resp, err := server.CancelWorkflow(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no gRPC error, got: %v", err)
	}

	if resp.Success {
		t.Error("Expected failure")
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}
}

func TestGetTaskResult_Success(t *testing.T) {
	engine := &MockWorkflowEngine{}
	server := NewWorkflowServiceServer(engine)

	req := &pb.GetTaskResultRequest{
		WorkflowId: "workflow-123",
		TaskId:     "task-1",
	}

	resp, err := server.GetTaskResult(context.Background(), req)
	if err != nil {
		t.Fatalf("GetTaskResult failed: %v", err)
	}

	if resp.TaskId != "task-1" {
		t.Errorf("Expected task-1, got %s", resp.TaskId)
	}

	if resp.Status == pb.TaskStatus_TASK_STATUS_UNSPECIFIED {
		t.Error("Expected valid status")
	}
}

func TestGetTaskResult_MissingIDs(t *testing.T) {
	engine := &MockWorkflowEngine{}
	server := NewWorkflowServiceServer(engine)

	req := &pb.GetTaskResultRequest{
		WorkflowId: "workflow-123",
	}

	_, err := server.GetTaskResult(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for missing task ID")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got %v", st.Code())
	}
}
