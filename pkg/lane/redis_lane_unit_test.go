package lane

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRedisLane_Unit_SubmitTrySubmitAndStats(t *testing.T) {
	client := newMockRedisClient(t)

	cfg := DefaultRedisConfig("unit")
	cfg.KeyPrefix = uniqueKeyPrefix("unit")
	cfg.Capacity = 2
	cfg.MaxConcurrency = 1
	cfg.BlockTimeout = 50 * time.Millisecond

	l, err := NewRedisLane(client, cfg)
	if err != nil {
		t.Fatalf("NewRedisLane failed: %v", err)
	}
	t.Cleanup(func() {
		_ = l.Close(context.Background())
	})

	// Without workers, queue should fill to capacity.
	if !l.TrySubmit(NewTaskFunc("t1", "unit", 1, nil)) {
		t.Fatal("expected first TrySubmit to succeed")
	}
	if !l.TrySubmit(NewTaskFunc("t2", "unit", 1, nil)) {
		t.Fatal("expected second TrySubmit to succeed")
	}
	if l.TrySubmit(NewTaskFunc("t3", "unit", 1, nil)) {
		t.Fatal("expected third TrySubmit to fail when queue is full")
	}

	stats := l.Stats()
	if stats.Pending != 2 {
		t.Fatalf("expected pending=2, got %d", stats.Pending)
	}
}

func TestRedisLane_Unit_RunAndExecute(t *testing.T) {
	client := newMockRedisClient(t)

	cfg := DefaultRedisConfig("unit-worker")
	cfg.KeyPrefix = uniqueKeyPrefix("unit-worker")
	cfg.Capacity = 20
	cfg.MaxConcurrency = 2
	cfg.BlockTimeout = 20 * time.Millisecond

	l, err := NewRedisLane(client, cfg)
	if err != nil {
		t.Fatalf("NewRedisLane failed: %v", err)
	}
	t.Cleanup(func() {
		_ = l.Close(context.Background())
	})

	var handled atomic.Int32
	l.SetTaskHandler(func(ctx context.Context, payload *RedisTaskPayload) error {
		handled.Add(1)
		return nil
	})
	l.Run()

	total := 6
	for i := 0; i < total; i++ {
		task := NewTaskFunc("job-"+time.Now().Add(time.Duration(i)).Format("150405.000000"), "unit-worker", i, nil)
		if err := l.Submit(context.Background(), task); err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if int(handled.Load()) >= total {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if int(handled.Load()) != total {
		t.Fatalf("expected handled=%d, got %d", total, handled.Load())
	}
	if got := l.Stats().Completed; got != int64(total) {
		t.Fatalf("expected completed=%d, got %d", total, got)
	}
}
