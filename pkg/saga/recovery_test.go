package saga

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type testRecoveryLogger struct {
	infos int32
	warns int32
}

func (l *testRecoveryLogger) Info(string, ...any) {
	atomic.AddInt32(&l.infos, 1)
}

func (l *testRecoveryLogger) Warn(string, ...any) {
	atomic.AddInt32(&l.warns, 1)
}

func TestRecoveryManagerRecoverRunningSagaFromCheckpoint(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	checkpointStore, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}

	var stepACount int32
	var stepBCount int32
	def, err := New("recover-running").
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			atomic.AddInt32(&stepACount, 1)
			return "a", nil
		})).
		Step("b", Action(func(context.Context, *StepContext) (any, error) {
			atomic.AddInt32(&stepBCount, 1)
			return "b", nil
		}), DependsOn("a")).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	cp := &Checkpoint{
		DefinitionName: def.Name,
		SagaID:         "recover-1",
		State:          SagaStateRunning,
		CompletedSteps: []string{"a"},
		StepResults:    map[string]any{"a": "a"},
		LastUpdated:    time.Now().UTC(),
	}
	if err := checkpointStore.Save(context.Background(), cp); err != nil {
		t.Fatalf("Save() checkpoint error = %v", err)
	}

	metrics := newCaptureSagaMetrics()
	orchestrator := NewSagaOrchestrator(WithMetrics(metrics))
	logger := &testRecoveryLogger{}
	manager, err := NewRecoveryManager(orchestrator, checkpointStore, logger)
	if err != nil {
		t.Fatalf("NewRecoveryManager() error = %v", err)
	}

	recovered, err := manager.Recover(context.Background(), map[string]*SagaDefinition{
		def.Name: def,
	}, nil)
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected 1 recovered saga, got %d", recovered)
	}
	if atomic.LoadInt32(&stepACount) != 0 {
		t.Fatalf("step a should be skipped during recovery, got count %d", stepACount)
	}
	if atomic.LoadInt32(&stepBCount) != 1 {
		t.Fatalf("step b should run once during recovery, got count %d", stepBCount)
	}

	instance, err := orchestrator.GetInstance("recover-1")
	if err != nil {
		t.Fatalf("GetInstance() error = %v", err)
	}
	if instance.State != SagaStateCompleted {
		t.Fatalf("expected completed state, got %s", instance.State)
	}
	if atomic.LoadInt32(&logger.infos) == 0 {
		t.Fatal("expected recovery logs to be emitted")
	}
	if metrics.recovery["success"] != 1 {
		t.Fatalf("expected one successful recovery metric, got %d", metrics.recovery["success"])
	}
}

func TestRecoveryManagerRecoverCompensatingSaga(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	checkpointStore, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}

	var compensatedCount int32
	def, err := New("recover-compensating").
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "a", nil }),
			Compensate(func(context.Context, *CompensationContext) error {
				atomic.AddInt32(&compensatedCount, 1)
				return nil
			}),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	cp := &Checkpoint{
		DefinitionName: def.Name,
		SagaID:         "recover-2",
		State:          SagaStateCompensating,
		CompletedSteps: []string{"a"},
		FailedStep:     "a",
		StepResults:    map[string]any{"a": "a"},
		LastUpdated:    time.Now().UTC(),
	}
	if err := checkpointStore.Save(context.Background(), cp); err != nil {
		t.Fatalf("Save() checkpoint error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	manager, err := NewRecoveryManager(orchestrator, checkpointStore, &testRecoveryLogger{})
	if err != nil {
		t.Fatalf("NewRecoveryManager() error = %v", err)
	}

	recovered, err := manager.Recover(context.Background(), map[string]*SagaDefinition{
		def.Name: def,
	}, nil)
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected 1 recovered saga, got %d", recovered)
	}
	if atomic.LoadInt32(&compensatedCount) != 1 {
		t.Fatalf("expected compensation to run once, got %d", compensatedCount)
	}
}

func TestRecoveryIdempotencyMultipleAttempts(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	checkpointStore, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}

	var count int32
	def, err := New("recover-idempotent").
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			atomic.AddInt32(&count, 1)
			return "a", nil
		})).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	cp := &Checkpoint{
		DefinitionName: def.Name,
		SagaID:         "recover-3",
		State:          SagaStateRunning,
		CompletedSteps: []string{},
		LastUpdated:    time.Now().UTC(),
	}
	if err := checkpointStore.Save(context.Background(), cp); err != nil {
		t.Fatalf("Save() checkpoint error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	manager, err := NewRecoveryManager(orchestrator, checkpointStore, &testRecoveryLogger{})
	if err != nil {
		t.Fatalf("NewRecoveryManager() error = %v", err)
	}

	recovered, err := manager.Recover(context.Background(), map[string]*SagaDefinition{def.Name: def}, nil)
	if err != nil {
		t.Fatalf("Recover() first error = %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected first recover count 1, got %d", recovered)
	}

	recovered, err = manager.Recover(context.Background(), map[string]*SagaDefinition{def.Name: def}, nil)
	if err != nil {
		t.Fatalf("Recover() second error = %v", err)
	}
	if recovered != 0 {
		t.Fatalf("expected second recover count 0 after terminal snapshot, got %d", recovered)
	}
	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("expected step execution once across retries, got %d", count)
	}
}

func TestCleanupManagerRunOnce(t *testing.T) {
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

	old := time.Now().UTC().Add(-48 * time.Hour)
	newer := time.Now().UTC()
	_, _ = wal.Append(context.Background(), WALEntry{SagaID: "terminal", Type: WALEntryTypeStepCompleted, Timestamp: old})
	_, _ = wal.Append(context.Background(), WALEntry{SagaID: "terminal", Type: WALEntryTypeStepCompleted, Timestamp: newer})
	_, _ = wal.Append(context.Background(), WALEntry{SagaID: "running", Type: WALEntryTypeStepCompleted, Timestamp: old})

	_ = checkpointStore.Save(context.Background(), &Checkpoint{
		SagaID:      "terminal",
		State:       SagaStateCompleted,
		LastUpdated: time.Now().UTC(),
	})
	_ = checkpointStore.Save(context.Background(), &Checkpoint{
		SagaID:      "running",
		State:       SagaStateRunning,
		LastUpdated: time.Now().UTC(),
	})

	cleaner := NewCleanupManager(
		wal,
		checkpointStore,
		func(sagaID string) bool { return sagaID == "terminal" },
		&testRecoveryLogger{},
	)

	deleted, err := cleaner.RunOnce(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one deleted wal entry, got %d", deleted)
	}

	terminalEntries, _ := wal.List(context.Background(), "terminal")
	if len(terminalEntries) != 1 {
		t.Fatalf("expected one retained terminal wal entry, got %d", len(terminalEntries))
	}
	runningEntries, _ := wal.List(context.Background(), "running")
	if len(runningEntries) != 1 {
		t.Fatalf("expected running saga wal untouched, got %d", len(runningEntries))
	}

	if _, err := checkpointStore.Load(context.Background(), "terminal"); err == nil {
		t.Fatal("expected terminal checkpoint to be cleaned")
	}
	if _, err := checkpointStore.Load(context.Background(), "running"); err != nil {
		t.Fatalf("running checkpoint should remain: %v", err)
	}
}

func TestCleanupManagerStartBackground(t *testing.T) {
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

	_, _ = wal.Append(context.Background(), WALEntry{
		SagaID:    "terminal-bg",
		Type:      WALEntryTypeStepCompleted,
		Timestamp: time.Now().UTC().Add(-48 * time.Hour),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cleaner := NewCleanupManager(
		wal,
		checkpointStore,
		func(string) bool { return true },
		&testRecoveryLogger{},
	)
	if err := cleaner.Start(ctx, 20*time.Millisecond, 24*time.Hour); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		entries, _ := wal.List(context.Background(), "terminal-bg")
		if len(entries) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("background cleanup did not delete expired wal entry, remaining=%d", len(entries))
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
}

func TestRecoveryManagerMissingDefinition(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	checkpointStore, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}
	_ = checkpointStore.Save(context.Background(), &Checkpoint{
		DefinitionName: "missing",
		SagaID:         "recover-missing",
		State:          SagaStateRunning,
		LastUpdated:    time.Now().UTC(),
	})

	logger := &testRecoveryLogger{}
	metrics := newCaptureSagaMetrics()
	manager, err := NewRecoveryManager(NewSagaOrchestrator(WithMetrics(metrics)), checkpointStore, logger)
	if err != nil {
		t.Fatalf("NewRecoveryManager() error = %v", err)
	}

	recovered, err := manager.Recover(context.Background(), map[string]*SagaDefinition{}, nil)
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if recovered != 0 {
		t.Fatalf("expected 0 recovered with missing definition, got %d", recovered)
	}
	if atomic.LoadInt32(&logger.warns) == 0 {
		t.Fatal("expected warning logs for missing definition")
	}
	if metrics.recovery["skipped"] != 1 {
		t.Fatalf("expected one skipped recovery metric, got %d", metrics.recovery["skipped"])
	}
}

func TestRecoveryManagerPropagatesResumeError(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	checkpointStore, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}

	def, err := New("recover-error").
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			return nil, errors.New("resume failed")
		})).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	_ = checkpointStore.Save(context.Background(), &Checkpoint{
		DefinitionName: def.Name,
		SagaID:         "recover-error",
		State:          SagaStateRunning,
		LastUpdated:    time.Now().UTC(),
	})

	metrics := newCaptureSagaMetrics()
	manager, err := NewRecoveryManager(NewSagaOrchestrator(WithMetrics(metrics)), checkpointStore, &testRecoveryLogger{})
	if err != nil {
		t.Fatalf("NewRecoveryManager() error = %v", err)
	}

	recovered, recoverErr := manager.Recover(context.Background(), map[string]*SagaDefinition{
		def.Name: def,
	}, nil)
	if recoverErr == nil {
		t.Fatal("expected recover error")
	}
	if recovered != 0 {
		t.Fatalf("expected no recovered sagas, got %d", recovered)
	}
	if metrics.recovery["failed"] != 1 {
		t.Fatalf("expected one failed recovery metric, got %d", metrics.recovery["failed"])
	}
}
