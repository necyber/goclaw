package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"github.com/goclaw/goclaw/pkg/saga"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SagaServiceServer implements gRPC SagaService.
type SagaServiceServer struct {
	pb.UnimplementedSagaServiceServer

	orchestrator    *saga.SagaOrchestrator
	checkpointStore saga.CheckpointStore

	defMu       sync.RWMutex
	definitions map[string]*saga.SagaDefinition
}

// NewSagaServiceServer creates a new Saga service server.
func NewSagaServiceServer(orchestrator *saga.SagaOrchestrator, checkpointStore saga.CheckpointStore) *SagaServiceServer {
	return &SagaServiceServer{
		orchestrator:    orchestrator,
		checkpointStore: checkpointStore,
		definitions:     make(map[string]*saga.SagaDefinition),
	}
}

// SubmitSaga submits a Saga for asynchronous execution.
func (s *SagaServiceServer) SubmitSaga(ctx context.Context, req *pb.SubmitSagaRequest) (*pb.SubmitSagaResponse, error) {
	_ = ctx
	if s.orchestrator == nil {
		return nil, status.Error(codes.Unavailable, "saga orchestrator unavailable")
	}

	definition, input, err := buildSagaDefinitionFromProto(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	sagaID := uuid.NewString()
	s.defMu.Lock()
	s.definitions[sagaID] = definition
	s.defMu.Unlock()

	go func() {
		_, _ = s.orchestrator.ExecuteWithID(context.Background(), sagaID, definition, input)
	}()

	return &pb.SubmitSagaResponse{
		SagaId:    sagaID,
		Name:      definition.Name,
		State:     pb.SagaState_SAGA_STATE_RUNNING,
		CreatedAt: timestamppb.Now(),
	}, nil
}

// GetSagaStatus gets one saga runtime status.
func (s *SagaServiceServer) GetSagaStatus(ctx context.Context, req *pb.GetSagaStatusRequest) (*pb.GetSagaStatusResponse, error) {
	if s.orchestrator == nil {
		return nil, status.Error(codes.Unavailable, "saga orchestrator unavailable")
	}
	if req == nil || req.SagaId == "" {
		return nil, status.Error(codes.InvalidArgument, "saga_id is required")
	}

	instance, err := s.orchestrator.GetInstance(req.SagaId)
	if err != nil {
		if errors.Is(err, saga.ErrSagaNotFound) {
			return nil, status.Error(codes.NotFound, "saga not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp, err := sagaInstanceToStatus(instance)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

// ListSagas lists saga instances with optional state filter and pagination.
func (s *SagaServiceServer) ListSagas(ctx context.Context, req *pb.ListSagasRequest) (*pb.ListSagasResponse, error) {
	if s.orchestrator == nil {
		return nil, status.Error(codes.Unavailable, "saga orchestrator unavailable")
	}
	if req == nil {
		req = &pb.ListSagasRequest{}
	}

	pageSize := int32(20)
	pageToken := ""
	if req.Pagination != nil {
		if req.Pagination.PageSize > 0 {
			pageSize = req.Pagination.PageSize
		}
		pageToken = req.Pagination.PageToken
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	offset, err := parseOffsetToken(pageToken)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	stateFilter, err := protoStateFilterToString(req.StateFilter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	instances, total, err := s.orchestrator.ListInstancesFiltered(ctx, saga.SagaListFilter{
		State:  stateFilter,
		Limit:  int(pageSize),
		Offset: offset,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*pb.SagaSummary, 0, len(instances))
	for _, instance := range instances {
		items = append(items, sagaInstanceToSummary(instance))
	}

	nextToken := ""
	if offset+len(instances) < total {
		nextToken = fmt.Sprintf("%d", offset+len(instances))
	}

	return &pb.ListSagasResponse{
		Sagas: items,
		Pagination: &pb.PaginationResponse{
			NextPageToken: nextToken,
			TotalCount:    int32(total),
		},
	}, nil
}

// CompensateSaga triggers manual compensation.
func (s *SagaServiceServer) CompensateSaga(ctx context.Context, req *pb.CompensateSagaRequest) (*pb.CompensateSagaResponse, error) {
	if s.orchestrator == nil {
		return nil, status.Error(codes.Unavailable, "saga orchestrator unavailable")
	}
	if req == nil || req.SagaId == "" {
		return nil, status.Error(codes.InvalidArgument, "saga_id is required")
	}

	definition := s.getDefinition(req.SagaId)
	if definition == nil {
		return nil, status.Error(codes.NotFound, "saga definition not found")
	}

	reason := errors.New("manual compensation requested")
	if strings.TrimSpace(req.Reason) != "" {
		reason = errors.New(req.Reason)
	}

	instance, err := s.orchestrator.TriggerCompensation(ctx, req.SagaId, definition, nil, reason)
	if err != nil {
		if errors.Is(err, saga.ErrSagaNotFound) {
			return nil, status.Error(codes.NotFound, "saga not found")
		}
		if strings.Contains(err.Error(), "pending-compensation") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CompensateSagaResponse{
		SagaId: req.SagaId,
		State:  sagaStateToProto(instance.State),
	}, nil
}

// WatchSaga streams saga state changes until terminal state.
func (s *SagaServiceServer) WatchSaga(req *pb.WatchSagaRequest, stream pb.SagaService_WatchSagaServer) error {
	if s.orchestrator == nil {
		return status.Error(codes.Unavailable, "saga orchestrator unavailable")
	}
	if req == nil || req.SagaId == "" {
		return status.Error(codes.InvalidArgument, "saga_id is required")
	}

	pollInterval := 200 * time.Millisecond
	if req.PollIntervalMs > 0 {
		pollInterval = time.Duration(req.PollIntervalMs) * time.Millisecond
	}

	ctx := stream.Context()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	lastFingerprint := ""
	first := true
	notFoundRetries := 0

	for {
		instance, err := s.orchestrator.GetInstance(req.SagaId)
		if err != nil {
			if errors.Is(err, saga.ErrSagaNotFound) {
				if first && notFoundRetries < 5 {
					notFoundRetries++
					select {
					case <-ctx.Done():
						return status.Error(codes.Canceled, "client disconnected")
					case <-ticker.C:
						continue
					}
				}
				return status.Error(codes.NotFound, "saga not found")
			}
			return status.Error(codes.Internal, err.Error())
		}
		notFoundRetries = 0

		fingerprint := fmt.Sprintf(
			"%s|%s|%s|%d",
			instance.State.String(),
			instance.FailedStep,
			instance.FailureReason,
			instance.UpdatedAt.UnixNano(),
		)
		if first || fingerprint != lastFingerprint {
			if err := stream.Send(sagaInstanceToWatchEvent(instance)); err != nil {
				return status.Errorf(codes.Internal, "failed to send saga event: %v", err)
			}
			lastFingerprint = fingerprint
			first = false
		}

		if instance.State.IsTerminal() {
			return nil
		}

		select {
		case <-ctx.Done():
			return status.Error(codes.Canceled, "client disconnected")
		case <-ticker.C:
		}
	}
}

func (s *SagaServiceServer) getDefinition(sagaID string) *saga.SagaDefinition {
	s.defMu.RLock()
	defer s.defMu.RUnlock()
	return s.definitions[sagaID]
}
