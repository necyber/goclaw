package saga

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSagaOrchestrator_WithWALAndCheckpoint(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	wal, err := NewBadgerWAL(db, WALOptions{WriteMode: WALWriteModeSync})
	if err != nil {
		t.Fatalf("NewBadgerWAL() error = %v", err)
	}
	checkpointStore, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}
	checkpointer, err := NewCheckpointer(checkpointStore)
	if err != nil {
		t.Fatalf("NewCheckpointer() error = %v", err)
	}

	def, err := New("integration").
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			return "a", nil
		})).
		Step("b", Action(func(context.Context, *StepContext) (any, error) {
			return "b", nil
		}), DependsOn("a")).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator(
		WithWAL(wal),
		WithCheckpointer(checkpointer),
	)

	instance, execErr := orchestrator.ExecuteWithID(context.Background(), "integration-1", def, nil)
	if execErr != nil {
		t.Fatalf("ExecuteWithID() error = %v", execErr)
	}
	if instance.State != SagaStateCompleted {
		t.Fatalf("expected completed state, got %s", instance.State)
	}

	entries, err := wal.List(context.Background(), "integration-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) < 4 {
		t.Fatalf("expected at least 4 wal entries, got %d", len(entries))
	}

	checkpoint, err := checkpointStore.Load(context.Background(), "integration-1")
	if err != nil {
		t.Fatalf("Load() checkpoint error = %v", err)
	}
	if len(checkpoint.CompletedSteps) != 2 {
		t.Fatalf("expected 2 completed steps in checkpoint, got %d", len(checkpoint.CompletedSteps))
	}
}

func TestSagaOrchestrator_WithWALAndCheckpointCompensation(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	wal, err := NewBadgerWAL(db, WALOptions{WriteMode: WALWriteModeSync})
	if err != nil {
		t.Fatalf("NewBadgerWAL() error = %v", err)
	}
	checkpointStore, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}
	checkpointer, err := NewCheckpointer(checkpointStore)
	if err != nil {
		t.Fatalf("NewCheckpointer() error = %v", err)
	}

	def, err := New("integration-compensate").
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "a", nil }),
			Compensate(func(context.Context, *CompensationContext) error { return nil }),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("boom") }),
			DependsOn("a"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator(
		WithWAL(wal),
		WithCheckpointer(checkpointer),
	)

	instance, execErr := orchestrator.ExecuteWithID(context.Background(), "integration-compensate-1", def, nil)
	if execErr == nil {
		t.Fatal("expected execute error")
	}
	if instance.State != SagaStateCompensated {
		t.Fatalf("expected compensated state, got %s", instance.State)
	}

	entries, err := wal.List(context.Background(), "integration-compensate-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	hasCompensation := false
	for _, entry := range entries {
		if entry.Type == WALEntryTypeCompensationCompleted {
			hasCompensation = true
			break
		}
	}
	if !hasCompensation {
		t.Fatal("expected compensation wal entries to be recorded")
	}

	checkpoint, err := checkpointStore.Load(context.Background(), "integration-compensate-1")
	if err != nil {
		t.Fatalf("Load() checkpoint error = %v", err)
	}
	if len(checkpoint.CompletedSteps) != 1 || checkpoint.CompletedSteps[0] != "a" {
		t.Fatalf("expected checkpoint to keep completed step a, got %#v", checkpoint.CompletedSteps)
	}
}

func TestSagaOrchestrator_PersistsCheckpointAcrossFailureTransitions(t *testing.T) {
	tests := []struct {
		name          string
		policy        CompensationPolicy
		expectedState SagaState
	}{
		{name: "manual", policy: ManualCompensate, expectedState: SagaStatePendingCompensation},
		{name: "skip", policy: SkipCompensate, expectedState: SagaStateCompensationFailed},
		{name: "auto", policy: AutoCompensate, expectedState: SagaStateCompensated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openTestBadger(t)
			t.Cleanup(func() { _ = db.Close() })

			checkpointStore, err := NewBadgerCheckpointStore(db)
			if err != nil {
				t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
			}
			checkpointer, err := NewCheckpointer(checkpointStore)
			if err != nil {
				t.Fatalf("NewCheckpointer() error = %v", err)
			}

			def, err := New("checkpoint-transition-"+tt.name).
				WithCompensationPolicy(tt.policy).
				Step("a",
					Action(func(context.Context, *StepContext) (any, error) { return "a", nil }),
					Compensate(func(context.Context, *CompensationContext) error { return nil }),
				).
				Step("b",
					Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("boom") }),
					DependsOn("a"),
				).
				Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}

			orchestrator := NewSagaOrchestrator(WithCheckpointer(checkpointer))
			const sagaID = "checkpoint-transition"

			instance, execErr := orchestrator.ExecuteWithID(context.Background(), sagaID, def, nil)
			if execErr == nil {
				t.Fatal("expected execute error")
			}
			if instance.State != tt.expectedState {
				t.Fatalf("expected state %s, got %s", tt.expectedState, instance.State)
			}

			checkpoint, err := checkpointStore.Load(context.Background(), sagaID)
			if err != nil {
				t.Fatalf("Load() checkpoint error = %v", err)
			}
			if checkpoint.State != tt.expectedState {
				t.Fatalf("expected checkpoint state %s, got %s", tt.expectedState, checkpoint.State)
			}
			if checkpoint.FailedStep != "b" {
				t.Fatalf("expected failed step b, got %q", checkpoint.FailedStep)
			}
			if len(checkpoint.CompletedSteps) != 1 || checkpoint.CompletedSteps[0] != "a" {
				t.Fatalf("expected completed step a in checkpoint, got %#v", checkpoint.CompletedSteps)
			}
		})
	}
}

func TestSagaOrchestratorResumeRunning_UsesCheckpointStateAndMaxConcurrent(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	checkpointStore, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}
	checkpointer, err := NewCheckpointer(checkpointStore)
	if err != nil {
		t.Fatalf("NewCheckpointer() error = %v", err)
	}

	var inFlight int32
	var maxInFlight int32

	def, err := New("resume-max-concurrent").
		WithMaxConcurrent(1).
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			return "seed", nil
		})).
		Step("b", Action(func(context.Context, *StepContext) (any, error) {
			n := atomic.AddInt32(&inFlight, 1)
			for {
				current := atomic.LoadInt32(&maxInFlight)
				if n <= current || atomic.CompareAndSwapInt32(&maxInFlight, current, n) {
					break
				}
			}
			defer atomic.AddInt32(&inFlight, -1)
			time.Sleep(20 * time.Millisecond)
			return "b", nil
		}), DependsOn("a")).
		Step("c", Action(func(_ context.Context, stepCtx *StepContext) (any, error) {
			if stepCtx.Results["a"] != "seed" {
				return nil, errors.New("missing resumed result for step a")
			}
			n := atomic.AddInt32(&inFlight, 1)
			for {
				current := atomic.LoadInt32(&maxInFlight)
				if n <= current || atomic.CompareAndSwapInt32(&maxInFlight, current, n) {
					break
				}
			}
			defer atomic.AddInt32(&inFlight, -1)
			time.Sleep(20 * time.Millisecond)
			return "c", nil
		}), DependsOn("a")).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	checkpoint := &Checkpoint{
		DefinitionName: def.Name,
		SagaID:         "resume-running-1",
		State:          SagaStateRunning,
		CompletedSteps: []string{"a"},
		StepResults:    map[string]any{"a": "seed"},
		LastUpdated:    time.Now().UTC(),
	}
	if err := checkpointStore.Save(context.Background(), checkpoint); err != nil {
		t.Fatalf("Save() checkpoint error = %v", err)
	}

	orchestrator := NewSagaOrchestrator(WithCheckpointer(checkpointer))
	instance, err := orchestrator.ResumeFromCheckpoint(context.Background(), def, checkpoint, nil)
	if err != nil {
		t.Fatalf("ResumeFromCheckpoint() error = %v", err)
	}
	if instance.State != SagaStateCompleted {
		t.Fatalf("expected completed state, got %s", instance.State)
	}
	if atomic.LoadInt32(&maxInFlight) > 1 {
		t.Fatalf("expected max in-flight step count <= 1, got %d", maxInFlight)
	}

	loaded, err := checkpointStore.Load(context.Background(), checkpoint.SagaID)
	if err != nil {
		t.Fatalf("Load() checkpoint error = %v", err)
	}
	if loaded.State != SagaStateCompleted {
		t.Fatalf("expected completed checkpoint state after resume, got %s", loaded.State)
	}
	if len(loaded.CompletedSteps) != 3 {
		t.Fatalf("expected 3 completed steps after resume, got %#v", loaded.CompletedSteps)
	}
	if loaded.StepResults["a"] != "seed" {
		t.Fatalf("expected resumed step result for a, got %#v", loaded.StepResults["a"])
	}
}
