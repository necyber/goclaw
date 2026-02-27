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
