package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Mock AdminEngine for testing
type mockAdminEngine struct {
	state            string
	metrics          *EngineMetrics
	uptime           time.Time
	lastError        string
	healthy          bool
	updateConfigErr  error
	listNodesErr     error
	addNodeErr       error
	removeNodeErr    error
	pauseErr         error
	resumeErr        error
	purgeErr         error
	laneStatsErr     error
	exportMetricsErr error
	nodes            []*ClusterNode
	pausedCount      int32
	resumedCount     int32
	purgedCount      int32
	laneStats        []*LaneStats
	metricsData      string
}

func (m *mockAdminEngine) GetEngineState() string {
	return m.state
}

func (m *mockAdminEngine) GetEngineMetrics() *EngineMetrics {
	return m.metrics
}

func (m *mockAdminEngine) GetUptime() time.Time {
	return m.uptime
}

func (m *mockAdminEngine) GetLastError() string {
	return m.lastError
}

func (m *mockAdminEngine) IsHealthy() bool {
	return m.healthy
}

func (m *mockAdminEngine) UpdateConfig(ctx context.Context, updates map[string]string, persist bool) (map[string]string, error) {
	if m.updateConfigErr != nil {
		return nil, m.updateConfigErr
	}
	return updates, nil
}

func (m *mockAdminEngine) ListClusterNodes(ctx context.Context) ([]*ClusterNode, error) {
	if m.listNodesErr != nil {
		return nil, m.listNodesErr
	}
	return m.nodes, nil
}

func (m *mockAdminEngine) AddClusterNode(ctx context.Context, nodeID, address string) error {
	if m.addNodeErr != nil {
		return m.addNodeErr
	}
	m.nodes = append(m.nodes, &ClusterNode{
		NodeID:   nodeID,
		Address:  address,
		Role:     "worker",
		Healthy:  true,
		JoinedAt: time.Now(),
	})
	return nil
}

func (m *mockAdminEngine) RemoveClusterNode(ctx context.Context, nodeID string) error {
	if m.removeNodeErr != nil {
		return m.removeNodeErr
	}
	for i, n := range m.nodes {
		if n.NodeID == nodeID {
			m.nodes = append(m.nodes[:i], m.nodes[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockAdminEngine) PauseWorkflows(ctx context.Context) (int32, error) {
	if m.pauseErr != nil {
		return 0, m.pauseErr
	}
	return m.pausedCount, nil
}

func (m *mockAdminEngine) ResumeWorkflows(ctx context.Context) (int32, error) {
	if m.resumeErr != nil {
		return 0, m.resumeErr
	}
	return m.resumedCount, nil
}

func (m *mockAdminEngine) PurgeWorkflows(ctx context.Context, ageThresholdHours int32, dryRun bool) (int32, error) {
	if m.purgeErr != nil {
		return 0, m.purgeErr
	}
	return m.purgedCount, nil
}

func (m *mockAdminEngine) GetLaneStats(ctx context.Context, laneName string) ([]*LaneStats, error) {
	if m.laneStatsErr != nil {
		return nil, m.laneStatsErr
	}
	return m.laneStats, nil
}

func (m *mockAdminEngine) ExportMetrics(ctx context.Context, format string, prefixFilter string) (string, error) {
	if m.exportMetricsErr != nil {
		return "", m.exportMetricsErr
	}
	return m.metricsData, nil
}

func TestGetEngineStatus(t *testing.T) {
	uptime := time.Now().Add(-1 * time.Hour)
	mockEngine := &mockAdminEngine{
		state: "running",
		metrics: &EngineMetrics{
			ActiveWorkflows:    5,
			CompletedWorkflows: 100,
			RunningTasks:       10,
			QueueDepth:         20,
			MemoryUsageBytes:   1024 * 1024 * 100,
			GoroutineCount:     50,
			CPUUsagePercent:    25.5,
		},
		uptime:    uptime,
		lastError: "",
		healthy:   true,
	}

	server := NewAdminServiceServer(mockEngine)

	resp, err := server.GetEngineStatus(context.Background(), &pb.GetEngineStatusRequest{})
	if err != nil {
		t.Fatalf("GetEngineStatus failed: %v", err)
	}

	if resp.State != pb.EngineState_ENGINE_STATE_RUNNING {
		t.Errorf("Expected state RUNNING, got %v", resp.State)
	}

	if resp.Metrics.ActiveWorkflows != 5 {
		t.Errorf("Expected 5 active workflows, got %d", resp.Metrics.ActiveWorkflows)
	}

	if !resp.Healthy {
		t.Error("Expected healthy=true")
	}
}

func TestUpdateConfig(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.UpdateConfigRequest
		mockErr     error
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "successful update",
			req: &pb.UpdateConfigRequest{
				ConfigUpdates: map[string]string{"key1": "value1"},
				Persist:       true,
			},
			expectError: false,
		},
		{
			name: "dry run",
			req: &pb.UpdateConfigRequest{
				ConfigUpdates: map[string]string{"key1": "value1"},
				DryRun:        true,
			},
			expectError: false,
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "empty updates",
			req: &pb.UpdateConfigRequest{
				ConfigUpdates: map[string]string{},
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "engine error",
			req: &pb.UpdateConfigRequest{
				ConfigUpdates: map[string]string{"key1": "value1"},
			},
			mockErr:     errors.New("update failed"),
			expectError: false, // Returns error in response, not gRPC error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := &mockAdminEngine{
				updateConfigErr: tt.mockErr,
			}
			server := NewAdminServiceServer(mockEngine)

			resp, err := server.UpdateConfig(context.Background(), tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if st, ok := status.FromError(err); ok {
					if st.Code() != tt.errorCode {
						t.Errorf("Expected error code %v, got %v", tt.errorCode, st.Code())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.mockErr != nil {
					if resp.Success {
						t.Error("Expected success=false for engine error")
					}
					if resp.Error == nil {
						t.Error("Expected error in response")
					}
				} else {
					if !resp.Success {
						t.Error("Expected success=true")
					}
				}
			}
		})
	}
}

func TestManageCluster(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.ManageClusterRequest
		mockNodes   []*ClusterNode
		mockErr     error
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "list nodes",
			req: &pb.ManageClusterRequest{
				Operation: pb.ClusterOperation_CLUSTER_OPERATION_LIST,
			},
			mockNodes: []*ClusterNode{
				{NodeID: "node1", Address: "localhost:9090", Role: "master", Healthy: true},
			},
			expectError: false,
		},
		{
			name: "add node",
			req: &pb.ManageClusterRequest{
				Operation:   pb.ClusterOperation_CLUSTER_OPERATION_ADD,
				NodeId:      "node2",
				NodeAddress: "localhost:9091",
			},
			expectError: false,
		},
		{
			name: "add node missing params",
			req: &pb.ManageClusterRequest{
				Operation: pb.ClusterOperation_CLUSTER_OPERATION_ADD,
				NodeId:    "node2",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "remove node",
			req: &pb.ManageClusterRequest{
				Operation:    pb.ClusterOperation_CLUSTER_OPERATION_REMOVE,
				NodeId:       "node1",
				Confirmation: true,
			},
			mockNodes: []*ClusterNode{
				{NodeID: "node1", Address: "localhost:9090"},
			},
			expectError: false,
		},
		{
			name: "remove node without confirmation",
			req: &pb.ManageClusterRequest{
				Operation: pb.ClusterOperation_CLUSTER_OPERATION_REMOVE,
				NodeId:    "node1",
			},
			expectError: true,
			errorCode:   codes.FailedPrecondition,
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := &mockAdminEngine{
				nodes:         tt.mockNodes,
				listNodesErr:  tt.mockErr,
				addNodeErr:    tt.mockErr,
				removeNodeErr: tt.mockErr,
			}
			server := NewAdminServiceServer(mockEngine)

			resp, err := server.ManageCluster(context.Background(), tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !resp.Success {
					t.Error("Expected success=true")
				}
			}
		})
	}
}

func TestPauseWorkflows(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.PauseWorkflowsRequest
		pausedCount int32
		mockErr     error
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "successful pause",
			req: &pb.PauseWorkflowsRequest{
				Confirmation: true,
			},
			pausedCount: 5,
			expectError: false,
		},
		{
			name: "no confirmation",
			req: &pb.PauseWorkflowsRequest{
				Confirmation: false,
			},
			expectError: true,
			errorCode:   codes.FailedPrecondition,
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorCode:   codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := &mockAdminEngine{
				pausedCount: tt.pausedCount,
				pauseErr:    tt.mockErr,
			}
			server := NewAdminServiceServer(mockEngine)

			resp, err := server.PauseWorkflows(context.Background(), tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !resp.Success {
					t.Error("Expected success=true")
				}
				if resp.PausedCount != tt.pausedCount {
					t.Errorf("Expected %d paused, got %d", tt.pausedCount, resp.PausedCount)
				}
			}
		})
	}
}

func TestResumeWorkflows(t *testing.T) {
	mockEngine := &mockAdminEngine{
		resumedCount: 3,
	}
	server := NewAdminServiceServer(mockEngine)

	resp, err := server.ResumeWorkflows(context.Background(), &pb.ResumeWorkflowsRequest{})
	if err != nil {
		t.Fatalf("ResumeWorkflows failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.ResumedCount != 3 {
		t.Errorf("Expected 3 resumed, got %d", resp.ResumedCount)
	}
}

func TestPurgeWorkflows(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.PurgeWorkflowsRequest
		purgedCount int32
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "successful purge",
			req: &pb.PurgeWorkflowsRequest{
				AgeThresholdHours: 24,
				Confirmation:      true,
			},
			purgedCount: 10,
			expectError: false,
		},
		{
			name: "dry run",
			req: &pb.PurgeWorkflowsRequest{
				AgeThresholdHours: 24,
				DryRun:            true,
			},
			purgedCount: 10,
			expectError: false,
		},
		{
			name: "no confirmation",
			req: &pb.PurgeWorkflowsRequest{
				AgeThresholdHours: 24,
			},
			expectError: true,
			errorCode:   codes.FailedPrecondition,
		},
		{
			name: "invalid threshold",
			req: &pb.PurgeWorkflowsRequest{
				AgeThresholdHours: 0,
				Confirmation:      true,
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := &mockAdminEngine{
				purgedCount: tt.purgedCount,
			}
			server := NewAdminServiceServer(mockEngine)

			resp, err := server.PurgeWorkflows(context.Background(), tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !resp.Success {
					t.Error("Expected success=true")
				}
				if resp.PurgedCount != tt.purgedCount {
					t.Errorf("Expected %d purged, got %d", tt.purgedCount, resp.PurgedCount)
				}
			}
		})
	}
}

func TestGetLaneStats(t *testing.T) {
	mockStats := []*LaneStats{
		{
			LaneName:         "default",
			QueueDepth:       10,
			WorkerCount:      5,
			ThroughputPerSec: 100.5,
			ErrorRate:        0.01,
		},
	}

	mockEngine := &mockAdminEngine{
		laneStats: mockStats,
	}
	server := NewAdminServiceServer(mockEngine)

	resp, err := server.GetLaneStats(context.Background(), &pb.GetLaneStatsRequest{
		LaneName: "default",
	})
	if err != nil {
		t.Fatalf("GetLaneStats failed: %v", err)
	}

	if len(resp.Lanes) != 1 {
		t.Errorf("Expected 1 lane, got %d", len(resp.Lanes))
	}

	if resp.Lanes[0].LaneName != "default" {
		t.Errorf("Expected lane name 'default', got %s", resp.Lanes[0].LaneName)
	}
}

func TestExportMetrics(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.ExportMetricsRequest
		metricsData string
		expectError bool
	}{
		{
			name: "json format",
			req: &pb.ExportMetricsRequest{
				Format: pb.MetricsFormat_METRICS_FORMAT_JSON,
			},
			metricsData: `{"metric1": 100}`,
			expectError: false,
		},
		{
			name: "prometheus format",
			req: &pb.ExportMetricsRequest{
				Format: pb.MetricsFormat_METRICS_FORMAT_PROMETHEUS,
			},
			metricsData: "# HELP metric1\nmetric1 100",
			expectError: false,
		},
		{
			name:        "nil request defaults to json",
			req:         nil,
			metricsData: `{"metric1": 100}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := &mockAdminEngine{
				metricsData: tt.metricsData,
			}
			server := NewAdminServiceServer(mockEngine)

			resp, err := server.ExportMetrics(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("ExportMetrics failed: %v", err)
			}

			if resp.MetricsData != tt.metricsData {
				t.Errorf("Expected metrics data %s, got %s", tt.metricsData, resp.MetricsData)
			}
		})
	}
}

func TestGetDebugInfo(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.GetDebugInfoRequest
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "goroutine info",
			req: &pb.GetDebugInfoRequest{
				Type: pb.DebugInfoType_DEBUG_INFO_TYPE_GOROUTINE,
			},
			expectError: false,
		},
		{
			name: "heap info",
			req: &pb.GetDebugInfoRequest{
				Type: pb.DebugInfoType_DEBUG_INFO_TYPE_HEAP,
			},
			expectError: false,
		},
		{
			name: "cpu info",
			req: &pb.GetDebugInfoRequest{
				Type:            pb.DebugInfoType_DEBUG_INFO_TYPE_CPU,
				DurationSeconds: 5,
			},
			expectError: false,
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "unsupported type",
			req: &pb.GetDebugInfoRequest{
				Type: pb.DebugInfoType_DEBUG_INFO_TYPE_UNSPECIFIED,
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := &mockAdminEngine{}
			server := NewAdminServiceServer(mockEngine)

			resp, err := server.GetDebugInfo(context.Background(), tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(resp.DebugData) == 0 {
					t.Error("Expected debug data, got empty")
				}
			}
		})
	}
}
