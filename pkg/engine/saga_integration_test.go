package engine

import (
	"context"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/saga"
	"github.com/goclaw/goclaw/pkg/storage/memory"
)

func TestEngineSagaOrchestratorDisabledByDefault(t *testing.T) {
	cfg := minConfig()
	cfg.Saga.Enabled = false

	eng, err := New(cfg, nil, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if eng.GetSagaOrchestrator() != nil {
		t.Fatal("expected nil saga orchestrator when saga is disabled")
	}
}

func TestEngineSagaOrchestratorLifecycle(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Storage.Type = "memory"
	cfg.Storage.Badger.Path = t.TempDir()
	cfg.Saga.Enabled = true
	cfg.Saga.WALCleanupInterval = 10 * time.Millisecond
	cfg.Saga.WALRetention = 24 * time.Hour

	eng, err := New(cfg, nil, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if eng.GetSagaOrchestrator() == nil {
		t.Fatal("expected saga orchestrator to be initialized")
	}

	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if eng.sagaCleanupCancel == nil {
		t.Fatal("expected saga cleanup routine to be started")
	}

	definition, err := saga.New("engine-saga").
		Step("a", saga.Action(func(context.Context, *saga.StepContext) (any, error) {
			return "ok", nil
		})).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	instance, err := eng.GetSagaOrchestrator().Execute(ctx, definition, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if instance.State != saga.SagaStateCompleted {
		t.Fatalf("expected completed saga, got %s", instance.State)
	}

	if err := eng.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if eng.sagaCleanupCancel != nil {
		t.Fatal("expected cleanup cancel to be cleared on stop")
	}
}

func TestEngineStartRecoversSagaFromPersistedDefinition(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Storage.Type = "memory"
	cfg.Storage.Badger.Path = t.TempDir()
	cfg.Saga.Enabled = true
	cfg.Saga.WALCleanupInterval = 10 * time.Millisecond
	cfg.Saga.WALRetention = 24 * time.Hour

	eng, err := New(cfg, nil, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if eng.GetSagaDefinitionStore() == nil {
		t.Fatal("expected saga definition store to be initialized")
	}
	if eng.GetSagaCheckpointStore() == nil {
		t.Fatal("expected saga checkpoint store to be initialized")
	}

	const sagaID = "startup-recover-1"
	snapshot := &saga.DefinitionSnapshot{
		Name:  "startup-recover",
		Steps: []saga.DefinitionStepSnapshot{{ID: "a"}},
	}
	if err := eng.GetSagaDefinitionStore().Save(context.Background(), sagaID, snapshot); err != nil {
		t.Fatalf("Save() definition snapshot error = %v", err)
	}
	if err := eng.GetSagaCheckpointStore().Save(context.Background(), &saga.Checkpoint{
		DefinitionName: snapshot.Name,
		SagaID:         sagaID,
		State:          saga.SagaStateRunning,
		LastUpdated:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Save() checkpoint error = %v", err)
	}

	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = eng.Stop(context.Background()) }()

	deadline := time.Now().Add(2 * time.Second)
	for {
		instance, getErr := eng.GetSagaOrchestrator().GetInstance(sagaID)
		if getErr == nil && instance.State == saga.SagaStateCompleted {
			return
		}
		if time.Now().After(deadline) {
			if getErr != nil {
				t.Fatalf("expected recovered saga instance, last GetInstance() error = %v", getErr)
			}
			t.Fatalf("expected recovered saga completed state, got %s", instance.State)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
