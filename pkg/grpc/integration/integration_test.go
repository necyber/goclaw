// +build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/grpc/client"
	"github.com/goclaw/goclaw/pkg/grpc/handlers"
	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"github.com/goclaw/goclaw/pkg/grpc/streaming"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// testServer wraps the gRPC server for testing
type testServer struct {
	server       *grpc.Server
	engine       *mockEngine
	listener     net.Listener
	address      string
	workflowSvc  *handlers.WorkflowServiceServer
	streamingSvc *handlers.StreamingServiceServer
	batchSvc     *handlers.BatchServiceServer
}

// mockEngine implements a simple engine for testing
type mockEngine struct {
	mu        sync.RWMutex
	workflows map[string]*handlers.WorkflowStatus
	tasks     map[string]map[string]*handlers.TaskResult
}

func newMockEngine() *mockEngine {
	return &mockEngine{
		workflows: make(map[string]*handlers.WorkflowStatus),
		tasks:     make(map[string]map[string]*handlers.TaskResult),
	}
}

func (m *mockEngine) SubmitWorkflow(ctx context.Context, name string, tasks []handlers.WorkflowTask) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	workflowID := fmt.Sprintf("wf-%d", time.Now().UnixNano())

	taskStatuses := make([]*handlers.TaskStatus, len(tasks))
	for i, t := range tasks {
		taskStatuses[i] = &handlers.TaskStatus{
			TaskID:      t.ID,
			Name:        t.Name,
			Status:      "PENDING",
			StartedAt:   0,
			CompletedAt: 0,
		}
	}

	m.workflows[workflowID] = &handlers.WorkflowStatus{
		WorkflowID: workflowID,
		Name:       name,
		Status:     "PENDING",
		Tasks:      taskStatuses,
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	// Initialize task results
	m.tasks[workflowID] = make(map[string]*handlers.TaskResult)
	for _, t := range tasks {
		m.tasks[workflowID][t.ID] = &handlers.TaskResult{
			TaskID:     t.ID,
			Status:     "PENDING",
			ResultData: []byte{},
		}
	}

	return workflowID, nil
}

func (m *mockEngine) GetWorkflowStatus(ctx context.Context, workflowID string) (*handlers.WorkflowStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ws, exists := m.workflows[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}
	return ws, nil
}

func (m *mockEngine) ListWorkflows(ctx context.Context, filter handlers.WorkflowFilter) ([]*handlers.WorkflowSummary, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var summaries []*handlers.WorkflowSummary
	for _, ws := range m.workflows {
		if filter.StatusFilter == "" || ws.Status == filter.StatusFilter {
			summaries = append(summaries, &handlers.WorkflowSummary{
				WorkflowID: ws.WorkflowID,
				Name:       ws.Name,
				Status:     ws.Status,
				CreatedAt:  ws.CreatedAt,
				UpdatedAt:  ws.UpdatedAt,
			})
		}
	}
	return summaries, "", nil
}

func (m *mockEngine) CancelWorkflow(ctx context.Context, workflowID string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ws, exists := m.workflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}
	ws.Status = "CANCELLED"
	ws.UpdatedAt = time.Now().Unix()
	return nil
}

func (m *mockEngine) GetTaskResult(ctx context.Context, workflowID, taskID string) (*handlers.TaskResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks, exists := m.tasks[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}
	result, exists := tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return result, nil
}

// setupTestServer creates and starts a test gRPC server
func setupTestServer(t *testing.T) *testServer {
	// Create mock engine
	mockEng := newMockEngine()

	// Create listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register services
	workflowSvc := handlers.NewWorkflowServiceServer(mockEng)
	pb.RegisterWorkflowServiceServer(grpcServer, workflowSvc)

	streamingRegistry := streaming.NewSubscriberRegistry()
	streamingSvc := handlers.NewStreamingServiceServer(streamingRegistry)
	pb.RegisterStreamingServiceServer(grpcServer, streamingSvc)

	batchSvc := handlers.NewBatchServiceServer(mockEng)
	pb.RegisterBatchServiceServer(grpcServer, batchSvc)

	// Start server in background
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	return &testServer{
		server:       grpcServer,
		engine:       mockEng,
		listener:     listener,
		address:      address,
		workflowSvc:  workflowSvc,
		streamingSvc: streamingSvc,
		batchSvc:     batchSvc,
	}
}

// teardownTestServer stops the test server
func (ts *testServer) teardown() {
	if ts.server != nil {
		ts.server.Stop()
	}
	if ts.listener != nil {
		ts.listener.Close()
	}
}

// createTestClient creates a test client
func createTestClient(t *testing.T, address string) *client.Client {
	opts := client.DefaultOptions(address)
	opts.Timeout = 5 * time.Second

	c, err := client.NewClient(opts)
	require.NoError(t, err)

	return c
}

// TestIntegration_WorkflowOperations tests workflow CRUD operations
func TestIntegration_WorkflowOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	c := createTestClient(t, ts.address)
	defer c.Close()

	ctx := context.Background()

	t.Run("Submit workflow", func(t *testing.T) {
		req := &pb.SubmitWorkflowRequest{
			Name: "test-workflow",
			Tasks: []*pb.TaskDefinition{
				{Id: "task-1", Name: "Task 1"},
				{Id: "task-2", Name: "Task 2", Dependencies: []string{"task-1"}},
			},
		}

		resp, err := c.Workflows().Submit(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.WorkflowId)
		assert.Nil(t, resp.Error)
	})

	t.Run("Get workflow status", func(t *testing.T) {
		// First submit a workflow
		submitResp, err := c.Workflows().Submit(ctx, &pb.SubmitWorkflowRequest{
			Name: "test-workflow-2",
			Tasks: []*pb.TaskDefinition{
				{Id: "task-1", Name: "Task 1"},
			},
		})
		require.NoError(t, err)
		workflowID := submitResp.WorkflowId

		// Get status
		statusResp, err := c.Workflows().Get(ctx, workflowID)
		require.NoError(t, err)
		require.NotNil(t, statusResp)
		assert.Equal(t, workflowID, statusResp.WorkflowId)
		assert.Equal(t, "test-workflow-2", statusResp.Name)
		assert.Len(t, statusResp.Tasks, 1)
	})

	t.Run("List workflows", func(t *testing.T) {
		// Submit multiple workflows
		for i := 0; i < 3; i++ {
			_, err := c.Workflows().Submit(ctx, &pb.SubmitWorkflowRequest{
				Name: fmt.Sprintf("workflow-%d", i),
				Tasks: []*pb.TaskDefinition{
					{Id: "task-1", Name: "Task 1"},
				},
			})
			require.NoError(t, err)
		}

		// List workflows
		listResp, err := c.Workflows().List(ctx, &pb.ListWorkflowsRequest{})
		require.NoError(t, err)
		require.NotNil(t, listResp)
		assert.GreaterOrEqual(t, len(listResp.Workflows), 1) // At least the ones we just submitted
	})

	t.Run("Cancel workflow", func(t *testing.T) {
		// Submit a workflow
		submitResp, err := c.Workflows().Submit(ctx, &pb.SubmitWorkflowRequest{
			Name: "workflow-to-cancel",
			Tasks: []*pb.TaskDefinition{
				{Id: "task-1", Name: "Task 1"},
			},
		})
		require.NoError(t, err)
		workflowID := submitResp.WorkflowId

		// Cancel it
		cancelResp, err := c.Workflows().Cancel(ctx, workflowID, false)
		require.NoError(t, err)
		require.NotNil(t, cancelResp)
		assert.True(t, cancelResp.Success)

		// Verify status
		statusResp, err := c.Workflows().Get(ctx, workflowID)
		require.NoError(t, err)
		assert.Equal(t, pb.WorkflowStatus_WORKFLOW_STATUS_CANCELLED, statusResp.Status)
	})

	t.Run("Get task result", func(t *testing.T) {
		// Submit a workflow
		submitResp, err := c.Workflows().Submit(ctx, &pb.SubmitWorkflowRequest{
			Name: "workflow-with-task",
			Tasks: []*pb.TaskDefinition{
				{Id: "task-1", Name: "Task 1"},
			},
		})
		require.NoError(t, err)
		workflowID := submitResp.WorkflowId

		// Get task result
		resultResp, err := c.Workflows().GetTaskResult(ctx, workflowID, "task-1")
		require.NoError(t, err)
		require.NotNil(t, resultResp)
		assert.Equal(t, "task-1", resultResp.TaskId)
	})
}

// TestIntegration_WorkflowOperations_Errors tests error handling
func TestIntegration_WorkflowOperations_Errors(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	c := createTestClient(t, ts.address)
	defer c.Close()

	ctx := context.Background()

	t.Run("Get non-existent workflow", func(t *testing.T) {
		resp, err := c.Workflows().Get(ctx, "non-existent-id")
		// The handler returns error in response, not as gRPC error
		if err == nil {
			assert.NotNil(t, resp.Error)
		}
	})

	t.Run("Submit workflow with empty name", func(t *testing.T) {
		_, err := c.Workflows().Submit(ctx, &pb.SubmitWorkflowRequest{
			Name: "",
			Tasks: []*pb.TaskDefinition{
				{Id: "task-1", Name: "Task 1"},
			},
		})
		require.Error(t, err)
		assert.True(t, client.IsInvalidArgument(err))
	})

	t.Run("Submit workflow with no tasks", func(t *testing.T) {
		_, err := c.Workflows().Submit(ctx, &pb.SubmitWorkflowRequest{
			Name:  "workflow",
			Tasks: []*pb.TaskDefinition{},
		})
		require.Error(t, err)
		assert.True(t, client.IsInvalidArgument(err))
	})
}

// TestIntegration_StreamingOperations tests streaming operations
func TestIntegration_StreamingOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	c := createTestClient(t, ts.address)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("Watch workflow", func(t *testing.T) {
		submitResp, err := c.Workflows().Submit(ctx, &pb.SubmitWorkflowRequest{
			Name: "streaming-workflow",
			Tasks: []*pb.TaskDefinition{
				{Id: "task-1", Name: "Task 1"},
			},
		})
		require.NoError(t, err)
		workflowID := submitResp.WorkflowId

		stream, err := c.Streaming().WatchWorkflow(ctx, workflowID, 0)
		require.NoError(t, err)

		update, err := stream.Recv()
		if err == nil {
			assert.Equal(t, workflowID, update.WorkflowId)
			// Sequence number might be 0 initially
		}
	})
}

// TestIntegration_BatchOperations tests batch operations
func TestIntegration_BatchOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	c := createTestClient(t, ts.address)
	defer c.Close()

	ctx := context.Background()

	t.Run("Submit workflows batch", func(t *testing.T) {
		req := &pb.SubmitWorkflowsRequest{
			Workflows: []*pb.SubmitWorkflowRequest{
				{Name: "batch-workflow-1", Tasks: []*pb.TaskDefinition{{Id: "task-1", Name: "Task 1"}}},
				{Name: "batch-workflow-2", Tasks: []*pb.TaskDefinition{{Id: "task-1", Name: "Task 1"}}},
				{Name: "batch-workflow-3", Tasks: []*pb.TaskDefinition{{Id: "task-1", Name: "Task 1"}}},
			},
		}

		resp, err := c.Batch().SubmitWorkflows(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Results, 3)
		for _, result := range resp.Results {
			assert.True(t, result.Success)
			assert.NotEmpty(t, result.WorkflowId)
		}
	})
}

// Benchmark tests
func BenchmarkIntegration_SubmitWorkflow(b *testing.B) {
	ts := setupTestServer(&testing.T{})
	defer ts.teardown()

	c := createTestClient(&testing.T{}, ts.address)
	defer c.Close()

	ctx := context.Background()
	req := &pb.SubmitWorkflowRequest{
		Name: "benchmark-workflow",
		Tasks: []*pb.TaskDefinition{{Id: "task-1", Name: "Task 1"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Workflows().Submit(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIntegration_BatchSubmit(b *testing.B) {
	ts := setupTestServer(&testing.T{})
	defer ts.teardown()

	c := createTestClient(&testing.T{}, ts.address)
	defer c.Close()

	ctx := context.Background()
	req := &pb.SubmitWorkflowsRequest{
		Workflows: []*pb.SubmitWorkflowRequest{
			{Name: "batch-1", Tasks: []*pb.TaskDefinition{{Id: "task-1", Name: "Task 1"}}},
			{Name: "batch-2", Tasks: []*pb.TaskDefinition{{Id: "task-1", Name: "Task 1"}}},
			{Name: "batch-3", Tasks: []*pb.TaskDefinition{{Id: "task-1", Name: "Task 1"}}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Batch().SubmitWorkflows(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
