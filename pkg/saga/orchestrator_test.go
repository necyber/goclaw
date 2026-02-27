package saga

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestSagaOrchestratorExecuteLinearWithResultPassing(t *testing.T) {
	def, err := New("linear").
		Step("a", Action(func(ctx context.Context, stepCtx *StepContext) (any, error) {
			return "token", nil
		})).
		Step("b", Action(func(ctx context.Context, stepCtx *StepContext) (any, error) {
			if stepCtx.Results["a"] != "token" {
				t.Fatalf("expected result from step a, got %#v", stepCtx.Results["a"])
			}
			return "done", nil
		}), DependsOn("a")).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.Execute(context.Background(), def, map[string]any{"request": "x"})
	if execErr != nil {
		t.Fatalf("Execute() error = %v", execErr)
	}
	if instance.State != SagaStateCompleted {
		t.Fatalf("expected completed state, got %s", instance.State)
	}
	if len(instance.CompletedSteps) != 2 {
		t.Fatalf("expected 2 completed steps, got %d", len(instance.CompletedSteps))
	}
}

func TestSagaOrchestratorExecuteParallelSteps(t *testing.T) {
	def, err := New("parallel").
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			return "a", nil
		})).
		Step("b", Action(func(context.Context, *StepContext) (any, error) {
			time.Sleep(10 * time.Millisecond)
			return "b", nil
		}), DependsOn("a")).
		Step("c", Action(func(context.Context, *StepContext) (any, error) {
			time.Sleep(10 * time.Millisecond)
			return "c", nil
		}), DependsOn("a")).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.Execute(context.Background(), def, nil)
	if execErr != nil {
		t.Fatalf("Execute() error = %v", execErr)
	}
	if instance.State != SagaStateCompleted {
		t.Fatalf("expected completed state, got %s", instance.State)
	}
	if len(instance.CompletedSteps) != 3 {
		t.Fatalf("expected 3 completed steps, got %d", len(instance.CompletedSteps))
	}
}

func TestSagaOrchestratorExecuteMixedGraph(t *testing.T) {
	def, err := New("mixed").
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			return "a", nil
		})).
		Step("b", Action(func(context.Context, *StepContext) (any, error) {
			return "b", nil
		}), DependsOn("a")).
		Step("c", Action(func(context.Context, *StepContext) (any, error) {
			return "c", nil
		}), DependsOn("a")).
		Step("d", Action(func(ctx context.Context, stepCtx *StepContext) (any, error) {
			if stepCtx.Results["b"] != "b" || stepCtx.Results["c"] != "c" {
				t.Fatalf("expected mixed dependencies results, got b=%v c=%v", stepCtx.Results["b"], stepCtx.Results["c"])
			}
			return "d", nil
		}), DependsOn("b", "c")).
		Step("e", Action(func(ctx context.Context, stepCtx *StepContext) (any, error) {
			if stepCtx.Results["d"] != "d" {
				t.Fatalf("expected result from step d, got %#v", stepCtx.Results["d"])
			}
			return "e", nil
		}), DependsOn("d")).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.Execute(context.Background(), def, nil)
	if execErr != nil {
		t.Fatalf("Execute() error = %v", execErr)
	}
	if instance.State != SagaStateCompleted {
		t.Fatalf("expected completed state, got %s", instance.State)
	}
	if len(instance.CompletedSteps) != 5 {
		t.Fatalf("expected 5 completed steps, got %d", len(instance.CompletedSteps))
	}
}

func TestSagaOrchestratorStepTimeoutTriggersCompensation(t *testing.T) {
	var compensated bool

	def, err := New("timeout").
		WithDefaultStepTimeout(20*time.Millisecond).
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "a", nil }),
			Compensate(func(context.Context, *CompensationContext) error {
				compensated = true
				return nil
			}),
		).
		Step("b",
			Action(func(ctx context.Context, stepCtx *StepContext) (any, error) {
				time.Sleep(60 * time.Millisecond)
				return "b", nil
			}),
			DependsOn("a"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.Execute(context.Background(), def, nil)
	if execErr == nil {
		t.Fatal("expected execute error due to step timeout")
	}
	if instance.State != SagaStateCompensated {
		t.Fatalf("expected compensated state, got %s", instance.State)
	}
	if !compensated {
		t.Fatal("expected compensation to execute")
	}
}

func TestSagaOrchestratorSagaTimeout(t *testing.T) {
	def, err := New("saga-timeout").
		WithTimeout(30*time.Millisecond).
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			time.Sleep(100 * time.Millisecond)
			return "a", nil
		})).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.Execute(context.Background(), def, nil)
	if execErr == nil {
		t.Fatal("expected saga timeout error")
	}
	if instance.State != SagaStateCompensated {
		t.Fatalf("expected compensated state after timeout, got %s", instance.State)
	}
}

func TestSagaOrchestratorManualCompensation(t *testing.T) {
	var compensated bool

	def, err := New("manual").
		WithCompensationPolicy(ManualCompensate).
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "a", nil }),
			Compensate(func(context.Context, *CompensationContext) error {
				compensated = true
				return nil
			}),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("boom") }),
			DependsOn("a"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.ExecuteWithID(context.Background(), "manual-1", def, nil)
	if execErr == nil {
		t.Fatal("expected execution failure")
	}
	if instance.State != SagaStatePendingCompensation {
		t.Fatalf("expected pending-compensation state, got %s", instance.State)
	}

	instance, err = orchestrator.TriggerCompensation(context.Background(), "manual-1", def, nil, execErr)
	if err != nil {
		t.Fatalf("TriggerCompensation() error = %v", err)
	}
	if instance.State != SagaStateCompensated {
		t.Fatalf("expected compensated state, got %s", instance.State)
	}
	if !compensated {
		t.Fatal("expected manual compensation to execute")
	}
}

func TestSagaOrchestratorConcurrentLimit(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 1)

	def, err := New("limit").
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			started <- struct{}{}
			<-release
			return "ok", nil
		})).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator(WithMaxConcurrentSagas(1))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = orchestrator.ExecuteWithID(context.Background(), "first", def, nil)
	}()

	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, execErr := orchestrator.ExecuteWithID(ctx, "second", def, nil)
	if execErr == nil {
		t.Fatal("expected second execution to fail due to concurrent limit timeout")
	}

	close(release)
	wg.Wait()
}

func TestSagaOrchestratorRecordsSagaMetrics(t *testing.T) {
	def, err := New("metrics").
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			return "ok", nil
		})).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	metrics := newCaptureSagaMetrics()
	orchestrator := NewSagaOrchestrator(WithMetrics(metrics))

	instance, execErr := orchestrator.Execute(context.Background(), def, nil)
	if execErr != nil {
		t.Fatalf("Execute() error = %v", execErr)
	}
	if instance.State != SagaStateCompleted {
		t.Fatalf("expected completed state, got %s", instance.State)
	}
	if metrics.executionCount("completed") != 1 {
		t.Fatalf("expected one completed saga execution metric, got %d", metrics.executionCount("completed"))
	}
	if metrics.activeInc != 1 || metrics.activeDec != 1 {
		t.Fatalf("expected active inc/dec to be 1/1, got %d/%d", metrics.activeInc, metrics.activeDec)
	}
}

type captureSagaMetrics struct {
	mu            sync.Mutex
	executions    map[string]int
	activeInc     int
	activeDec     int
	compensations map[string]int
	retries       int
	recovery      map[string]int
}

func newCaptureSagaMetrics() *captureSagaMetrics {
	return &captureSagaMetrics{
		executions:    make(map[string]int),
		compensations: make(map[string]int),
		recovery:      make(map[string]int),
	}
}

func (m *captureSagaMetrics) RecordSagaExecution(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executions[status]++
}

func (m *captureSagaMetrics) RecordSagaDuration(status string, duration time.Duration) {
	_ = status
	_ = duration
}

func (m *captureSagaMetrics) IncActiveSagas() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeInc++
}

func (m *captureSagaMetrics) DecActiveSagas() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeDec++
}

func (m *captureSagaMetrics) RecordCompensation(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.compensations[status]++
}

func (m *captureSagaMetrics) RecordCompensationDuration(duration time.Duration) { _ = duration }

func (m *captureSagaMetrics) RecordCompensationRetry() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retries++
}

func (m *captureSagaMetrics) RecordSagaRecovery(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recovery[status]++
}

func (m *captureSagaMetrics) executionCount(status string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executions[status]
}
