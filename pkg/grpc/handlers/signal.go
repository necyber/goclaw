package handlers

import (
	"context"
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"github.com/goclaw/goclaw/pkg/signal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SignalServiceServer implements the gRPC SignalService.
type SignalServiceServer struct {
	pb.UnimplementedSignalServiceServer
	bus            signal.Bus
	collectTimeout time.Duration
}

// NewSignalServiceServer creates a new signal service server.
func NewSignalServiceServer(bus signal.Bus) *SignalServiceServer {
	return &SignalServiceServer{
		bus:            bus,
		collectTimeout: 30 * time.Second,
	}
}

// SignalTask publishes a signal to a task or collects results from tasks.
func (s *SignalServiceServer) SignalTask(ctx context.Context, req *pb.SignalTaskRequest) (*pb.SignalTaskResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	if s.bus == nil {
		return &pb.SignalTaskResponse{
			Success: false,
			Error: &pb.Error{
				Code:    "SIGNAL_BUS_NOT_CONFIGURED",
				Message: "signal bus not configured",
			},
		}, nil
	}

	switch req.Type {
	case pb.SignalType_SIGNAL_TYPE_STEER:
		if req.TaskId == "" {
			return nil, status.Error(codes.InvalidArgument, "task_id is required")
		}
		if len(req.Parameters) == 0 {
			return nil, status.Error(codes.InvalidArgument, "parameters are required")
		}
		params := make(map[string]interface{}, len(req.Parameters))
		for key, value := range req.Parameters {
			params[key] = value
		}
		if err := signal.SendSteer(ctx, s.bus, req.TaskId, params); err != nil {
			return &pb.SignalTaskResponse{
				Success: false,
				Error: &pb.Error{
					Code:    "SIGNAL_STEER_FAILED",
					Message: err.Error(),
				},
			}, nil
		}
		return &pb.SignalTaskResponse{Success: true}, nil

	case pb.SignalType_SIGNAL_TYPE_INTERRUPT:
		if req.TaskId == "" {
			return nil, status.Error(codes.InvalidArgument, "task_id is required")
		}
		timeout := time.Duration(req.TimeoutMs) * time.Millisecond
		if err := signal.SendInterrupt(ctx, s.bus, req.TaskId, req.Graceful, req.Reason, timeout); err != nil {
			return &pb.SignalTaskResponse{
				Success: false,
				Error: &pb.Error{
					Code:    "SIGNAL_INTERRUPT_FAILED",
					Message: err.Error(),
				},
			}, nil
		}
		return &pb.SignalTaskResponse{Success: true}, nil

	case pb.SignalType_SIGNAL_TYPE_COLLECT:
		taskIDs := req.TaskIds
		if len(taskIDs) == 0 && req.TaskId != "" {
			taskIDs = []string{req.TaskId}
		}
		if len(taskIDs) == 0 {
			return nil, status.Error(codes.InvalidArgument, "task_ids are required")
		}
		timeout := time.Duration(req.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = s.collectTimeout
		}
		collector := signal.NewCollector(s.bus, taskIDs, timeout)
		results, err := collector.Collect(ctx)
		resp := &pb.SignalTaskResponse{
			Success: err == nil,
			Results: make([]*pb.CollectResult, 0, len(results)),
		}
		for taskID, payload := range results {
			if payload == nil {
				continue
			}
			resp.Results = append(resp.Results, &pb.CollectResult{
				TaskId:       taskID,
				ResultData:   payload.Result,
				ErrorMessage: payload.Error,
			})
		}
		if err != nil {
			resp.Error = &pb.Error{
				Code:    "COLLECT_INCOMPLETE",
				Message: err.Error(),
			}
		}
		return resp, nil

	default:
		return nil, status.Error(codes.InvalidArgument, "unknown signal type")
	}
}
