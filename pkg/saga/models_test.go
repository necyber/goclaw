package saga

import (
	"context"
	"errors"
	"testing"
	"time"
)

func noopAction(context.Context, *StepContext) (any, error) {
	return "ok", nil
}

func noopCompensation(context.Context, *CompensationContext) error {
	return nil
}

func TestBuilderBuildSuccess(t *testing.T) {
	def, err := New("order-processing").
		WithTimeout(2*time.Minute).
		WithDefaultStepTimeout(10*time.Second).
		WithCompensationPolicy(AutoCompensate).
		Step("reserve",
			Action(noopAction),
			Compensate(noopCompensation),
		).
		Step("charge",
			Action(noopAction),
			Compensate(noopCompensation),
			DependsOn("reserve"),
		).
		Step("ship",
			Action(noopAction),
			DependsOn("charge"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if def.Name != "order-processing" {
		t.Fatalf("unexpected saga name: %s", def.Name)
	}
	if len(def.Steps) != 3 {
		t.Fatalf("unexpected step count: %d", len(def.Steps))
	}
	if def.Steps["ship"].Compensation != nil {
		t.Fatal("ship compensation should be nil")
	}
}

func TestBuilderValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		build func() (*SagaDefinition, error)
	}{
		{
			name: "duplicate step id",
			build: func() (*SagaDefinition, error) {
				return New("dup").
					Step("a", Action(noopAction)).
					Step("a", Action(noopAction)).
					Build()
			},
		},
		{
			name: "missing action",
			build: func() (*SagaDefinition, error) {
				return New("missing-action").
					Step("a").
					Build()
			},
		},
		{
			name: "unknown dependency",
			build: func() (*SagaDefinition, error) {
				return New("unknown-dep").
					Step("a", Action(noopAction), DependsOn("b")).
					Build()
			},
		},
		{
			name: "cycle",
			build: func() (*SagaDefinition, error) {
				return New("cycle").
					Step("a", Action(noopAction), DependsOn("b")).
					Step("b", Action(noopAction), DependsOn("a")).
					Build()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.build(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestTopologicalLayersParallel(t *testing.T) {
	def, err := New("parallel").
		Step("a", Action(noopAction)).
		Step("b", Action(noopAction), DependsOn("a")).
		Step("c", Action(noopAction), DependsOn("a")).
		Step("d", Action(noopAction), DependsOn("b", "c")).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	layers, err := def.TopologicalLayers()
	if err != nil {
		t.Fatalf("TopologicalLayers() unexpected error: %v", err)
	}

	if len(layers) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(layers))
	}
	if len(layers[0]) != 1 || layers[0][0] != "a" {
		t.Fatalf("unexpected first layer: %#v", layers[0])
	}
	if len(layers[1]) != 2 {
		t.Fatalf("unexpected second layer size: %#v", layers[1])
	}
	if len(layers[2]) != 1 || layers[2][0] != "d" {
		t.Fatalf("unexpected third layer: %#v", layers[2])
	}
}

func TestSagaStateTransitions(t *testing.T) {
	cases := []struct {
		name      string
		current   SagaState
		next      SagaState
		expectErr bool
	}{
		{
			name:      "created to running",
			current:   SagaStateCreated,
			next:      SagaStateRunning,
			expectErr: false,
		},
		{
			name:      "running to completed",
			current:   SagaStateRunning,
			next:      SagaStateCompleted,
			expectErr: false,
		},
		{
			name:      "running to pending compensation",
			current:   SagaStateRunning,
			next:      SagaStatePendingCompensation,
			expectErr: false,
		},
		{
			name:      "completed to running invalid",
			current:   SagaStateCompleted,
			next:      SagaStateRunning,
			expectErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTransition(tc.current, tc.next)
			if (err != nil) != tc.expectErr {
				t.Fatalf("expected err=%v, got %v", tc.expectErr, err)
			}
		})
	}
}

func TestSagaInstanceLifecycle(t *testing.T) {
	def, err := New("inst").
		Step("a", Action(noopAction)).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	inst := NewSagaInstance("s-1", def)
	if inst.State != SagaStateCreated {
		t.Fatalf("expected created state, got %s", inst.State)
	}

	if err := inst.TransitionTo(SagaStateRunning); err != nil {
		t.Fatalf("running transition failed: %v", err)
	}
	if inst.StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}

	inst.MarkStepCompleted("a", "done")
	if len(inst.CompletedSteps) != 1 {
		t.Fatalf("expected 1 completed step, got %d", len(inst.CompletedSteps))
	}
	if got := inst.StepResults["a"]; got != "done" {
		t.Fatalf("unexpected step result: %v", got)
	}

	inst.SetFailure("a", errors.New("boom"))
	if inst.FailedStep != "a" || inst.FailureReason == "" {
		t.Fatal("expected failure details to be recorded")
	}

	if err := inst.TransitionTo(SagaStateCompensating); err != nil {
		t.Fatalf("compensating transition failed: %v", err)
	}
	inst.MarkStepCompensated("a")
	if len(inst.Compensated) != 1 {
		t.Fatalf("expected 1 compensated step, got %d", len(inst.Compensated))
	}

	if err := inst.TransitionTo(SagaStateCompensated); err != nil {
		t.Fatalf("compensated transition failed: %v", err)
	}
	if inst.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set for terminal state")
	}
}
