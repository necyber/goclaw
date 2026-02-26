package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockBatchEngine implements WorkflowEngine for batch testing
type mockBatchEngine struct {
	submitFunc        func(ctx context.Context, name string, tasks []WorkflowTask) (string, error)
	getStatusFunc     func(ctx context.Context, workflowID string) (*WorkflowStatus, error)
	cancelFunc        func(ctx context.Context, workflowID string, force bool) error
	getTaskResultFunc func(ctx context.Context, workflowID, taskID string) (*TaskResult, error)
}

func (m *mockBatchEngine) SubmitWorkflow(ctx context.Context, name string, tasks []WorkflowTask) (string, error) {
	if m.submitFunc != nil {
		return m.submitFunc(ctx, name, tasks)
	}
	return "wf-" + name, nil
}

func (m *mockBatchEngine) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	if m.getStatusFunc != nil {
		return m.getStatusFunc(ctx, workflowID)
	}
	return &WorkflowStatus{
		WorkflowID: workflowID,
		Name:       "test-workflow",
		Status:     "RUNNING",
		Tasks:      []*TaskStatus{},
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}, nil
}

func (m *mockBatchEngine) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*WorkflowSummary, string, error) {
	return nil, "", nil
}

func (m *mockBatchEngine) CancelWorkflow(ctx context.Context, workflowID string, force bool) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, workflowID, force)
	}
	return nil
}

func (m *mockBatchEngine) GetTaskResult(ctx context.Context, workflowID, taskID string) (*TaskResult, error) {
	if m.getTaskResultFunc != nil {
		return m.getTaskResultFunc(ctx, workflowID, taskID)
	}
	return &TaskResult{
		TaskID:     taskID,
		Status:     "COMPLETED",
		ResultData: []byte("result"),
		ErrorMsg:   "",
	}, nil
}

func TestSubmitWorkflows_Success(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	req := &pb.SubmitWorkflowsRequest{
		Workflows: []*pb.SubmitWorkflowRequest{
			{
				Name: "workflow-1",
				Tasks: []*pb.TaskDefinition{
					{Id: "task-1", Name: "Task 1"},
				},
			},
			{
				Name: "workflow-2",
				Tasks: []*pb.TaskDefinition{
					{Id: "task-2", Name: "Task 2"},
				},
			},
		},
	}

	resp, err := server.SubmitWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 2)
	assert.True(t, resp.Results[0].Success)
	assert.True(t, resp.Results[1].Success)
	assert.Equal(t, "wf-workflow-1", resp.Results[0].WorkflowId)
	assert.Equal(t, "wf-workflow-2", resp.Results[1].WorkflowId)
}

func TestSubmitWorkflows_EmptyRequest(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	_, err := server.SubmitWorkflows(context.Background(), &pb.SubmitWorkflowsRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestSubmitWorkflows_ExceedsMaxBatchSize(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	workflows := make([]*pb.SubmitWorkflowRequest, MaxBatchSize+1)
	for i := range workflows {
		workflows[i] = &pb.SubmitWorkflowRequest{
			Name: "workflow",
			Tasks: []*pb.TaskDefinition{
				{Id: "task-1", Name: "Task 1"},
			},
		}
	}

	req := &pb.SubmitWorkflowsRequest{Workflows: workflows}
	_, err := server.SubmitWorkflows(context.Background(), req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestSubmitWorkflows_Atomic_Success(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	req := &pb.SubmitWorkflowsRequest{
		Workflows: []*pb.SubmitWorkflowRequest{
			{
				Name: "workflow-1",
				Tasks: []*pb.TaskDefinition{
					{Id: "task-1", Name: "Task 1"},
				},
			},
			{
				Name: "workflow-2",
				Tasks: []*pb.TaskDefinition{
					{Id: "task-2", Name: "Task 2"},
				},
			},
		},
		Atomic: true,
	}

	resp, err := server.SubmitWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 2)
	assert.True(t, resp.Results[0].Success)
	assert.True(t, resp.Results[1].Success)
}

func TestSubmitWorkflows_Atomic_Rollback(t *testing.T) {
	callCount := 0
	cancelledIDs := []string{}

	engine := &mockBatchEngine{
		submitFunc: func(ctx context.Context, name string, tasks []WorkflowTask) (string, error) {
			callCount++
			if callCount == 2 {
				return "", errors.New("submission failed")
			}
			return "wf-" + name, nil
		},
		cancelFunc: func(ctx context.Context, workflowID string, force bool) error {
			cancelledIDs = append(cancelledIDs, workflowID)
			return nil
		},
	}
	server := NewBatchServiceServer(engine)

	req := &pb.SubmitWorkflowsRequest{
		Workflows: []*pb.SubmitWorkflowRequest{
			{
				Name: "workflow-1",
				Tasks: []*pb.TaskDefinition{
					{Id: "task-1", Name: "Task 1"},
				},
			},
			{
				Name: "workflow-2",
				Tasks: []*pb.TaskDefinition{
					{Id: "task-2", Name: "Task 2"},
				},
			},
		},
		Atomic: true,
	}

	resp, err := server.SubmitWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "rolled back")
	assert.Len(t, cancelledIDs, 1)
	assert.Equal(t, "wf-workflow-1", cancelledIDs[0])
}

func TestSubmitWorkflows_Atomic_ValidationFailed(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	req := &pb.SubmitWorkflowsRequest{
		Workflows: []*pb.SubmitWorkflowRequest{
			{
				Name: "workflow-1",
				Tasks: []*pb.TaskDefinition{
					{Id: "task-1", Name: "Task 1"},
				},
			},
			{
				Name: "", // Invalid: empty name
				Tasks: []*pb.TaskDefinition{
					{Id: "task-2", Name: "Task 2"},
				},
			},
		},
		Atomic: true,
	}

	resp, err := server.SubmitWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "name is required")
}

func TestSubmitWorkflows_Idempotency(t *testing.T) {
	callCount := 0
	engine := &mockBatchEngine{
		submitFunc: func(ctx context.Context, name string, tasks []WorkflowTask) (string, error) {
			callCount++
			return "wf-" + name, nil
		},
	}
	server := NewBatchServiceServer(engine)

	req := &pb.SubmitWorkflowsRequest{
		Workflows: []*pb.SubmitWorkflowRequest{
			{
				Name: "workflow-1",
				Tasks: []*pb.TaskDefinition{
					{Id: "task-1", Name: "Task 1"},
				},
			},
		},
		IdempotencyKey: "test-key-123",
	}

	// First call
	resp1, err := server.SubmitWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	assert.Equal(t, 1, callCount)

	// Second call with same idempotency key
	resp2, err := server.SubmitWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	assert.Equal(t, 1, callCount) // Should not increment
	assert.Equal(t, resp1.Results[0].WorkflowId, resp2.Results[0].WorkflowId)
}

func TestSubmitWorkflows_Ordered(t *testing.T) {
	submittedNames := []string{}
	engine := &mockBatchEngine{
		submitFunc: func(ctx context.Context, name string, tasks []WorkflowTask) (string, error) {
			submittedNames = append(submittedNames, name)
			time.Sleep(10 * time.Millisecond) // Simulate work
			return "wf-" + name, nil
		},
	}
	server := NewBatchServiceServer(engine)

	req := &pb.SubmitWorkflowsRequest{
		Workflows: []*pb.SubmitWorkflowRequest{
			{Name: "workflow-1", Tasks: []*pb.TaskDefinition{{Id: "t1", Name: "T1"}}},
			{Name: "workflow-2", Tasks: []*pb.TaskDefinition{{Id: "t2", Name: "T2"}}},
			{Name: "workflow-3", Tasks: []*pb.TaskDefinition{{Id: "t3", Name: "T3"}}},
		},
		Ordered: true,
	}

	resp, err := server.SubmitWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, []string{"workflow-1", "workflow-2", "workflow-3"}, submittedNames)
}

func TestGetWorkflowStatuses_Success(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	req := &pb.GetWorkflowStatusesRequest{
		WorkflowIds: []string{"wf-1", "wf-2", "wf-3"},
	}

	resp, err := server.GetWorkflowStatuses(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 3)
	assert.True(t, resp.Results[0].Found)
	assert.Equal(t, "wf-1", resp.Results[0].WorkflowId)
}

func TestGetWorkflowStatuses_EmptyRequest(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	_, err := server.GetWorkflowStatuses(context.Background(), &pb.GetWorkflowStatusesRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetWorkflowStatuses_NotFound(t *testing.T) {
	engine := &mockBatchEngine{
		getStatusFunc: func(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
			return nil, errors.New("workflow not found")
		},
	}
	server := NewBatchServiceServer(engine)

	req := &pb.GetWorkflowStatusesRequest{
		WorkflowIds: []string{"wf-nonexistent"},
	}

	resp, err := server.GetWorkflowStatuses(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)
	assert.False(t, resp.Results[0].Found)
	assert.NotNil(t, resp.Results[0].Error)
}

func TestGetWorkflowStatuses_Pagination(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	workflowIDs := make([]string, 100)
	for i := range workflowIDs {
		workflowIDs[i] = "wf-" + string(rune(i))
	}

	req := &pb.GetWorkflowStatusesRequest{
		WorkflowIds: workflowIDs,
		Pagination: &pb.PaginationRequest{
			PageSize: 10,
		},
	}

	resp, err := server.GetWorkflowStatuses(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 10)
	assert.NotEmpty(t, resp.Pagination.NextPageToken)

	// Get next page
	req.Pagination.PageToken = resp.Pagination.NextPageToken
	resp2, err := server.GetWorkflowStatuses(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	assert.Len(t, resp2.Results, 10)
}

func TestCancelWorkflows_Success(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	req := &pb.CancelWorkflowsRequest{
		WorkflowIds: []string{"wf-1", "wf-2"},
		Force:       true,
	}

	resp, err := server.CancelWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 2)
	assert.True(t, resp.Results[0].Success)
	assert.True(t, resp.Results[1].Success)
}

func TestCancelWorkflows_EmptyRequest(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	_, err := server.CancelWorkflows(context.Background(), &pb.CancelWorkflowsRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestCancelWorkflows_AlreadyTerminal(t *testing.T) {
	engine := &mockBatchEngine{
		cancelFunc: func(ctx context.Context, workflowID string, force bool) error {
			return errors.New("workflow already completed")
		},
		getStatusFunc: func(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
			return &WorkflowStatus{
				WorkflowID: workflowID,
				Status:     "COMPLETED",
			}, nil
		},
	}
	server := NewBatchServiceServer(engine)

	req := &pb.CancelWorkflowsRequest{
		WorkflowIds: []string{"wf-completed"},
	}

	resp, err := server.CancelWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)
	assert.True(t, resp.Results[0].Success)
	assert.True(t, resp.Results[0].AlreadyTerminal)
}

func TestCancelWorkflows_WithTimeout(t *testing.T) {
	engine := &mockBatchEngine{
		cancelFunc: func(ctx context.Context, workflowID string, force bool) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}
	server := NewBatchServiceServer(engine)

	req := &pb.CancelWorkflowsRequest{
		WorkflowIds:    []string{"wf-1"},
		TimeoutSeconds: 1,
	}

	resp, err := server.CancelWorkflows(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)
}

func TestGetTaskResults_Success(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	req := &pb.GetTaskResultsRequest{
		WorkflowId: "wf-1",
		TaskIds:    []string{"task-1", "task-2", "task-3"},
	}

	resp, err := server.GetTaskResults(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 3)
	assert.True(t, resp.Results[0].Found)
	assert.Equal(t, "task-1", resp.Results[0].TaskId)
}

func TestGetTaskResults_EmptyWorkflowID(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	req := &pb.GetTaskResultsRequest{
		TaskIds: []string{"task-1"},
	}

	_, err := server.GetTaskResults(context.Background(), req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetTaskResults_EmptyTaskIDs(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	req := &pb.GetTaskResultsRequest{
		WorkflowId: "wf-1",
	}

	_, err := server.GetTaskResults(context.Background(), req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetTaskResults_NotFound(t *testing.T) {
	engine := &mockBatchEngine{
		getTaskResultFunc: func(ctx context.Context, workflowID, taskID string) (*TaskResult, error) {
			return nil, errors.New("task not found")
		},
	}
	server := NewBatchServiceServer(engine)

	req := &pb.GetTaskResultsRequest{
		WorkflowId: "wf-1",
		TaskIds:    []string{"task-nonexistent"},
	}

	resp, err := server.GetTaskResults(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)
	assert.False(t, resp.Results[0].Found)
	assert.NotNil(t, resp.Results[0].Error)
}

func TestGetTaskResults_Pagination(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	taskIDs := make([]string, 50)
	for i := range taskIDs {
		taskIDs[i] = "task-" + string(rune(i))
	}

	req := &pb.GetTaskResultsRequest{
		WorkflowId: "wf-1",
		TaskIds:    taskIDs,
		Pagination: &pb.PaginationRequest{
			PageSize: 10,
		},
	}

	resp, err := server.GetTaskResults(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 10)
	assert.NotEmpty(t, resp.Pagination.NextPageToken)

	// Get next page
	req.Pagination.PageToken = resp.Pagination.NextPageToken
	resp2, err := server.GetTaskResults(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	assert.Len(t, resp2.Results, 10)
}

func TestSetWorkerPoolSize(t *testing.T) {
	engine := &mockBatchEngine{}
	server := NewBatchServiceServer(engine)

	assert.Equal(t, DefaultWorkerPoolSize, server.workerPoolSize)

	server.SetWorkerPoolSize(20)
	assert.Equal(t, 20, server.workerPoolSize)

	// Invalid size should be ignored
	server.SetWorkerPoolSize(0)
	assert.Equal(t, 20, server.workerPoolSize)

	server.SetWorkerPoolSize(-5)
	assert.Equal(t, 20, server.workerPoolSize)
}

func TestIdempotencyCache(t *testing.T) {
	cache := NewIdempotencyCache(100 * time.Millisecond)

	// Set and get
	cache.Set("key1", "value1")
	val := cache.Get("key1")
	assert.Equal(t, "value1", val)

	// Non-existent key
	val = cache.Get("nonexistent")
	assert.Nil(t, val)

	// Expiration
	cache.Set("key2", "value2")
	time.Sleep(150 * time.Millisecond)
	val = cache.Get("key2")
	assert.Nil(t, val)
}

func TestIsTerminalStatus(t *testing.T) {
	assert.True(t, isTerminalStatus("COMPLETED"))
	assert.True(t, isTerminalStatus("FAILED"))
	assert.True(t, isTerminalStatus("CANCELLED"))
	assert.False(t, isTerminalStatus("PENDING"))
	assert.False(t, isTerminalStatus("RUNNING"))
}
