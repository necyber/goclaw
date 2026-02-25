package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/pprof"
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AdminEngine defines the interface for admin operations
type AdminEngine interface {
	GetEngineState() string
	GetEngineMetrics() *EngineMetrics
	GetUptime() time.Time
	GetLastError() string
	IsHealthy() bool
	UpdateConfig(ctx context.Context, updates map[string]string, persist bool) (map[string]string, error)
	ListClusterNodes(ctx context.Context) ([]*ClusterNode, error)
	AddClusterNode(ctx context.Context, nodeID, address string) error
	RemoveClusterNode(ctx context.Context, nodeID string) error
	PauseWorkflows(ctx context.Context) (int32, error)
	ResumeWorkflows(ctx context.Context) (int32, error)
	PurgeWorkflows(ctx context.Context, ageThresholdHours int32, dryRun bool) (int32, error)
	GetLaneStats(ctx context.Context, laneName string) ([]*LaneStats, error)
	ExportMetrics(ctx context.Context, format string, prefixFilter string) (string, error)
}

// EngineMetrics represents engine runtime metrics
type EngineMetrics struct {
	ActiveWorkflows    int64
	CompletedWorkflows int64
	RunningTasks       int64
	QueueDepth         int64
	MemoryUsageBytes   int64
	GoroutineCount     int32
	CPUUsagePercent    float64
}

// ClusterNode represents a cluster node
type ClusterNode struct {
	NodeID   string
	Address  string
	Role     string
	Healthy  bool
	JoinedAt time.Time
}

// LaneStats represents lane statistics
type LaneStats struct {
	LaneName        string
	QueueDepth      int32
	WorkerCount     int32
	ThroughputPerSec float64
	ErrorRate       float64
}

// AdminServiceServer implements the gRPC AdminService
type AdminServiceServer struct {
	pb.UnimplementedAdminServiceServer
	engine AdminEngine
}

// NewAdminServiceServer creates a new admin service server
func NewAdminServiceServer(engine AdminEngine) *AdminServiceServer {
	return &AdminServiceServer{
		engine: engine,
	}
}

// GetEngineStatus returns the current engine status and metrics
func (s *AdminServiceServer) GetEngineStatus(ctx context.Context, req *pb.GetEngineStatusRequest) (*pb.GetEngineStatusResponse, error) {
	state := s.engine.GetEngineState()
	metrics := s.engine.GetEngineMetrics()
	uptime := s.engine.GetUptime()
	lastError := s.engine.GetLastError()
	healthy := s.engine.IsHealthy()

	return &pb.GetEngineStatusResponse{
		State: convertToProtoEngineState(state),
		Metrics: &pb.EngineMetrics{
			ActiveWorkflows:    metrics.ActiveWorkflows,
			CompletedWorkflows: metrics.CompletedWorkflows,
			RunningTasks:       metrics.RunningTasks,
			QueueDepth:         metrics.QueueDepth,
			MemoryUsageBytes:   metrics.MemoryUsageBytes,
			GoroutineCount:     metrics.GoroutineCount,
			CpuUsagePercent:    metrics.CPUUsagePercent,
		},
		UptimeSince: timestamppb.New(uptime),
		LastError:   lastError,
		Healthy:     healthy,
	}, nil
}

// UpdateConfig updates engine configuration
func (s *AdminServiceServer) UpdateConfig(ctx context.Context, req *pb.UpdateConfigRequest) (*pb.UpdateConfigResponse, error) {
	if req == nil || len(req.ConfigUpdates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "config_updates is required")
	}

	// Dry run mode - validate only
	if req.DryRun {
		return &pb.UpdateConfigResponse{
			Success:        true,
			AppliedChanges: req.ConfigUpdates,
		}, nil
	}

	// Apply config updates
	applied, err := s.engine.UpdateConfig(ctx, req.ConfigUpdates, req.Persist)
	if err != nil {
		return &pb.UpdateConfigResponse{
			Success: false,
			Error: &pb.Error{
				Code:    "CONFIG_UPDATE_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &pb.UpdateConfigResponse{
		Success:        true,
		AppliedChanges: applied,
	}, nil
}

// ManageCluster manages cluster nodes
func (s *AdminServiceServer) ManageCluster(ctx context.Context, req *pb.ManageClusterRequest) (*pb.ManageClusterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	var err error
	var nodes []*ClusterNode

	switch req.Operation {
	case pb.ClusterOperation_CLUSTER_OPERATION_LIST:
		nodes, err = s.engine.ListClusterNodes(ctx)
		if err != nil {
			return &pb.ManageClusterResponse{
				Success: false,
				Error: &pb.Error{
					Code:    "LIST_NODES_FAILED",
					Message: err.Error(),
				},
			}, nil
		}

	case pb.ClusterOperation_CLUSTER_OPERATION_ADD:
		if req.NodeId == "" || req.NodeAddress == "" {
			return nil, status.Error(codes.InvalidArgument, "node_id and node_address are required for ADD operation")
		}
		err = s.engine.AddClusterNode(ctx, req.NodeId, req.NodeAddress)
		if err != nil {
			return &pb.ManageClusterResponse{
				Success: false,
				Error: &pb.Error{
					Code:    "ADD_NODE_FAILED",
					Message: err.Error(),
				},
			}, nil
		}
		nodes, _ = s.engine.ListClusterNodes(ctx)

	case pb.ClusterOperation_CLUSTER_OPERATION_REMOVE:
		if req.NodeId == "" {
			return nil, status.Error(codes.InvalidArgument, "node_id is required for REMOVE operation")
		}
		if !req.Confirmation {
			return nil, status.Error(codes.FailedPrecondition, "confirmation is required for destructive operations")
		}
		err = s.engine.RemoveClusterNode(ctx, req.NodeId)
		if err != nil {
			return &pb.ManageClusterResponse{
				Success: false,
				Error: &pb.Error{
					Code:    "REMOVE_NODE_FAILED",
					Message: err.Error(),
				},
			}, nil
		}
		nodes, _ = s.engine.ListClusterNodes(ctx)

	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported cluster operation")
	}

	// Convert nodes to proto format
	pbNodes := make([]*pb.ClusterNode, len(nodes))
	for i, n := range nodes {
		pbNodes[i] = &pb.ClusterNode{
			NodeId:   n.NodeID,
			Address:  n.Address,
			Role:     n.Role,
			Healthy:  n.Healthy,
			JoinedAt: timestamppb.New(n.JoinedAt),
		}
	}

	return &pb.ManageClusterResponse{
		Success: true,
		Nodes:   pbNodes,
	}, nil
}

// PauseWorkflows pauses all active workflows
func (s *AdminServiceServer) PauseWorkflows(ctx context.Context, req *pb.PauseWorkflowsRequest) (*pb.PauseWorkflowsResponse, error) {
	if req == nil || !req.Confirmation {
		return nil, status.Error(codes.FailedPrecondition, "confirmation is required for pause operation")
	}

	count, err := s.engine.PauseWorkflows(ctx)
	if err != nil {
		return &pb.PauseWorkflowsResponse{
			Success: false,
			Error: &pb.Error{
				Code:    "PAUSE_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &pb.PauseWorkflowsResponse{
		Success:     true,
		PausedCount: count,
	}, nil
}

// ResumeWorkflows resumes all paused workflows
func (s *AdminServiceServer) ResumeWorkflows(ctx context.Context, req *pb.ResumeWorkflowsRequest) (*pb.ResumeWorkflowsResponse, error) {
	count, err := s.engine.ResumeWorkflows(ctx)
	if err != nil {
		return &pb.ResumeWorkflowsResponse{
			Success: false,
			Error: &pb.Error{
				Code:    "RESUME_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &pb.ResumeWorkflowsResponse{
		Success:      true,
		ResumedCount: count,
	}, nil
}

// PurgeWorkflows purges old completed workflows
func (s *AdminServiceServer) PurgeWorkflows(ctx context.Context, req *pb.PurgeWorkflowsRequest) (*pb.PurgeWorkflowsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	if req.AgeThresholdHours <= 0 {
		return nil, status.Error(codes.InvalidArgument, "age_threshold_hours must be positive")
	}

	if !req.DryRun && !req.Confirmation {
		return nil, status.Error(codes.FailedPrecondition, "confirmation is required for purge operation")
	}

	count, err := s.engine.PurgeWorkflows(ctx, req.AgeThresholdHours, req.DryRun)
	if err != nil {
		return &pb.PurgeWorkflowsResponse{
			Success: false,
			Error: &pb.Error{
				Code:    "PURGE_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &pb.PurgeWorkflowsResponse{
		Success:     true,
		PurgedCount: count,
	}, nil
}

// GetLaneStats returns lane statistics
func (s *AdminServiceServer) GetLaneStats(ctx context.Context, req *pb.GetLaneStatsRequest) (*pb.GetLaneStatsResponse, error) {
	laneName := ""
	if req != nil {
		laneName = req.LaneName
	}

	stats, err := s.engine.GetLaneStats(ctx, laneName)
	if err != nil {
		return &pb.GetLaneStatsResponse{
			Error: &pb.Error{
				Code:    "GET_STATS_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	// Convert to proto format
	pbStats := make([]*pb.LaneStats, len(stats))
	for i, s := range stats {
		pbStats[i] = &pb.LaneStats{
			LaneName:         s.LaneName,
			QueueDepth:       s.QueueDepth,
			WorkerCount:      s.WorkerCount,
			ThroughputPerSec: s.ThroughputPerSec,
			ErrorRate:        s.ErrorRate,
		}
	}

	return &pb.GetLaneStatsResponse{
		Lanes: pbStats,
	}, nil
}

// ExportMetrics exports metrics in the requested format
func (s *AdminServiceServer) ExportMetrics(ctx context.Context, req *pb.ExportMetricsRequest) (*pb.ExportMetricsResponse, error) {
	if req == nil {
		req = &pb.ExportMetricsRequest{
			Format: pb.MetricsFormat_METRICS_FORMAT_JSON,
		}
	}

	format := "json"
	switch req.Format {
	case pb.MetricsFormat_METRICS_FORMAT_PROMETHEUS:
		format = "prometheus"
	case pb.MetricsFormat_METRICS_FORMAT_JSON:
		format = "json"
	}

	data, err := s.engine.ExportMetrics(ctx, format, req.PrefixFilter)
	if err != nil {
		return &pb.ExportMetricsResponse{
			Error: &pb.Error{
				Code:    "EXPORT_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &pb.ExportMetricsResponse{
		MetricsData: data,
	}, nil
}

// GetDebugInfo returns debug information
func (s *AdminServiceServer) GetDebugInfo(ctx context.Context, req *pb.GetDebugInfoRequest) (*pb.GetDebugInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	var data []byte
	var err error

	switch req.Type {
	case pb.DebugInfoType_DEBUG_INFO_TYPE_GOROUTINE:
		data, err = getGoroutineProfile()
	case pb.DebugInfoType_DEBUG_INFO_TYPE_HEAP:
		data, err = getHeapProfile()
	case pb.DebugInfoType_DEBUG_INFO_TYPE_CPU:
		duration := time.Duration(req.DurationSeconds) * time.Second
		if duration <= 0 {
			duration = 30 * time.Second
		}
		data, err = getCPUProfile(ctx, duration)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported debug info type")
	}

	if err != nil {
		return &pb.GetDebugInfoResponse{
			Error: &pb.Error{
				Code:    "DEBUG_INFO_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &pb.GetDebugInfoResponse{
		DebugData: data,
	}, nil
}

// Helper functions for debug info

func getGoroutineProfile() ([]byte, error) {
	profile := pprof.Lookup("goroutine")
	if profile == nil {
		return nil, fmt.Errorf("goroutine profile not available")
	}

	var buf []byte
	// Get goroutine stack traces
	buf = make([]byte, 1024*1024) // 1MB buffer
	n := runtime.Stack(buf, true)
	return buf[:n], nil
}

func getHeapProfile() ([]byte, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	data := map[string]interface{}{
		"alloc":         memStats.Alloc,
		"total_alloc":   memStats.TotalAlloc,
		"sys":           memStats.Sys,
		"num_gc":        memStats.NumGC,
		"heap_alloc":    memStats.HeapAlloc,
		"heap_sys":      memStats.HeapSys,
		"heap_idle":     memStats.HeapIdle,
		"heap_inuse":    memStats.HeapInuse,
		"heap_released": memStats.HeapReleased,
		"heap_objects":  memStats.HeapObjects,
	}

	return json.Marshal(data)
}

func getCPUProfile(ctx context.Context, duration time.Duration) ([]byte, error) {
	// Note: CPU profiling requires writing to a buffer
	// This is a simplified version that returns CPU stats
	var buf []byte
	data := map[string]interface{}{
		"num_cpu":      runtime.NumCPU(),
		"num_goroutine": runtime.NumGoroutine(),
		"duration_sec": duration.Seconds(),
		"note":         "Full CPU profiling requires pprof endpoint",
	}

	buf, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// convertToProtoEngineState converts engine state string to proto enum
func convertToProtoEngineState(state string) pb.EngineState {
	switch state {
	case "idle":
		return pb.EngineState_ENGINE_STATE_IDLE
	case "running":
		return pb.EngineState_ENGINE_STATE_RUNNING
	case "stopped":
		return pb.EngineState_ENGINE_STATE_STOPPED
	case "error":
		return pb.EngineState_ENGINE_STATE_ERROR
	default:
		return pb.EngineState_ENGINE_STATE_UNSPECIFIED
	}
}
