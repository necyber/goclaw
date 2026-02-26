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

func TestRedisLane_Unit_PriorityDequeuesHighestFirst(t *testing.T) {
	client := newMockRedisClient(t)

	cfg := DefaultRedisConfig("priority")
	cfg.KeyPrefix = uniqueKeyPrefix("priority")
	cfg.EnablePriority = true
	cfg.Capacity = 10
	cfg.MaxConcurrency = 1

	l, err := NewRedisLane(client, cfg)
	if err != nil {
		t.Fatalf("NewRedisLane failed: %v", err)
	}

	if err := l.Submit(context.Background(), NewTaskFunc("low", "priority", 1, nil)); err != nil {
		t.Fatalf("submit low failed: %v", err)
	}
	if err := l.Submit(context.Background(), NewTaskFunc("high", "priority", 10, nil)); err != nil {
		t.Fatalf("submit high failed: %v", err)
	}

	first, err := l.dequeue(context.Background())
	if err != nil {
		t.Fatalf("first dequeue failed: %v", err)
	}
	second, err := l.dequeue(context.Background())
	if err != nil {
		t.Fatalf("second dequeue failed: %v", err)
	}
	if first == nil || second == nil {
		t.Fatalf("expected two dequeued payloads, got first=%v second=%v", first, second)
	}
	if first.ID != "high" || second.ID != "low" {
		t.Fatalf("expected dequeue order high->low, got %s->%s", first.ID, second.ID)
	}
}

func TestRedisLane_Unit_DedupDuplicateAndResubmit(t *testing.T) {
	client := newMockRedisClient(t)

	cfg := DefaultRedisConfig("dedup")
	cfg.KeyPrefix = uniqueKeyPrefix("dedup")
	cfg.EnableDedup = true
	cfg.Capacity = 8
	cfg.MaxConcurrency = 1
	cfg.BlockTimeout = 20 * time.Millisecond

	l, err := NewRedisLane(client, cfg)
	if err != nil {
		t.Fatalf("NewRedisLane failed: %v", err)
	}
	t.Cleanup(func() {
		_ = l.Close(context.Background())
	})

	if err := l.Submit(context.Background(), NewTaskFunc("task-1", "dedup", 1, nil)); err != nil {
		t.Fatalf("first submit failed: %v", err)
	}
	if err := l.Submit(context.Background(), NewTaskFunc("task-1", "dedup", 1, nil)); !IsTaskDuplicateError(err) {
		t.Fatalf("expected TaskDuplicateError, got: %v", err)
	}

	l.Run()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if l.Stats().Completed >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if l.Stats().Completed < 1 {
		t.Fatalf("expected first task to complete, got stats: %+v", l.Stats())
	}

	if err := l.Submit(context.Background(), NewTaskFunc("task-1", "dedup", 1, nil)); err != nil {
		t.Fatalf("resubmit after completion should succeed, got: %v", err)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if l.Stats().Completed >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if l.Stats().Completed < 2 {
		t.Fatalf("expected second completion after resubmit, got stats: %+v", l.Stats())
	}
}

func TestRedisLane_Unit_BackpressureDrop(t *testing.T) {
	client := newMockRedisClient(t)

	cfg := DefaultRedisConfig("drop")
	cfg.KeyPrefix = uniqueKeyPrefix("drop")
	cfg.Capacity = 1
	cfg.Backpressure = Drop

	l, err := NewRedisLane(client, cfg)
	if err != nil {
		t.Fatalf("NewRedisLane failed: %v", err)
	}

	if err := l.Submit(context.Background(), NewTaskFunc("d1", "drop", 1, nil)); err != nil {
		t.Fatalf("first submit failed: %v", err)
	}
	if err := l.Submit(context.Background(), NewTaskFunc("d2", "drop", 1, nil)); !IsTaskDroppedError(err) {
		t.Fatalf("expected TaskDroppedError, got: %v", err)
	}
}

func TestRedisLane_Unit_BackpressureRedirect(t *testing.T) {
	client := newMockRedisClient(t)

	fallback, err := New(&Config{
		Name:           "redirect-target",
		Capacity:       8,
		MaxConcurrency: 1,
		Backpressure:   Block,
	})
	if err != nil {
		t.Fatalf("new fallback lane failed: %v", err)
	}
	t.Cleanup(func() {
		_ = fallback.Close(context.Background())
	})

	manager := NewManager()
	if err := manager.RegisterLane(fallback); err != nil {
		t.Fatalf("register fallback lane failed: %v", err)
	}

	cfg := DefaultRedisConfig("redirect-src")
	cfg.KeyPrefix = uniqueKeyPrefix("redirect-src")
	cfg.Capacity = 1
	cfg.Backpressure = Redirect
	cfg.RedirectLane = "redirect-target"

	src, err := NewRedisLane(client, cfg)
	if err != nil {
		t.Fatalf("new source redis lane failed: %v", err)
	}
	src.SetManager(manager)

	if err := src.Submit(context.Background(), NewTaskFunc("r1", "redirect-src", 1, nil)); err != nil {
		t.Fatalf("first submit failed: %v", err)
	}
	if err := src.Submit(context.Background(), NewTaskFunc("r2", "redirect-src", 1, nil)); err != nil {
		t.Fatalf("redirect submit failed: %v", err)
	}

	stats := fallback.Stats()
	if stats.Pending != 1 {
		t.Fatalf("expected redirected task pending=1 in fallback lane, got %d", stats.Pending)
	}
}

func TestRedisLane_Unit_BackpressureBlock(t *testing.T) {
	client := newMockRedisClient(t)

	cfg := DefaultRedisConfig("block")
	cfg.KeyPrefix = uniqueKeyPrefix("block")
	cfg.Capacity = 1
	cfg.Backpressure = Block
	cfg.MaxConcurrency = 1
	cfg.BlockTimeout = 20 * time.Millisecond

	l, err := NewRedisLane(client, cfg)
	if err != nil {
		t.Fatalf("NewRedisLane failed: %v", err)
	}
	t.Cleanup(func() {
		_ = l.Close(context.Background())
	})

	if err := l.Submit(context.Background(), NewTaskFunc("b1", "block", 1, nil)); err != nil {
		t.Fatalf("first submit failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done <- l.Submit(ctx, NewTaskFunc("b2", "block", 1, nil))
	}()

	select {
	case err := <-done:
		t.Fatalf("submit should block before worker starts, got err=%v", err)
	case <-time.After(120 * time.Millisecond):
		// expected: still blocked
	}

	l.Run()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("blocked submit returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("blocked submit did not resume")
	}
}
