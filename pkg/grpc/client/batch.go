package client

import (
	"context"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
)

// BatchOperations provides high-level batch operations
type BatchOperations struct {
	client *Client
}

// Batch returns batch operations
func (c *Client) Batch() *BatchOperations {
	return &BatchOperations{client: c}
}

// SubmitWorkflows submits multiple workflows
func (b *BatchOperations) SubmitWorkflows(ctx context.Context, req *pb.SubmitWorkflowsRequest) (*pb.SubmitWorkflowsResponse, error) {
	return withRetry(b.client, ctx, func(ctx context.Context) (*pb.SubmitWorkflowsResponse, error) {
		return b.client.batchClient.SubmitWorkflows(ctx, req)
	})
}

// GetWorkflowStatuses retrieves statuses for multiple workflows
func (b *BatchOperations) GetWorkflowStatuses(ctx context.Context, workflowIDs []string) (*pb.GetWorkflowStatusesResponse, error) {
	req := &pb.GetWorkflowStatusesRequest{
		WorkflowIds: workflowIDs,
	}

	return withRetry(b.client, ctx, func(ctx context.Context) (*pb.GetWorkflowStatusesResponse, error) {
		return b.client.batchClient.GetWorkflowStatuses(ctx, req)
	})
}

// CancelWorkflows cancels multiple workflows
func (b *BatchOperations) CancelWorkflows(ctx context.Context, workflowIDs []string, force bool) (*pb.CancelWorkflowsResponse, error) {
	req := &pb.CancelWorkflowsRequest{
		WorkflowIds: workflowIDs,
		Force:       force,
	}

	return withRetry(b.client, ctx, func(ctx context.Context) (*pb.CancelWorkflowsResponse, error) {
		return b.client.batchClient.CancelWorkflows(ctx, req)
	})
}

// GetTaskResults retrieves results for multiple tasks
func (b *BatchOperations) GetTaskResults(ctx context.Context, workflowID string, taskIDs []string) (*pb.GetTaskResultsResponse, error) {
	req := &pb.GetTaskResultsRequest{
		WorkflowId: workflowID,
		TaskIds:    taskIDs,
	}

	return withRetry(b.client, ctx, func(ctx context.Context) (*pb.GetTaskResultsResponse, error) {
		return b.client.batchClient.GetTaskResults(ctx, req)
	})
}

// SubmitWorkflowsAtomic submits workflows in atomic mode (all-or-nothing)
func (b *BatchOperations) SubmitWorkflowsAtomic(ctx context.Context, workflows []*pb.SubmitWorkflowRequest, idempotencyKey string) (*pb.SubmitWorkflowsResponse, error) {
	req := &pb.SubmitWorkflowsRequest{
		Workflows:      workflows,
		Atomic:         true,
		IdempotencyKey: idempotencyKey,
	}

	return b.SubmitWorkflows(ctx, req)
}

// SubmitWorkflowsOrdered submits workflows in ordered mode (sequential)
func (b *BatchOperations) SubmitWorkflowsOrdered(ctx context.Context, workflows []*pb.SubmitWorkflowRequest) (*pb.SubmitWorkflowsResponse, error) {
	req := &pb.SubmitWorkflowsRequest{
		Workflows: workflows,
		Ordered:   true,
	}

	return b.SubmitWorkflows(ctx, req)
}

// GetAllWorkflowStatuses retrieves all workflow statuses across pages
func (b *BatchOperations) GetAllWorkflowStatuses(ctx context.Context, workflowIDs []string) ([]*pb.WorkflowStatusResult, error) {
	var allResults []*pb.WorkflowStatusResult
	pageToken := ""

	for {
		req := &pb.GetWorkflowStatusesRequest{
			WorkflowIds: workflowIDs,
			Pagination: &pb.PaginationRequest{
				PageSize:  100,
				PageToken: pageToken,
			},
		}

		resp, err := b.client.batchClient.GetWorkflowStatuses(ctx, req)
		if err != nil {
			return nil, err
		}

		allResults = append(allResults, resp.Results...)

		if resp.Pagination == nil || resp.Pagination.NextPageToken == "" {
			break
		}

		pageToken = resp.Pagination.NextPageToken
	}

	return allResults, nil
}

// GetAllTaskResults retrieves all task results across pages
func (b *BatchOperations) GetAllTaskResults(ctx context.Context, workflowID string, taskIDs []string) ([]*pb.TaskResultDetail, error) {
	var allResults []*pb.TaskResultDetail
	pageToken := ""

	for {
		req := &pb.GetTaskResultsRequest{
			WorkflowId: workflowID,
			TaskIds:    taskIDs,
			Pagination: &pb.PaginationRequest{
				PageSize:  100,
				PageToken: pageToken,
			},
		}

		resp, err := b.client.batchClient.GetTaskResults(ctx, req)
		if err != nil {
			return nil, err
		}

		allResults = append(allResults, resp.Results...)

		if resp.Pagination == nil || resp.Pagination.NextPageToken == "" {
			break
		}

		pageToken = resp.Pagination.NextPageToken
	}

	return allResults, nil
}
