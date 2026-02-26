package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/signal"
	"github.com/goclaw/goclaw/pkg/storage/memory"
)

func TestEngine_SignalContextInjected(t *testing.T) {
	bus := signal.NewLocalBus(16)
	defer bus.Close()

	eng, err := New(minConfig(), nil, memory.NewMemoryStorage(), WithSignalBus(bus))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer eng.Stop(ctx)

	received := make(chan *signal.Signal, 1)

	wf := &Workflow{
		ID: "wf-signal",
		Tasks: []*dag.Task{
			{ID: "t1", Name: "t1", Agent: "test"},
		},
		TaskFns: map[string]func(context.Context) error{
			"t1": func(ctx context.Context) error {
				ch := signal.FromContext(ctx)
				if ch == nil {
					return fmt.Errorf("missing signal channel")
				}
				select {
				case sig := <-ch:
					received <- sig
					return nil
				case <-time.After(500 * time.Millisecond):
					return fmt.Errorf("timeout waiting for signal")
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		},
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = signal.SendSteer(context.Background(), bus, "t1", map[string]interface{}{"mode": "fast"})
	}()

	result, err := eng.Submit(ctx, wf)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if result.Status != WorkflowStatusSuccess {
		t.Fatalf("expected success, got %v", result.Status)
	}

	select {
	case sig := <-received:
		if sig.Type != signal.SignalSteer {
			t.Fatalf("expected steer signal, got %s", sig.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("signal not received")
	}
}
