package saga

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// IdempotencyStore tracks executed compensation operations.
type IdempotencyStore interface {
	Seen(key string) bool
	Mark(key string)
}

// InMemoryIdempotencyStore is a thread-safe idempotency store.
type InMemoryIdempotencyStore struct {
	store sync.Map
}

// NewInMemoryIdempotencyStore creates a new in-memory idempotency store.
func NewInMemoryIdempotencyStore() *InMemoryIdempotencyStore {
	return &InMemoryIdempotencyStore{}
}

// Seen checks whether a key was already recorded.
func (s *InMemoryIdempotencyStore) Seen(key string) bool {
	_, ok := s.store.Load(key)
	return ok
}

// Mark records one idempotency key.
func (s *InMemoryIdempotencyStore) Mark(key string) {
	s.store.Store(key, struct{}{})
}

// CompensationExecutor performs reverse execution for completed steps.
type CompensationExecutor struct {
	wal              WAL
	idempotencyStore IdempotencyStore
}

// NewCompensationExecutor creates a compensation executor.
func NewCompensationExecutor(wal WAL, store IdempotencyStore) *CompensationExecutor {
	if store == nil {
		store = NewInMemoryIdempotencyStore()
	}
	return &CompensationExecutor{
		wal:              wal,
		idempotencyStore: store,
	}
}

// Execute runs compensation in reverse topological layers.
func (e *CompensationExecutor) Execute(
	ctx context.Context,
	definition *SagaDefinition,
	instance *SagaInstance,
	input any,
	cause error,
) error {
	if definition == nil {
		return fmt.Errorf("saga definition cannot be nil")
	}
	if instance == nil {
		return fmt.Errorf("saga instance cannot be nil")
	}

	layers, err := definition.TopologicalLayers()
	if err != nil {
		return err
	}

	completed := make(map[string]struct{}, len(instance.CompletedSteps))
	for _, stepID := range instance.CompletedSteps {
		completed[stepID] = struct{}{}
	}

	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]

		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error

		for _, stepID := range layer {
			if _, ok := completed[stepID]; !ok {
				continue
			}
			step := definition.Steps[stepID]
			if step == nil || step.Compensation == nil {
				continue
			}
			if step.CompensationPolicy == SkipCompensate {
				continue
			}

			wg.Add(1)
			go func(step *Step) {
				defer wg.Done()
				if err := e.executeStepCompensation(ctx, definition, instance, step, input, cause); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}(step)
		}

		wg.Wait()
		if firstErr != nil {
			return firstErr
		}
	}

	return nil
}

func (e *CompensationExecutor) executeStepCompensation(
	ctx context.Context,
	definition *SagaDefinition,
	instance *SagaInstance,
	step *Step,
	input any,
	cause error,
) error {
	stepID := step.ID
	key := CompensationIdempotencyKey(instance.ID, stepID)
	if e.idempotencyStore.Seen(key) {
		return nil
	}

	retryCfg := definition.Retry
	maxRetries := retryCfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := e.writeWAL(ctx, WALEntry{
			SagaID: instance.ID,
			StepID: stepID,
			Type:   WALEntryTypeCompensationStarted,
		}); err != nil {
			return err
		}

		stepCtx, cancel := context.WithCancel(ctx)
		if timeout := step.Timeout; timeout > 0 {
			stepCtx, cancel = context.WithTimeout(ctx, timeout)
		} else if definition.DefaultStepTimeout > 0 {
			stepCtx, cancel = context.WithTimeout(ctx, definition.DefaultStepTimeout)
		}

		err := step.Compensation(stepCtx, &CompensationContext{
			SagaID:     instance.ID,
			StepID:     stepID,
			FailedStep: instance.FailedStep,
			Failure:    cause,
			Input:      input,
			Result:     instance.StepResults[stepID],
			AllResults: copyResultMap(instance.StepResults),
		})
		cancel()
		if err == nil {
			e.idempotencyStore.Mark(key)
			instance.MarkStepCompensated(stepID)
			if walErr := e.writeWAL(ctx, WALEntry{
				SagaID: instance.ID,
				StepID: stepID,
				Type:   WALEntryTypeCompensationCompleted,
			}); walErr != nil {
				return walErr
			}
			return nil
		}

		if walErr := e.writeWAL(ctx, WALEntry{
			SagaID: instance.ID,
			StepID: stepID,
			Type:   WALEntryTypeCompensationFailed,
			Data:   []byte(err.Error()),
		}); walErr != nil {
			return walErr
		}

		if attempt == maxRetries {
			return fmt.Errorf("compensation failed for step %s after %d attempts: %w", stepID, maxRetries+1, err)
		}

		time.Sleep(backoffForAttempt(retryCfg, attempt))
	}

	return nil
}

func (e *CompensationExecutor) writeWAL(ctx context.Context, entry WALEntry) error {
	if e.wal == nil {
		return nil
	}
	_, err := e.wal.Append(ctx, entry)
	return err
}

func backoffForAttempt(cfg CompensationRetryConfig, attempt int) time.Duration {
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 100 * time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 5 * time.Second
	}
	if cfg.BackoffFactor < 1 {
		cfg.BackoffFactor = 2.0
	}

	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffFactor, float64(attempt))
	duration := time.Duration(backoff)
	if duration > cfg.MaxBackoff {
		return cfg.MaxBackoff
	}
	return duration
}

// CompensationIdempotencyKey builds an idempotency key for one compensation operation.
func CompensationIdempotencyKey(sagaID, stepID string) string {
	return fmt.Sprintf("%s:%s", sagaID, stepID)
}
