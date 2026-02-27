package saga

import (
	"context"
	"testing"
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
