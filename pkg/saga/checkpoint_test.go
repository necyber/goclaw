package saga

import (
	"context"
	"testing"
	"time"
)

func TestCheckpointSerializationRoundTrip(t *testing.T) {
	original := &Checkpoint{
		SagaID:         "saga-1",
		State:          SagaStateRunning,
		CompletedSteps: []string{"a", "b"},
		FailedStep:     "c",
		StepResults: map[string]any{
			"a": "ok",
			"b": map[string]any{"amount": 10},
		},
		LastUpdated: time.Now().UTC(),
	}

	data, err := SerializeCheckpoint(original)
	if err != nil {
		t.Fatalf("SerializeCheckpoint() error = %v", err)
	}

	decoded, err := DeserializeCheckpoint(data)
	if err != nil {
		t.Fatalf("DeserializeCheckpoint() error = %v", err)
	}

	if decoded.SagaID != original.SagaID {
		t.Fatalf("saga_id mismatch: got %s want %s", decoded.SagaID, original.SagaID)
	}
	if decoded.State != original.State {
		t.Fatalf("state mismatch: got %v want %v", decoded.State, original.State)
	}
	if len(decoded.CompletedSteps) != len(original.CompletedSteps) {
		t.Fatalf("completed steps mismatch: got %d want %d", len(decoded.CompletedSteps), len(original.CompletedSteps))
	}
}

func TestBadgerCheckpointStoreSaveLoadAndOverwrite(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}

	ctx := context.Background()
	first := &Checkpoint{
		SagaID:         "saga-2",
		State:          SagaStateRunning,
		CompletedSteps: []string{"a"},
		StepResults:    map[string]any{"a": "ok"},
	}
	if err := store.Save(ctx, first); err != nil {
		t.Fatalf("Save(first) error = %v", err)
	}

	second := &Checkpoint{
		SagaID:         "saga-2",
		State:          SagaStateCompensating,
		CompletedSteps: []string{"a", "b"},
		FailedStep:     "c",
		StepResults:    map[string]any{"a": "ok", "b": "ok"},
	}
	if err := store.Save(ctx, second); err != nil {
		t.Fatalf("Save(second) error = %v", err)
	}

	loaded, err := store.Load(ctx, "saga-2")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.State != SagaStateCompensating {
		t.Fatalf("expected overwritten state compensating, got %s", loaded.State)
	}
	if loaded.FailedStep != "c" {
		t.Fatalf("expected failed step c, got %s", loaded.FailedStep)
	}
	if len(loaded.CompletedSteps) != 2 {
		t.Fatalf("expected 2 completed steps, got %d", len(loaded.CompletedSteps))
	}
}

func TestCheckpointerRecordStepCompletion(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}
	checkpointer, err := NewCheckpointer(store)
	if err != nil {
		t.Fatalf("NewCheckpointer() error = %v", err)
	}

	inst := NewSagaInstance("saga-3", nil)
	inst.State = SagaStateRunning

	if err := checkpointer.RecordStepCompletion(context.Background(), inst, "reserve", map[string]any{"ok": true}); err != nil {
		t.Fatalf("RecordStepCompletion() error = %v", err)
	}

	loaded, err := store.Load(context.Background(), "saga-3")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.CompletedSteps) != 1 || loaded.CompletedSteps[0] != "reserve" {
		t.Fatalf("unexpected completed steps: %#v", loaded.CompletedSteps)
	}
	if loaded.State != SagaStateRunning {
		t.Fatalf("expected running state, got %s", loaded.State)
	}
}
