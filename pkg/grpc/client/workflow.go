package client

import (
	"context"
	"fmt"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
)

// WorkflowOperations provides high-level workflow operations
type WorkflowOperations struct {
	client *Client
}

// Workflows returns workflow operations
func (c *Client) Workflows() *WorkflowOperations {
	return &WorkflowOperations{client: c}
}

// Submit submits a new workflow
func (w *WorkflowOperations) Submit(ctx context.Context, req *pb.SubmitWorkflowRequest) (*pb.SubmitWorkflowResponse, error) {
	return withRetry(w.client, ctx, func(ctx context.Context) (*pb.SubmitWorkflowResponse, error) {
		return w.client.workflowClient.SubmitWorkflow(ctx, req)
	})
}

// Get retrieves workflow status
func (w *WorkflowOperations) Get(ctx context.Context, workflowID string) (*pb.GetWorkflowStatusResponse, error) {
	req := &pb.GetWorkflowStatusRequest{
		WorkflowId: workflowID,
	}

	return withRetry(w.client, ctx, func(ctx context.Context) (*pb.GetWorkflowStatusResponse, error) {
		return w.client.workflowClient.GetWorkflowStatus(ctx, req)
	})
}

// List lists workflows with optional filtering
func (w *WorkflowOperations) List(ctx context.Context, req *pb.ListWorkflowsRequest) (*pb.ListWorkflowsResponse, error) {
	return withRetry(w.client, ctx, func(ctx context.Context) (*pb.ListWorkflowsResponse, error) {
		return w.client.workflowClient.ListWorkflows(ctx, req)
	})
}

// Cancel cancels a workflow
func (w *WorkflowOperations) Cancel(ctx context.Context, workflowID string, force bool) (*pb.CancelWorkflowResponse, error) {
	req := &pb.CancelWorkflowRequest{
		WorkflowId: workflowID,
		Force:      force,
	}

	return withRetry(w.client, ctx, func(ctx context.Context) (*pb.CancelWorkflowResponse, error) {
		return w.client.workflowClient.CancelWorkflow(ctx, req)
	})
}

// GetTaskResult retrieves a task result
func (w *WorkflowOperations) GetTaskResult(ctx context.Context, workflowID, taskID string) (*pb.GetTaskResultResponse, error) {
	req := &pb.GetTaskResultRequest{
		WorkflowId: workflowID,
		TaskId:     taskID,
	}

	return withRetry(w.client, ctx, func(ctx context.Context) (*pb.GetTaskResultResponse, error) {
		return w.client.workflowClient.GetTaskResult(ctx, req)
	})
}

// ListAll lists all workflows across multiple pages
func (w *WorkflowOperations) ListAll(ctx context.Context, statusFilter pb.WorkflowStatus) ([]*pb.WorkflowSummary, error) {
	var allWorkflows []*pb.WorkflowSummary
	pageToken := ""

	for {
		req := &pb.ListWorkflowsRequest{
			StatusFilter: statusFilter,
			Pagination: &pb.PaginationRequest{
				PageSize:  100,
				PageToken: pageToken,
			},
		}

		resp, err := w.List(ctx, req)
		if err != nil {
			return nil, err
		}

		allWorkflows = append(allWorkflows, resp.Workflows...)

		if resp.Pagination == nil || resp.Pagination.NextPageToken == "" {
			break
		}

		pageToken = resp.Pagination.NextPageToken
	}

	return allWorkflows, nil
}

// WaitForCompletion waits for a workflow to complete
func (w *WorkflowOperations) WaitForCompletion(ctx context.Context, workflowID string) (*pb.GetWorkflowStatusResponse, error) {
	stream, err := w.client.streamingClient.WatchWorkflow(ctx, &pb.WatchWorkflowRequest{
		WorkflowId: workflowID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to watch workflow: %w", err)
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			return nil, fmt.Errorf("stream error: %w", err)
		}

		// Check if terminal state
		if isTerminalWorkflowStatus(update.Status) {
			return w.Get(ctx, workflowID)
		}
	}
}

// isTerminalWorkflowStatus checks if status is terminal
func isTerminalWorkflowStatus(status pb.WorkflowStatus) bool {
	return status == pb.WorkflowStatus_WORKFLOW_STATUS_COMPLETED ||
		status == pb.WorkflowStatus_WORKFLOW_STATUS_FAILED ||
		status == pb.WorkflowStatus_WORKFLOW_STATUS_CANCELLED
}
