package client

import (
	"context"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
)

// AdminOperations provides high-level admin operations
type AdminOperations struct {
	client *Client
}

// Admin returns admin operations
func (c *Client) Admin() *AdminOperations {
	return &AdminOperations{client: c}
}

// GetEngineStatus retrieves engine status
func (a *AdminOperations) GetEngineStatus(ctx context.Context) (*pb.GetEngineStatusResponse, error) {
	req := &pb.GetEngineStatusRequest{}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.GetEngineStatusResponse, error) {
		return a.client.adminClient.GetEngineStatus(ctx, req)
	})
}

// UpdateConfig updates engine configuration
func (a *AdminOperations) UpdateConfig(ctx context.Context, config map[string]string) (*pb.UpdateConfigResponse, error) {
	req := &pb.UpdateConfigRequest{
		ConfigUpdates: config,
	}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.UpdateConfigResponse, error) {
		return a.client.adminClient.UpdateConfig(ctx, req)
	})
}

// ListClusterNodes lists cluster nodes
func (a *AdminOperations) ListClusterNodes(ctx context.Context) (*pb.ManageClusterResponse, error) {
	req := &pb.ManageClusterRequest{
		Operation: pb.ClusterOperation_CLUSTER_OPERATION_LIST,
	}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.ManageClusterResponse, error) {
		return a.client.adminClient.ManageCluster(ctx, req)
	})
}

// AddClusterNode adds a node to the cluster
func (a *AdminOperations) AddClusterNode(ctx context.Context, nodeID, address string) (*pb.ManageClusterResponse, error) {
	req := &pb.ManageClusterRequest{
		Operation: pb.ClusterOperation_CLUSTER_OPERATION_ADD,
		NodeId:    nodeID,
		NodeAddress: address,
	}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.ManageClusterResponse, error) {
		return a.client.adminClient.ManageCluster(ctx, req)
	})
}

// RemoveClusterNode removes a node from the cluster
func (a *AdminOperations) RemoveClusterNode(ctx context.Context, nodeID string) (*pb.ManageClusterResponse, error) {
	req := &pb.ManageClusterRequest{
		Operation: pb.ClusterOperation_CLUSTER_OPERATION_REMOVE,
		NodeId:    nodeID,
	}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.ManageClusterResponse, error) {
		return a.client.adminClient.ManageCluster(ctx, req)
	})
}

// PauseWorkflows pauses workflow execution
func (a *AdminOperations) PauseWorkflows(ctx context.Context) (*pb.PauseWorkflowsResponse, error) {
	req := &pb.PauseWorkflowsRequest{
		Confirmation: true,
	}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.PauseWorkflowsResponse, error) {
		return a.client.adminClient.PauseWorkflows(ctx, req)
	})
}

// ResumeWorkflows resumes workflow execution
func (a *AdminOperations) ResumeWorkflows(ctx context.Context) (*pb.ResumeWorkflowsResponse, error) {
	req := &pb.ResumeWorkflowsRequest{}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.ResumeWorkflowsResponse, error) {
		return a.client.adminClient.ResumeWorkflows(ctx, req)
	})
}

// PurgeWorkflows purges completed workflows
func (a *AdminOperations) PurgeWorkflows(ctx context.Context, ageThresholdHours int32, confirm bool) (*pb.PurgeWorkflowsResponse, error) {
	req := &pb.PurgeWorkflowsRequest{
		AgeThresholdHours: ageThresholdHours,
		Confirmation:      confirm,
	}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.PurgeWorkflowsResponse, error) {
		return a.client.adminClient.PurgeWorkflows(ctx, req)
	})
}

// GetLaneStats retrieves lane statistics
func (a *AdminOperations) GetLaneStats(ctx context.Context) (*pb.GetLaneStatsResponse, error) {
	req := &pb.GetLaneStatsRequest{}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.GetLaneStatsResponse, error) {
		return a.client.adminClient.GetLaneStats(ctx, req)
	})
}

// ExportMetrics exports metrics
func (a *AdminOperations) ExportMetrics(ctx context.Context, format pb.MetricsFormat) (*pb.ExportMetricsResponse, error) {
	req := &pb.ExportMetricsRequest{
		Format: format,
	}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.ExportMetricsResponse, error) {
		return a.client.adminClient.ExportMetrics(ctx, req)
	})
}

// GetDebugInfo retrieves debug information
func (a *AdminOperations) GetDebugInfo(ctx context.Context, debugType pb.DebugInfoType, durationSeconds int32) (*pb.GetDebugInfoResponse, error) {
	req := &pb.GetDebugInfoRequest{
		Type:            debugType,
		DurationSeconds: durationSeconds,
	}

	return withRetry(a.client, ctx, func(ctx context.Context) (*pb.GetDebugInfoResponse, error) {
		return a.client.adminClient.GetDebugInfo(ctx, req)
	})
}
