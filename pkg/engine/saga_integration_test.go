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
