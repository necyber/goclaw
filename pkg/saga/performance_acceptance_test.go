package saga

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSagaSupports100ConcurrentExecutions(t *testing.T) {
	def, err := New("concurrency-100").
		Step("a", Action(func(context.Context, *StepContext) (any, error) {
			time.Sleep(2 * time.Millisecond)
			return "ok", nil
		})).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator(WithMaxConcurrentSagas(100))

	const workers = 100
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			sagaID := fmt.Sprintf("concurrent-%d", id)
			instance, execErr := orchestrator.ExecuteWithID(ctx, sagaID, def, nil)
			if execErr != nil {
				errCh <- execErr
				return
			}
			if instance == nil || instance.State != SagaStateCompleted {
				errCh <- fmt.Errorf("unexpected saga state for %s: %v", sagaID, instance)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent execution failed: %v", err)
		}
	}
}

func TestRecoveryScan1000SagasUnderOneSecond(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	checkpointStore, err := NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewBadgerCheckpointStore() error = %v", err)
	}

	const sagaCount = 1000
	for i := 0; i < sagaCount; i++ {
		cp := &Checkpoint{
			DefinitionName: "missing-definition",
			SagaID:         fmt.Sprintf("scan-%d", i),
			State:          SagaStateRunning,
			LastUpdated:    time.Now().UTC(),
		}
		if saveErr := checkpointStore.Save(context.Background(), cp); saveErr != nil {
			t.Fatalf("Save() checkpoint error = %v", saveErr)
		}
	}

	orchestrator := NewSagaOrchestrator()
	manager, err := NewRecoveryManager(orchestrator, checkpointStore, nil)
	if err != nil {
		t.Fatalf("NewRecoveryManager() error = %v", err)
	}

	start := time.Now()
	recovered, recoverErr := manager.Recover(context.Background(), map[string]*SagaDefinition{}, nil)
	elapsed := time.Since(start)

	if recoverErr != nil {
		t.Fatalf("Recover() error = %v", recoverErr)
	}
	if recovered != 0 {
		t.Fatalf("expected 0 recovered sagas for missing definitions, got %d", recovered)
	}
	if elapsed >= time.Second {
		t.Fatalf("recovery scan elapsed = %s, want < 1s", elapsed)
	}
}
