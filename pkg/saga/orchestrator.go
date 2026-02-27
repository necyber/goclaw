package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// ErrSagaNotFound is returned when saga instance cannot be located.
var ErrSagaNotFound = errors.New("saga instance not found")

// OrchestratorOption customizes SagaOrchestrator initialization.
type OrchestratorOption func(orchestrator *SagaOrchestrator)

// WithMaxConcurrentSagas sets maximum concurrent saga executions.
func WithMaxConcurrentSagas(max int) OrchestratorOption {
	return func(orchestrator *SagaOrchestrator) {
		if max > 0 {
			orchestrator.maxConcurrent = max
			orchestrator.sema = make(chan struct{}, max)
		}
	}
}

// WithWAL wires WAL persistence into the orchestrator.
func WithWAL(wal WAL) OrchestratorOption {
	return func(orchestrator *SagaOrchestrator) {
		orchestrator.wal = wal
		orchestrator.compensationExecutor.wal = wal
	}
}

// WithCheckpointer wires checkpoint support into the orchestrator.
func WithCheckpointer(checkpointer *Checkpointer) OrchestratorOption {
	return func(orchestrator *SagaOrchestrator) {
		orchestrator.checkpointer = checkpointer
	}
}

// WithIdempotencyStore wires idempotency store into compensation executor.
func WithIdempotencyStore(store IdempotencyStore) OrchestratorOption {
	return func(orchestrator *SagaOrchestrator) {
		orchestrator.compensationExecutor.idempotencyStore = store
	}
}

// SagaOrchestrator executes declarative Saga definitions.
type SagaOrchestrator struct {
	mu                   sync.RWMutex
	instances            map[string]*SagaInstance
	wal                  WAL
	checkpointer         *Checkpointer
	compensationExecutor *CompensationExecutor
	maxConcurrent        int
	sema                 chan struct{}
}

// NewSagaOrchestrator creates a Saga orchestrator.
func NewSagaOrchestrator(options ...OrchestratorOption) *SagaOrchestrator {
	orchestrator := &SagaOrchestrator{
		instances:            make(map[string]*SagaInstance),
		compensationExecutor: NewCompensationExecutor(nil, NewInMemoryIdempotencyStore()),
		maxConcurrent:        100,
		sema:                 make(chan struct{}, 100),
	}
	for _, option := range options {
		if option != nil {
			option(orchestrator)
		}
	}
	return orchestrator
}

// Execute runs a Saga definition from start to terminal state.
func (o *SagaOrchestrator) Execute(ctx context.Context, definition *SagaDefinition, input any) (*SagaInstance, error) {
	return o.ExecuteWithID(ctx, uuid.NewString(), definition, input)
}

// ExecuteWithID runs a saga using a provided instance ID.
func (o *SagaOrchestrator) ExecuteWithID(
	ctx context.Context,
	sagaID string,
	definition *SagaDefinition,
	input any,
) (*SagaInstance, error) {
	if definition == nil {
		return nil, fmt.Errorf("saga definition cannot be nil")
	}
	if err := definition.Validate(); err != nil {
		return nil, err
	}

	select {
	case o.sema <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-o.sema }()

	sagaCtx := ctx
	cancel := func() {}
	if definition.Timeout > 0 {
		sagaCtx, cancel = context.WithTimeout(ctx, definition.Timeout)
	}
	defer cancel()

	instance := NewSagaInstance(sagaID, definition)
	if err := instance.TransitionTo(SagaStateRunning); err != nil {
		return nil, err
	}
	o.saveInstance(instance)

	layers, err := definition.TopologicalLayers()
	if err != nil {
		return nil, err
	}

	results := make(map[string]any)
	var resultsMu sync.Mutex
	var failedStep string
	var execErr error

	for _, layer := range layers {
		var wg sync.WaitGroup
		layerErrCh := make(chan stepFailure, len(layer))

		for _, stepID := range layer {
			step := definition.Steps[stepID]
			if step == nil {
				continue
			}

			wg.Add(1)
			go func(step *Step) {
				defer wg.Done()
				result, err := o.executeStep(sagaCtx, definition, instance, step, input, results, &resultsMu)
				if err != nil {
					layerErrCh <- stepFailure{stepID: step.ID, err: err}
					return
				}
				resultsMu.Lock()
				results[step.ID] = result
				resultsMu.Unlock()
			}(step)
		}

		wg.Wait()
		close(layerErrCh)
		if failure, ok := <-layerErrCh; ok {
			failedStep = failure.stepID
			execErr = failure.err
			break
		}
	}

	if execErr == nil && sagaCtx.Err() != nil {
		failedStep = "saga-timeout"
		execErr = sagaCtx.Err()
	}

	if execErr != nil {
		instance.SetFailure(failedStep, execErr)
		switch definition.Policy {
		case ManualCompensate:
			_ = instance.TransitionTo(SagaStatePendingCompensation)
			o.saveInstance(instance)
			return instance, execErr
		case SkipCompensate:
			_ = instance.TransitionTo(SagaStateCompensationFailed)
			o.saveInstance(instance)
			return instance, execErr
		default:
			_ = instance.TransitionTo(SagaStateCompensating)
			if compErr := o.compensationExecutor.Execute(sagaCtx, definition, instance, input, execErr); compErr != nil {
				_ = instance.TransitionTo(SagaStateCompensationFailed)
				instance.SetFailure(failedStep, compErr)
				o.saveInstance(instance)
				return instance, compErr
			}
			_ = instance.TransitionTo(SagaStateCompensated)
			o.saveInstance(instance)
			return instance, execErr
		}
	}

	if err := instance.TransitionTo(SagaStateCompleted); err != nil {
		return nil, err
	}
	o.saveInstance(instance)
	return instance, nil
}

// TriggerCompensation manually starts compensation for pending-compensation saga.
func (o *SagaOrchestrator) TriggerCompensation(
	ctx context.Context,
	sagaID string,
	definition *SagaDefinition,
	input any,
	reason error,
) (*SagaInstance, error) {
	if definition == nil {
		return nil, fmt.Errorf("saga definition cannot be nil")
	}
	instance, err := o.GetInstance(sagaID)
	if err != nil {
		return nil, err
	}
	if instance.State != SagaStatePendingCompensation {
		return nil, fmt.Errorf("manual compensation requires pending-compensation state, got %s", instance.State)
	}

	if err := instance.TransitionTo(SagaStateCompensating); err != nil {
		return nil, err
	}
	if err := o.compensationExecutor.Execute(ctx, definition, instance, input, reason); err != nil {
		_ = instance.TransitionTo(SagaStateCompensationFailed)
		instance.SetFailure(instance.FailedStep, err)
		o.saveInstance(instance)
		return instance, err
	}

	if err := instance.TransitionTo(SagaStateCompensated); err != nil {
		return nil, err
	}
	o.saveInstance(instance)
	return instance, nil
}

// GetInstance gets one Saga instance by ID.
func (o *SagaOrchestrator) GetInstance(sagaID string) (*SagaInstance, error) {
	o.mu.RLock()
	instance, ok := o.instances[sagaID]
	o.mu.RUnlock()
	if !ok {
		return nil, ErrSagaNotFound
	}
	return cloneInstance(instance), nil
}

// ListInstances returns all in-memory saga instances.
func (o *SagaOrchestrator) ListInstances() []*SagaInstance {
	o.mu.RLock()
	defer o.mu.RUnlock()

	instances := make([]*SagaInstance, 0, len(o.instances))
	for _, instance := range o.instances {
		instances = append(instances, cloneInstance(instance))
	}
	return instances
}

func (o *SagaOrchestrator) executeStep(
	ctx context.Context,
	definition *SagaDefinition,
	instance *SagaInstance,
	step *Step,
	input any,
	results map[string]any,
	resultsMu *sync.Mutex,
) (any, error) {
	if err := o.writeWAL(ctx, WALEntry{
		SagaID: instance.ID,
		StepID: step.ID,
		Type:   WALEntryTypeStepStarted,
	}); err != nil {
		return nil, err
	}

	stepCtx := ctx
	cancel := func() {}
	if step.Timeout > 0 {
		stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
	} else if definition.DefaultStepTimeout > 0 {
		stepCtx, cancel = context.WithTimeout(ctx, definition.DefaultStepTimeout)
	}
	defer cancel()

	resultsMu.Lock()
	snapshot := copyResultMap(results)
	resultsMu.Unlock()

	result, err := step.Action(stepCtx, &StepContext{
		SagaID:  instance.ID,
		StepID:  step.ID,
		Input:   input,
		Results: snapshot,
	})
	if err == nil && stepCtx.Err() != nil {
		err = stepCtx.Err()
	}
	if err != nil {
		_ = o.writeWAL(ctx, WALEntry{
			SagaID: instance.ID,
			StepID: step.ID,
			Type:   WALEntryTypeStepFailed,
			Data:   []byte(err.Error()),
		})
		return nil, err
	}

	if err := o.writeWAL(ctx, WALEntry{
		SagaID: instance.ID,
		StepID: step.ID,
		Type:   WALEntryTypeStepCompleted,
	}); err != nil {
		return nil, err
	}

	if o.checkpointer != nil {
		if err := o.checkpointer.RecordStepCompletion(ctx, instance, step.ID, result); err != nil {
			return nil, err
		}
	} else {
		instance.MarkStepCompleted(step.ID, result)
	}
	o.saveInstance(instance)

	return result, nil
}

func (o *SagaOrchestrator) saveInstance(instance *SagaInstance) {
	o.mu.Lock()
	o.instances[instance.ID] = cloneInstance(instance)
	o.mu.Unlock()
}

func (o *SagaOrchestrator) writeWAL(ctx context.Context, entry WALEntry) error {
	if o.wal == nil {
		return nil
	}
	_, err := o.wal.Append(ctx, entry)
	return err
}

type stepFailure struct {
	stepID string
	err    error
}

func cloneInstance(instance *SagaInstance) *SagaInstance {
	if instance == nil {
		return nil
	}

	completed := make([]string, len(instance.CompletedSteps))
	copy(completed, instance.CompletedSteps)
	compensated := make([]string, len(instance.Compensated))
	copy(compensated, instance.Compensated)

	clone := &SagaInstance{
		ID:             instance.ID,
		DefinitionName: instance.DefinitionName,
		State:          instance.State,
		CompletedSteps: completed,
		Compensated:    compensated,
		FailedStep:     instance.FailedStep,
		FailureReason:  instance.FailureReason,
		StepResults:    copyResultMap(instance.StepResults),
		CreatedAt:      instance.CreatedAt,
		UpdatedAt:      instance.UpdatedAt,
	}
	if instance.StartedAt != nil {
		started := *instance.StartedAt
		clone.StartedAt = &started
	}
	if instance.CompletedAt != nil {
		finished := *instance.CompletedAt
		clone.CompletedAt = &finished
	}
	return clone
}
