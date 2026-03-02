package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
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

// WithSagaStore wires persistent saga storage for runtime instances.
func WithSagaStore(store SagaStore) OrchestratorOption {
	return func(orchestrator *SagaOrchestrator) {
		orchestrator.store = store
	}
}

// WithDefinitionStore wires persistent saga definition storage.
func WithDefinitionStore(store SagaDefinitionStore) OrchestratorOption {
	return func(orchestrator *SagaOrchestrator) {
		orchestrator.definitionStore = store
	}
}

// WithMetrics wires metrics recording into saga execution paths.
func WithMetrics(metrics MetricsRecorder) OrchestratorOption {
	return func(orchestrator *SagaOrchestrator) {
		if metrics == nil {
			return
		}
		orchestrator.metrics = metrics
		orchestrator.compensationExecutor.metrics = metrics
	}
}

// SagaOrchestrator executes declarative Saga definitions.
type SagaOrchestrator struct {
	mu                   sync.RWMutex
	instances            map[string]*SagaInstance
	store                SagaStore
	definitionStore      SagaDefinitionStore
	wal                  WAL
	checkpointer         *Checkpointer
	compensationExecutor *CompensationExecutor
	metrics              MetricsRecorder
	maxConcurrent        int
	sema                 chan struct{}
}

// NewSagaOrchestrator creates a Saga orchestrator.
func NewSagaOrchestrator(options ...OrchestratorOption) *SagaOrchestrator {
	orchestrator := &SagaOrchestrator{
		instances:            make(map[string]*SagaInstance),
		compensationExecutor: NewCompensationExecutor(nil, NewInMemoryIdempotencyStore()),
		metrics:              &nopMetricsRecorder{},
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
	ctx, sagaSpan := sagaTracer().Start(ctx, spanSagaExecuteForward)
	sagaSpan.SetAttributes(attribute.String("saga.id", sagaID))
	if definition != nil {
		sagaSpan.SetAttributes(attribute.String("saga.definition", definition.Name))
	}
	defer sagaSpan.End()

	startedAt := time.Now()
	o.metrics.IncActiveSagas()
	defer o.metrics.DecActiveSagas()

	if definition == nil {
		sagaSpan.SetStatus(otelcodes.Error, "definition_nil")
		return nil, fmt.Errorf("saga definition cannot be nil")
	}
	if err := definition.Validate(); err != nil {
		sagaSpan.RecordError(err)
		sagaSpan.SetStatus(otelcodes.Error, "definition_invalid")
		return nil, err
	}

	select {
	case o.sema <- struct{}{}:
	case <-ctx.Done():
		sagaSpan.RecordError(ctx.Err())
		sagaSpan.SetStatus(otelcodes.Error, "cancelled")
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
	if err := o.transitionToAndPersist(sagaCtx, instance, SagaStateRunning); err != nil {
		return nil, err
	}

	layers, err := definition.TopologicalLayers()
	if err != nil {
		return nil, err
	}

	results := make(map[string]any)
	var resultsMu sync.Mutex
	var instanceMu sync.Mutex
	var failedStep string
	var execErr error
	stepSema := make(chan struct{}, definition.MaxConcurrent)

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
				if err := acquireStepSlot(sagaCtx, stepSema); err != nil {
					layerErrCh <- stepFailure{stepID: step.ID, err: err}
					return
				}
				defer releaseStepSlot(stepSema)
				result, err := o.executeStep(sagaCtx, definition, instance, step, input, results, &resultsMu, &instanceMu)
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
		sagaSpan.RecordError(execErr)
		if err := o.setFailureAndPersist(sagaCtx, instance, failedStep, execErr); err != nil {
			sagaSpan.RecordError(err)
			sagaSpan.SetStatus(otelcodes.Error, "checkpoint_failed")
			return nil, err
		}
		switch definition.Policy {
		case ManualCompensate:
			_ = o.transitionToAndPersist(sagaCtx, instance, SagaStatePendingCompensation)
			o.recordExecutionMetrics(instance, startedAt)
			sagaSpan.SetStatus(otelcodes.Error, SagaStatePendingCompensation.String())
			return instance, execErr
		case SkipCompensate:
			_ = o.transitionToAndPersist(sagaCtx, instance, SagaStateCompensationFailed)
			o.recordExecutionMetrics(instance, startedAt)
			sagaSpan.SetStatus(otelcodes.Error, SagaStateCompensationFailed.String())
			return instance, execErr
		default:
			_ = o.transitionToAndPersist(sagaCtx, instance, SagaStateCompensating)
			if compErr := o.compensationExecutor.Execute(sagaCtx, definition, instance, input, execErr); compErr != nil {
				_ = o.transitionToAndPersist(sagaCtx, instance, SagaStateCompensationFailed)
				_ = o.setFailureAndPersist(sagaCtx, instance, failedStep, compErr)
				o.recordExecutionMetrics(instance, startedAt)
				sagaSpan.RecordError(compErr)
				sagaSpan.SetStatus(otelcodes.Error, SagaStateCompensationFailed.String())
				return instance, compErr
			}
			_ = o.transitionToAndPersist(sagaCtx, instance, SagaStateCompensated)
			o.recordExecutionMetrics(instance, startedAt)
			sagaSpan.SetStatus(otelcodes.Error, SagaStateCompensated.String())
			return instance, execErr
		}
	}

	if err := o.transitionToAndPersist(sagaCtx, instance, SagaStateCompleted); err != nil {
		sagaSpan.RecordError(err)
		sagaSpan.SetStatus(otelcodes.Error, "transition_failed")
		return nil, err
	}
	o.recordExecutionMetrics(instance, startedAt)
	sagaSpan.SetStatus(otelcodes.Ok, SagaStateCompleted.String())
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
	startedAt := time.Now()
	o.metrics.IncActiveSagas()
	defer o.metrics.DecActiveSagas()

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

	if err := o.transitionToAndPersist(ctx, instance, SagaStateCompensating); err != nil {
		return nil, err
	}
	if err := o.compensationExecutor.Execute(ctx, definition, instance, input, reason); err != nil {
		_ = o.transitionToAndPersist(ctx, instance, SagaStateCompensationFailed)
		_ = o.setFailureAndPersist(ctx, instance, instance.FailedStep, err)
		o.recordExecutionMetrics(instance, startedAt)
		return instance, err
	}

	if err := o.transitionToAndPersist(ctx, instance, SagaStateCompensated); err != nil {
		return nil, err
	}
	o.recordExecutionMetrics(instance, startedAt)
	return instance, nil
}

// ResumeFromCheckpoint resumes a saga from persisted checkpoint state.
func (o *SagaOrchestrator) ResumeFromCheckpoint(
	ctx context.Context,
	definition *SagaDefinition,
	checkpoint *Checkpoint,
	input any,
) (*SagaInstance, error) {
	ctx, recoverySpan := sagaTracer().Start(ctx, spanSagaRecoveryResume)
	if checkpoint != nil {
		recoverySpan.SetAttributes(
			attribute.String("saga.id", checkpoint.SagaID),
			attribute.String("saga.state", checkpoint.State.String()),
		)
	}
	defer recoverySpan.End()

	if definition == nil {
		recoverySpan.SetStatus(otelcodes.Error, "definition_nil")
		return nil, fmt.Errorf("saga definition cannot be nil")
	}
	if checkpoint == nil {
		recoverySpan.SetStatus(otelcodes.Error, "checkpoint_nil")
		return nil, fmt.Errorf("checkpoint cannot be nil")
	}
	if checkpoint.SagaID == "" {
		recoverySpan.SetStatus(otelcodes.Error, "checkpoint_id_empty")
		return nil, fmt.Errorf("checkpoint saga_id cannot be empty")
	}

	instance := &SagaInstance{
		ID:             checkpoint.SagaID,
		DefinitionName: definition.Name,
		State:          checkpoint.State,
		CompletedSteps: append([]string(nil), checkpoint.CompletedSteps...),
		StepResults:    copyResultMap(checkpoint.StepResults),
		FailedStep:     checkpoint.FailedStep,
		CreatedAt:      checkpoint.LastUpdated,
		UpdatedAt:      checkpoint.LastUpdated,
		Compensated:    make([]string, 0),
	}
	o.saveInstance(instance)

	switch checkpoint.State {
	case SagaStateRunning:
		startedAt := time.Now()
		o.metrics.IncActiveSagas()
		defer o.metrics.DecActiveSagas()

		resumed, err := o.resumeRunning(ctx, definition, instance, input)
		if resumed != nil {
			o.recordExecutionMetrics(resumed, startedAt)
		}
		if err != nil {
			recoverySpan.RecordError(err)
			recoverySpan.SetStatus(otelcodes.Error, "resume_failed")
		} else {
			recoverySpan.SetStatus(otelcodes.Ok, "resumed")
		}
		return resumed, err
	case SagaStateCompensating:
		startedAt := time.Now()
		o.metrics.IncActiveSagas()
		defer o.metrics.DecActiveSagas()

		recoveryErr := fmt.Errorf("resumed compensation from checkpoint")
		if err := o.compensationExecutor.Execute(ctx, definition, instance, input, recoveryErr); err != nil {
			_ = o.transitionToAndPersist(ctx, instance, SagaStateCompensationFailed)
			_ = o.setFailureAndPersist(ctx, instance, instance.FailedStep, err)
			o.recordExecutionMetrics(instance, startedAt)
			recoverySpan.RecordError(err)
			recoverySpan.SetStatus(otelcodes.Error, SagaStateCompensationFailed.String())
			return instance, err
		}
		if err := o.transitionToAndPersist(ctx, instance, SagaStateCompensated); err != nil {
			recoverySpan.RecordError(err)
			recoverySpan.SetStatus(otelcodes.Error, "transition_failed")
			return nil, err
		}
		o.recordExecutionMetrics(instance, startedAt)
		recoverySpan.SetStatus(otelcodes.Ok, SagaStateCompensated.String())
		return instance, nil
	default:
		recoverySpan.SetStatus(otelcodes.Ok, "noop")
		return instance, nil
	}
}

// GetInstance gets one Saga instance by ID.
func (o *SagaOrchestrator) GetInstance(sagaID string) (*SagaInstance, error) {
	o.mu.RLock()
	instance, ok := o.instances[sagaID]
	o.mu.RUnlock()
	if !ok {
		if o.store == nil {
			return nil, ErrSagaNotFound
		}
		stored, err := o.store.Get(context.Background(), sagaID)
		if err != nil {
			return nil, err
		}
		return cloneInstance(stored), nil
	}
	return cloneInstance(instance), nil
}

// ListInstances returns all in-memory saga instances.
func (o *SagaOrchestrator) ListInstances() []*SagaInstance {
	if o.store != nil {
		stored, _, err := o.store.List(context.Background(), SagaListFilter{})
		if err == nil {
			return stored
		}
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	instances := make([]*SagaInstance, 0, len(o.instances))
	for _, instance := range o.instances {
		instances = append(instances, cloneInstance(instance))
	}
	return instances
}

// ListInstancesFiltered lists saga instances with optional state filter and pagination.
func (o *SagaOrchestrator) ListInstancesFiltered(ctx context.Context, filter SagaListFilter) ([]*SagaInstance, int, error) {
	if o.store != nil {
		return o.store.List(ctx, filter)
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	all := make([]*SagaInstance, 0, len(o.instances))
	for _, instance := range o.instances {
		if filter.State != "" && instance.State.String() != filter.State {
			continue
		}
		all = append(all, cloneInstance(instance))
	}

	total := len(all)
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.Offset > total {
		filter.Offset = total
	}
	end := total
	if filter.Limit > 0 && filter.Offset+filter.Limit < end {
		end = filter.Offset + filter.Limit
	}
	return all[filter.Offset:end], total, nil
}

// SaveDefinitionSnapshot persists a serializable definition snapshot for one saga instance.
func (o *SagaOrchestrator) SaveDefinitionSnapshot(ctx context.Context, sagaID string, snapshot *DefinitionSnapshot) error {
	if o.definitionStore == nil {
		return nil
	}
	return o.definitionStore.Save(ctx, sagaID, snapshot)
}

// LoadDefinitionSnapshot loads a persisted definition snapshot for one saga instance.
func (o *SagaOrchestrator) LoadDefinitionSnapshot(ctx context.Context, sagaID string) (*DefinitionSnapshot, error) {
	if o.definitionStore == nil {
		return nil, ErrSagaDefinitionNotFound
	}
	return o.definitionStore.Load(ctx, sagaID)
}

// LoadDefinition loads and reconstructs an executable definition from persisted snapshot.
func (o *SagaOrchestrator) LoadDefinition(ctx context.Context, sagaID string) (*SagaDefinition, any, error) {
	snapshot, err := o.LoadDefinitionSnapshot(ctx, sagaID)
	if err != nil {
		return nil, nil, err
	}
	return BuildDefinitionFromSnapshot(snapshot)
}

func (o *SagaOrchestrator) executeStep(
	ctx context.Context,
	definition *SagaDefinition,
	instance *SagaInstance,
	step *Step,
	input any,
	results map[string]any,
	resultsMu *sync.Mutex,
	instanceMu *sync.Mutex,
) (any, error) {
	ctx, stepSpan := sagaTracer().Start(ctx, spanSagaStepForward)
	stepSpan.SetAttributes(
		attribute.String("saga.id", instance.ID),
		attribute.String("saga.definition", definition.Name),
		attribute.String("saga.step.id", step.ID),
	)
	defer stepSpan.End()

	if err := o.writeWAL(ctx, WALEntry{
		SagaID: instance.ID,
		StepID: step.ID,
		Type:   WALEntryTypeStepStarted,
	}); err != nil {
		stepSpan.RecordError(err)
		stepSpan.SetStatus(otelcodes.Error, "wal_write_failed")
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
		stepSpan.RecordError(err)
		stepSpan.SetStatus(otelcodes.Error, "step_failed")
		return nil, err
	}

	if err := o.writeWAL(ctx, WALEntry{
		SagaID: instance.ID,
		StepID: step.ID,
		Type:   WALEntryTypeStepCompleted,
	}); err != nil {
		stepSpan.RecordError(err)
		stepSpan.SetStatus(otelcodes.Error, "wal_write_failed")
		return nil, err
	}

	if instanceMu != nil {
		instanceMu.Lock()
		defer instanceMu.Unlock()
	}

	if o.checkpointer != nil {
		if err := o.checkpointer.RecordStepCompletion(ctx, instance, step.ID, result); err != nil {
			stepSpan.RecordError(err)
			stepSpan.SetStatus(otelcodes.Error, "checkpoint_failed")
			return nil, err
		}
	} else {
		instance.MarkStepCompleted(step.ID, result)
	}
	o.saveInstance(instance)
	stepSpan.SetStatus(otelcodes.Ok, "completed")

	return result, nil
}

func (o *SagaOrchestrator) saveInstance(instance *SagaInstance) {
	o.mu.Lock()
	o.instances[instance.ID] = cloneInstance(instance)
	o.mu.Unlock()
	if o.store != nil {
		_ = o.store.Save(context.Background(), instance)
	}
}

func (o *SagaOrchestrator) persistInstance(ctx context.Context, instance *SagaInstance) error {
	if instance == nil {
		return fmt.Errorf("saga instance cannot be nil")
	}
	o.saveInstance(instance)
	if o.checkpointer != nil {
		if err := o.checkpointer.SaveSnapshot(ctx, instance); err != nil {
			return err
		}
	}
	return nil
}

func (o *SagaOrchestrator) transitionToAndPersist(ctx context.Context, instance *SagaInstance, next SagaState) error {
	if err := instance.TransitionTo(next); err != nil {
		return err
	}
	return o.persistInstance(ctx, instance)
}

func (o *SagaOrchestrator) setFailureAndPersist(ctx context.Context, instance *SagaInstance, stepID string, failure error) error {
	instance.SetFailure(stepID, failure)
	return o.persistInstance(ctx, instance)
}

func (o *SagaOrchestrator) writeWAL(ctx context.Context, entry WALEntry) error {
	if o.wal == nil {
		return nil
	}
	_, err := o.wal.Append(ctx, entry)
	return err
}

func acquireStepSlot(ctx context.Context, sema chan struct{}) error {
	if sema == nil {
		return nil
	}
	select {
	case sema <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseStepSlot(sema chan struct{}) {
	if sema == nil {
		return
	}
	<-sema
}

func (o *SagaOrchestrator) resumeRunning(
	ctx context.Context,
	definition *SagaDefinition,
	instance *SagaInstance,
	input any,
) (*SagaInstance, error) {
	layers, err := definition.TopologicalLayers()
	if err != nil {
		return nil, err
	}

	completedSet := make(map[string]struct{}, len(instance.CompletedSteps))
	for _, stepID := range instance.CompletedSteps {
		completedSet[stepID] = struct{}{}
	}

	results := copyResultMap(instance.StepResults)
	var resultsMu sync.Mutex
	var instanceMu sync.Mutex

	var failedStep string
	var execErr error
	stepSema := make(chan struct{}, definition.MaxConcurrent)
	for _, layer := range layers {
		var wg sync.WaitGroup
		layerErrCh := make(chan stepFailure, len(layer))

		for _, stepID := range layer {
			if _, done := completedSet[stepID]; done {
				continue
			}
			step := definition.Steps[stepID]
			if step == nil {
				continue
			}

			wg.Add(1)
			go func(step *Step) {
				defer wg.Done()
				if err := acquireStepSlot(ctx, stepSema); err != nil {
					layerErrCh <- stepFailure{stepID: step.ID, err: err}
					return
				}
				defer releaseStepSlot(stepSema)
				result, err := o.executeStep(ctx, definition, instance, step, input, results, &resultsMu, &instanceMu)
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

	if execErr != nil {
		if err := o.setFailureAndPersist(ctx, instance, failedStep, execErr); err != nil {
			return nil, err
		}
		switch definition.Policy {
		case ManualCompensate:
			_ = o.transitionToAndPersist(ctx, instance, SagaStatePendingCompensation)
			return instance, execErr
		case SkipCompensate:
			_ = o.transitionToAndPersist(ctx, instance, SagaStateCompensationFailed)
			return instance, execErr
		default:
			_ = o.transitionToAndPersist(ctx, instance, SagaStateCompensating)
			if compErr := o.compensationExecutor.Execute(ctx, definition, instance, input, execErr); compErr != nil {
				_ = o.transitionToAndPersist(ctx, instance, SagaStateCompensationFailed)
				_ = o.setFailureAndPersist(ctx, instance, failedStep, compErr)
				return instance, compErr
			}
			_ = o.transitionToAndPersist(ctx, instance, SagaStateCompensated)
			return instance, execErr
		}
	}

	if err := o.transitionToAndPersist(ctx, instance, SagaStateCompleted); err != nil {
		return nil, err
	}
	return instance, nil
}

type stepFailure struct {
	stepID string
	err    error
}

func cloneInstance(instance *SagaInstance) *SagaInstance {
	if instance == nil {
		return nil
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()

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

func (o *SagaOrchestrator) recordExecutionMetrics(instance *SagaInstance, startedAt time.Time) {
	if instance == nil {
		return
	}
	status := instance.State.String()
	o.metrics.RecordSagaExecution(status)
	o.metrics.RecordSagaDuration(status, time.Since(startedAt))
}
