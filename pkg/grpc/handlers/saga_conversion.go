package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"github.com/goclaw/goclaw/pkg/saga"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func buildSagaDefinitionFromProto(req *pb.SubmitSagaRequest) (*saga.SagaDefinition, any, *saga.DefinitionSnapshot, error) {
	snapshot, err := snapshotFromProto(req)
	if err != nil {
		return nil, nil, nil, err
	}

	definition, input, err := saga.BuildDefinitionFromSnapshot(snapshot)
	if err != nil {
		return nil, nil, nil, err
	}
	return definition, input, snapshot, nil
}

func snapshotFromProto(req *pb.SubmitSagaRequest) (*saga.DefinitionSnapshot, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.Steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	policy, err := protoPolicyToSagaPolicy(req.Policy)
	if err != nil {
		return nil, err
	}

	snapshot := &saga.DefinitionSnapshot{
		Name:          req.Name,
		Policy:        policyToString(policy),
		TimeoutMS:     int(req.TimeoutMs),
		StepTimeoutMS: int(req.StepTimeoutMs),
		Metadata:      cloneStringMap(req.Metadata),
		Steps:         make([]saga.DefinitionStepSnapshot, 0, len(req.Steps)),
	}
	if req.Input != nil {
		snapshot.Input = req.Input.AsMap()
	}
	for i, step := range req.Steps {
		if step == nil {
			return nil, fmt.Errorf("step %d cannot be nil", i)
		}
		if step.Id == "" {
			return nil, fmt.Errorf("step %d id is required", i)
		}
		snapshot.Steps = append(snapshot.Steps, saga.DefinitionStepSnapshot{
			ID:                 step.Id,
			DependsOn:          append([]string(nil), step.DependsOn...),
			DelayMS:            int(step.DelayMs),
			ShouldFail:         step.ShouldFail,
			TimeoutMS:          int(step.TimeoutMs),
			EnableCompensation: step.EnableCompensation,
			SkipCompensation:   step.SkipCompensation,
		})
	}
	return snapshot, nil
}

func protoPolicyToSagaPolicy(policy pb.SagaCompensationPolicy) (saga.CompensationPolicy, error) {
	switch policy {
	case pb.SagaCompensationPolicy_SAGA_COMPENSATION_POLICY_UNSPECIFIED,
		pb.SagaCompensationPolicy_SAGA_COMPENSATION_POLICY_AUTO:
		return saga.AutoCompensate, nil
	case pb.SagaCompensationPolicy_SAGA_COMPENSATION_POLICY_MANUAL:
		return saga.ManualCompensate, nil
	case pb.SagaCompensationPolicy_SAGA_COMPENSATION_POLICY_SKIP:
		return saga.SkipCompensate, nil
	default:
		return saga.AutoCompensate, fmt.Errorf("unsupported compensation policy: %s", policy.String())
	}
}

func policyToString(policy saga.CompensationPolicy) string {
	switch policy {
	case saga.ManualCompensate:
		return "manual"
	case saga.SkipCompensate:
		return "skip"
	default:
		return "auto"
	}
}

func sagaStateToProto(state saga.SagaState) pb.SagaState {
	switch state {
	case saga.SagaStateCreated:
		return pb.SagaState_SAGA_STATE_CREATED
	case saga.SagaStateRunning:
		return pb.SagaState_SAGA_STATE_RUNNING
	case saga.SagaStateCompleted:
		return pb.SagaState_SAGA_STATE_COMPLETED
	case saga.SagaStateCompensating:
		return pb.SagaState_SAGA_STATE_COMPENSATING
	case saga.SagaStatePendingCompensation:
		return pb.SagaState_SAGA_STATE_PENDING_COMPENSATION
	case saga.SagaStateCompensated:
		return pb.SagaState_SAGA_STATE_COMPENSATED
	case saga.SagaStateCompensationFailed:
		return pb.SagaState_SAGA_STATE_COMPENSATION_FAILED
	case saga.SagaStateRecovering:
		return pb.SagaState_SAGA_STATE_RECOVERING
	default:
		return pb.SagaState_SAGA_STATE_UNSPECIFIED
	}
}

func protoStateFilterToString(state pb.SagaState) (string, error) {
	switch state {
	case pb.SagaState_SAGA_STATE_UNSPECIFIED:
		return "", nil
	case pb.SagaState_SAGA_STATE_CREATED:
		return saga.SagaStateCreated.String(), nil
	case pb.SagaState_SAGA_STATE_RUNNING:
		return saga.SagaStateRunning.String(), nil
	case pb.SagaState_SAGA_STATE_COMPLETED:
		return saga.SagaStateCompleted.String(), nil
	case pb.SagaState_SAGA_STATE_COMPENSATING:
		return saga.SagaStateCompensating.String(), nil
	case pb.SagaState_SAGA_STATE_PENDING_COMPENSATION:
		return saga.SagaStatePendingCompensation.String(), nil
	case pb.SagaState_SAGA_STATE_COMPENSATED:
		return saga.SagaStateCompensated.String(), nil
	case pb.SagaState_SAGA_STATE_COMPENSATION_FAILED:
		return saga.SagaStateCompensationFailed.String(), nil
	case pb.SagaState_SAGA_STATE_RECOVERING:
		return saga.SagaStateRecovering.String(), nil
	default:
		return "", fmt.Errorf("unsupported state filter: %s", state.String())
	}
}

func sagaInstanceToStatus(instance *saga.SagaInstance) (*pb.GetSagaStatusResponse, error) {
	results := make([]*pb.SagaStepResult, 0, len(instance.StepResults))
	keys := make([]string, 0, len(instance.StepResults))
	for stepID := range instance.StepResults {
		keys = append(keys, stepID)
	}
	sort.Strings(keys)

	for _, stepID := range keys {
		raw, err := json.Marshal(instance.StepResults[stepID])
		if err != nil {
			return nil, err
		}
		results = append(results, &pb.SagaStepResult{
			StepId:     stepID,
			ResultJson: raw,
		})
	}

	return &pb.GetSagaStatusResponse{
		SagaId:           instance.ID,
		Name:             instance.DefinitionName,
		State:            sagaStateToProto(instance.State),
		CompletedSteps:   append([]string(nil), instance.CompletedSteps...),
		CompensatedSteps: append([]string(nil), instance.Compensated...),
		FailedStep:       instance.FailedStep,
		FailureReason:    instance.FailureReason,
		StepResults:      results,
		CreatedAt:        timestamppb.New(instance.CreatedAt),
		UpdatedAt:        timestamppb.New(instance.UpdatedAt),
		StartedAt:        timestampPtr(instance.StartedAt),
		CompletedAt:      timestampPtr(instance.CompletedAt),
	}, nil
}

func sagaInstanceToSummary(instance *saga.SagaInstance) *pb.SagaSummary {
	return &pb.SagaSummary{
		SagaId:      instance.ID,
		Name:        instance.DefinitionName,
		State:       sagaStateToProto(instance.State),
		CreatedAt:   timestamppb.New(instance.CreatedAt),
		CompletedAt: timestampPtr(instance.CompletedAt),
	}
}

func sagaInstanceToWatchEvent(instance *saga.SagaInstance) *pb.WatchSagaEvent {
	return &pb.WatchSagaEvent{
		SagaId:        instance.ID,
		State:         sagaStateToProto(instance.State),
		FailedStep:    instance.FailedStep,
		FailureReason: instance.FailureReason,
		UpdatedAt:     timestamppb.New(instance.UpdatedAt),
	}
}

func parseOffsetToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(token)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid page token")
	}
	return offset, nil
}

func timestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	copied := make(map[string]string, len(source))
	for k, v := range source {
		copied[k] = v
	}
	return copied
}
