package saga

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBuilderOptionsCoverage(t *testing.T) {
	def, err := New("builder-options").
		WithMaxConcurrent(2).
		Step("a",
			Action(noopAction),
			StepTimeout(2*time.Second),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if def.MaxConcurrent != 2 {
		t.Fatalf("expected max concurrent 2, got %d", def.MaxConcurrent)
	}
	if def.Steps["a"].Timeout != 2*time.Second {
		t.Fatalf("expected step timeout 2s, got %v", def.Steps["a"].Timeout)
	}
}

func TestNopMetricsAndLoggerCoverage(t *testing.T) {
	metrics := &nopMetricsRecorder{}
	metrics.RecordSagaExecution("completed")
	metrics.RecordSagaDuration("completed", time.Second)
	metrics.IncActiveSagas()
	metrics.DecActiveSagas()
	metrics.RecordCompensation("success")
	metrics.RecordCompensationDuration(time.Second)
	metrics.RecordCompensationRetry()
	metrics.RecordSagaRecovery("success")

	logger := &nopRecoveryLogger{}
	logger.Info("test")
	logger.Warn("test")
}

func TestOpenBadgerWALCoverage(t *testing.T) {
	wal, err := OpenBadgerWAL(t.TempDir(), WALOptions{})
	if err != nil {
		t.Fatalf("OpenBadgerWAL() error = %v", err)
	}
	defer wal.Close()

	if _, err := wal.Append(context.Background(), WALEntry{
		SagaID: "open-badger",
		Type:   WALEntryTypeStepStarted,
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
}

func TestOrchestratorListAndGetFromStoreCoverage(t *testing.T) {
	store := NewMemorySagaStore()
	def, err := New("store-list").Step("a", Action(noopAction)).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	instance := NewSagaInstance("store-1", def)
	_ = instance.TransitionTo(SagaStateRunning)
	_ = instance.TransitionTo(SagaStateCompleted)
	if err := store.Save(context.Background(), instance); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator(WithSagaStore(store))
	got, err := orchestrator.GetInstance("store-1")
	if err != nil {
		t.Fatalf("GetInstance() error = %v", err)
	}
	if got.ID != "store-1" {
		t.Fatalf("expected store-1, got %s", got.ID)
	}

	items := orchestrator.ListInstances()
	if len(items) != 1 {
		t.Fatalf("expected one listed instance, got %d", len(items))
	}

	filtered, total, err := orchestrator.ListInstancesFiltered(context.Background(), SagaListFilter{
		State:  SagaStateCompleted.String(),
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListInstancesFiltered() error = %v", err)
	}
	if total != 1 || len(filtered) != 1 {
		t.Fatalf("expected one filtered instance, got total=%d len=%d", total, len(filtered))
	}
}

func TestOrchestratorListAndFilterInMemoryCoverage(t *testing.T) {
	def, err := New("in-memory-list").
		Step("a", Action(noopAction)).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	if _, execErr := orchestrator.ExecuteWithID(context.Background(), "in-memory-1", def, nil); execErr != nil {
		t.Fatalf("ExecuteWithID() error = %v", execErr)
	}

	items := orchestrator.ListInstances()
	if len(items) != 1 {
		t.Fatalf("expected one in-memory instance, got %d", len(items))
	}

	filtered, total, err := orchestrator.ListInstancesFiltered(context.Background(), SagaListFilter{
		State:  SagaStateCompleted.String(),
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListInstancesFiltered() error = %v", err)
	}
	if total != 1 || len(filtered) != 1 {
		t.Fatalf("expected one filtered in-memory instance, got total=%d len=%d", total, len(filtered))
	}
}

func TestOrchestratorValidationAndErrorPathsCoverage(t *testing.T) {
	def, err := New("manual-error-coverage").
		WithCompensationPolicy(ManualCompensate).
		Step("a",
			Action(noopAction),
			Compensate(noopCompensation),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("fail") }),
			DependsOn("a"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	instance, execErr := orchestrator.ExecuteWithID(context.Background(), "manual-error-coverage", def, nil)
	if execErr == nil {
		t.Fatal("expected execute error")
	}
	if instance.State != SagaStatePendingCompensation {
		t.Fatalf("expected pending-compensation, got %s", instance.State)
	}

	if _, err := orchestrator.TriggerCompensation(context.Background(), "manual-error-coverage", nil, nil, errors.New("x")); err == nil {
		t.Fatal("expected TriggerCompensation() error for nil definition")
	}

	completedDef, err := New("completed").Step("a", Action(noopAction)).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if _, err := orchestrator.ExecuteWithID(context.Background(), "completed-coverage", completedDef, nil); err != nil {
		t.Fatalf("ExecuteWithID() error = %v", err)
	}
	if _, err := orchestrator.TriggerCompensation(context.Background(), "completed-coverage", completedDef, nil, errors.New("x")); err == nil {
		t.Fatal("expected TriggerCompensation() error for invalid state")
	}

	if _, err := orchestrator.ResumeFromCheckpoint(context.Background(), nil, &Checkpoint{SagaID: "x"}, nil); err == nil {
		t.Fatal("expected ResumeFromCheckpoint() error for nil definition")
	}
	if _, err := orchestrator.ResumeFromCheckpoint(context.Background(), completedDef, nil, nil); err == nil {
		t.Fatal("expected ResumeFromCheckpoint() error for nil checkpoint")
	}
	if _, err := orchestrator.ResumeFromCheckpoint(context.Background(), completedDef, &Checkpoint{}, nil); err == nil {
		t.Fatal("expected ResumeFromCheckpoint() error for empty saga id")
	}
}

type recordingIdempotencyStore struct {
	seen  map[string]bool
	marks map[string]bool
}

func newRecordingIdempotencyStore() *recordingIdempotencyStore {
	return &recordingIdempotencyStore{
		seen:  make(map[string]bool),
		marks: make(map[string]bool),
	}
}

func (s *recordingIdempotencyStore) Seen(key string) bool {
	return s.seen[key]
}

func (s *recordingIdempotencyStore) Mark(key string) {
	s.seen[key] = true
	s.marks[key] = true
}

func TestWithIdempotencyStoreOptionCoverage(t *testing.T) {
	store := newRecordingIdempotencyStore()
	orchestrator := NewSagaOrchestrator(WithIdempotencyStore(store))

	def, err := New("idem-store").
		Step("a",
			Action(noopAction),
			Compensate(noopCompensation),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("fail") }),
			DependsOn("a"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if _, execErr := orchestrator.ExecuteWithID(context.Background(), "idem-store", def, nil); execErr == nil {
		t.Fatal("expected execute error")
	}

	if !store.marks[CompensationIdempotencyKey("idem-store", "a")] {
		t.Fatal("expected custom idempotency store to be used")
	}
}
