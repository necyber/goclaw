package engine

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	config := Config{
		Name:       "test-engine",
		ConfigPath: "test-config.yaml",
	}

	eng := New(config)
	if eng == nil {
		t.Fatal("expected engine to be created, got nil")
	}

	if eng.config.Name != config.Name {
		t.Errorf("expected name %q, got %q", config.Name, eng.config.Name)
	}

	if eng.state != StateIdle {
		t.Errorf("expected initial state %v, got %v", StateIdle, eng.state)
	}
}

func TestEngine_Start(t *testing.T) {
	config := Config{Name: "test-engine"}
	eng := New(config)
	ctx := context.Background()

	// First start should succeed
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("unexpected error on first start: %v", err)
	}

	if eng.state != StateRunning {
		t.Errorf("expected state %v, got %v", StateRunning, eng.state)
	}

	// Second start should fail
	if err := eng.Start(ctx); err == nil {
		t.Error("expected error on second start, got nil")
	}
}

func TestEngine_Stop(t *testing.T) {
	config := Config{Name: "test-engine"}
	eng := New(config)
	ctx := context.Background()

	// Stop on idle engine should return nil
	if err := eng.Stop(ctx); err != nil {
		t.Errorf("unexpected error stopping idle engine: %v", err)
	}

	// Start and then stop
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("unexpected error starting: %v", err)
	}

	if err := eng.Stop(ctx); err != nil {
		t.Errorf("unexpected error stopping: %v", err)
	}

	if eng.state != StateStopped {
		t.Errorf("expected state %v, got %v", StateStopped, eng.state)
	}
}

func TestEngine_State(t *testing.T) {
	config := Config{Name: "test-engine"}
	eng := New(config)
	ctx := context.Background()

	// Initial state
	if state := eng.State(); state != StateIdle {
		t.Errorf("expected initial state %v, got %v", StateIdle, state)
	}

	// After start
	eng.Start(ctx)
	if state := eng.State(); state != StateRunning {
		t.Errorf("expected state %v, got %v", StateRunning, state)
	}

	// After stop
	eng.Stop(ctx)
	if state := eng.State(); state != StateStopped {
		t.Errorf("expected state %v, got %v", StateStopped, state)
	}
}
