package saga

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCompensationOrderLinearAndParallel(t *testing.T) {
	var mu sync.Mutex
	order := make([]string, 0)
	record := func(id string) {
		mu.Lock()
		order = append(order, id)
		mu.Unlock()
	}

	def, err := New("comp-order").
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "a", nil }),
			Compensate(func(context.Context, *CompensationContext) error { record("a"); return nil }),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return "b", nil }),
			Compensate(func(context.Context, *CompensationContext) error { record("b"); return nil }),
			DependsOn("a"),
		).
		Step("c",
			Action(func(context.Context, *StepContext) (any, error) { return "c", nil }),
			Compensate(func(context.Context, *CompensationContext) error { record("c"); return nil }),
			DependsOn("a"),
		).
		Step("d",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("fail-d") }),
			DependsOn("b", "c"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.Execute(context.Background(), def, "input")
	if execErr == nil {
		t.Fatal("expected execution error")
	}
	if instance.State != SagaStateCompensated {
		t.Fatalf("expected compensated state, got %s", instance.State)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 compensation operations, got %d (%v)", len(order), order)
	}

	// b and c can be parallel, but a must be compensated last.
	last := order[len(order)-1]
	if last != "a" {
		t.Fatalf("expected a to be compensated last, got order %v", order)
	}
}

func TestCompensationRetry(t *testing.T) {
	attempts := 0
	def, err := New("comp-retry").
		WithRetryConfig(CompensationRetryConfig{
			MaxRetries:     3,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     5 * time.Millisecond,
			BackoffFactor:  2.0,
		}).
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "a", nil }),
			Compensate(func(context.Context, *CompensationContext) error {
				attempts++
				if attempts < 3 {
					return errors.New("transient")
				}
				return nil
			}),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("fail-b") }),
			DependsOn("a"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.Execute(context.Background(), def, nil)
	if execErr == nil {
		t.Fatal("expected execution error")
	}
	if instance.State != SagaStateCompensated {
		t.Fatalf("expected compensated state, got %s", instance.State)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 compensation attempts, got %d", attempts)
	}
}

func TestCompensationTimeoutAndFailureTransition(t *testing.T) {
	def, err := New("comp-timeout").
		WithDefaultStepTimeout(10*time.Millisecond).
		WithRetryConfig(CompensationRetryConfig{
			MaxRetries:     1,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     2 * time.Millisecond,
			BackoffFactor:  2.0,
		}).
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "a", nil }),
			Compensate(func(ctx context.Context, c *CompensationContext) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(50 * time.Millisecond):
					return nil
				}
			}),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("fail-b") }),
			DependsOn("a"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.Execute(context.Background(), def, nil)
	if execErr == nil {
		t.Fatal("expected execution error")
	}
	if instance.State != SagaStateCompensationFailed {
		t.Fatalf("expected compensation-failed state, got %s", instance.State)
	}
}

func TestCompensationContextInjection(t *testing.T) {
	var gotInput any
	var gotResult any
	var gotFailedStep string

	def, err := New("comp-context").
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "result-a", nil }),
			Compensate(func(context.Context, *CompensationContext) error { return nil }),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return "result-b", nil }),
			Compensate(func(ctx context.Context, c *CompensationContext) error {
				gotInput = c.Input
				gotResult = c.Result
				gotFailedStep = c.FailedStep
				return nil
			}),
			DependsOn("a"),
		).
		Step("c",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("fail-c") }),
			DependsOn("b"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	_, execErr := orchestrator.Execute(context.Background(), def, map[string]string{"request": "ctx"})
	if execErr == nil {
		t.Fatal("expected execution error")
	}
	if gotInput == nil || gotResult == nil {
		t.Fatalf("expected compensation context to include input/result, got input=%v result=%v", gotInput, gotResult)
	}
	if gotFailedStep != "c" {
		t.Fatalf("expected failed step c, got %s", gotFailedStep)
	}
}

func TestIdempotencyUtilities(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	key := CompensationIdempotencyKey("saga-1", "step-a")

	if store.Seen(key) {
		t.Fatal("expected key unseen before mark")
	}
	store.Mark(key)
	if !store.Seen(key) {
		t.Fatal("expected key seen after mark")
	}
}

func TestBackoffForAttempt(t *testing.T) {
	cfg := CompensationRetryConfig{
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     400 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	delays := []time.Duration{
		backoffForAttempt(cfg, 0),
		backoffForAttempt(cfg, 1),
		backoffForAttempt(cfg, 2),
		backoffForAttempt(cfg, 3),
	}

	if delays[0] != 100*time.Millisecond {
		t.Fatalf("unexpected delay 0: %v", delays[0])
	}
	if delays[1] != 200*time.Millisecond {
		t.Fatalf("unexpected delay 1: %v", delays[1])
	}
	if delays[2] != 400*time.Millisecond {
		t.Fatalf("unexpected delay 2: %v", delays[2])
	}
	if delays[3] != 400*time.Millisecond {
		t.Fatalf("delay should be capped by max backoff, got %v", delays[3])
	}
}
