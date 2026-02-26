package lane

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestErrorsAndHelpers(t *testing.T) {
	if !IsLaneFullError(&LaneFullError{LaneName: "l1", Capacity: 1}) {
		t.Fatal("expected IsLaneFullError true")
	}
	if !IsLaneClosedError(&LaneClosedError{LaneName: "l1"}) {
		t.Fatal("expected IsLaneClosedError true")
	}
	if !IsTaskDroppedError(&TaskDroppedError{LaneName: "l1", TaskID: "t1"}) {
		t.Fatal("expected IsTaskDroppedError true")
	}
	if !IsTaskDuplicateError(&TaskDuplicateError{LaneName: "l1", TaskID: "t1"}) {
		t.Fatal("expected IsTaskDuplicateError true")
	}
	if !IsLaneNotFoundError(&LaneNotFoundError{LaneName: "l1"}) {
		t.Fatal("expected IsLaneNotFoundError true")
	}
	if got := (&RateLimitError{LaneName: "l1", WaitTime: 1.23}).Error(); !strings.Contains(got, "rate limit") {
		t.Fatalf("unexpected RateLimitError message: %s", got)
	}
}

func TestPriorityQueueExtraPaths(t *testing.T) {
	pq := NewPriorityQueue()
	if pq.Peek() != nil {
		t.Fatal("peek on empty queue should be nil")
	}

	pq.Push(NewTaskFunc("a", "lane", 1, nil))
	pq.Push(NewTaskFunc("b", "lane", 10, nil))
	if pq.IsEmpty() {
		t.Fatal("queue should not be empty")
	}
	if pq.Peek().ID() != "b" {
		t.Fatalf("expected highest priority task b, got %s", pq.Peek().ID())
	}

	pq.Clear()
	if !pq.IsEmpty() {
		t.Fatal("queue should be empty after clear")
	}
}

func TestConcurrentPriorityQueuePaths(t *testing.T) {
	cpq := NewConcurrentPriorityQueue()
	if cpq.IsClosed() {
		t.Fatal("queue should not be closed initially")
	}
	if _, ok := cpq.TryPop(); ok {
		t.Fatal("TryPop should fail on empty queue")
	}

	cpq.Push(NewTaskFunc("x", "lane", 3, nil))
	task, ok := cpq.TryPop()
	if !ok || task == nil || task.ID() != "x" {
		t.Fatalf("TryPop expected task x, got ok=%v task=%v", ok, task)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		task, ok := cpq.Pop()
		if ok || task != nil {
			t.Errorf("expected closed empty pop to return nil,false, got %v,%v", task, ok)
		}
	}()

	time.Sleep(20 * time.Millisecond)
	cpq.Close()
	<-done

	if !cpq.IsClosed() {
		t.Fatal("queue should be closed")
	}
}

func TestTokenBucketExtraPaths(t *testing.T) {
	tb := NewTokenBucket(1, 1)
	if !tb.Allow() {
		t.Fatal("first token should be allowed")
	}
	if tb.Allow() {
		t.Fatal("second immediate allow should fail")
	}
	if tb.WaitTimeout(20 * time.Millisecond) {
		t.Fatal("WaitTimeout should fail when token not replenished in time")
	}

	tb.SetRate(1000)
	tb.SetCapacity(2)
	if tb.Rate() != 1000 {
		t.Fatalf("unexpected rate: %v", tb.Rate())
	}
	if tb.Capacity() != 2 {
		t.Fatalf("unexpected capacity: %v", tb.Capacity())
	}
	if tb.Tokens() > 2 {
		t.Fatalf("tokens should be capped by capacity, got %v", tb.Tokens())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := tb.Wait(ctx); err != nil {
		t.Fatalf("Wait should succeed with high refill rate: %v", err)
	}
}

func TestLeakyBucketExtraPaths(t *testing.T) {
	lb := NewLeakyBucket(1, 1)
	defer lb.Stop()

	if !lb.Allow() {
		t.Fatal("first allow should succeed")
	}
	if lb.Allow() {
		t.Fatal("second allow should fail when full")
	}
	if lb.QueueSize() != 1 {
		t.Fatalf("expected queue size 1, got %d", lb.QueueSize())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := lb.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}

	lb.Stop()
	if err := lb.Wait(context.Background()); !errors.Is(err, ErrLeakyBucketStopped) {
		t.Fatalf("expected ErrLeakyBucketStopped, got %v", err)
	}
	if got := ErrLeakyBucketStopped.Error(); got == "" {
		t.Fatal("expected non-empty stopped error message")
	}
}

func TestWorkerPoolExtraPaths(t *testing.T) {
	release := make(chan struct{})
	var processed atomic.Int64

	pool := NewWorkerPool(1, func(task Task) {
		if task.ID() == "panic" {
			panic("boom")
		}
		if task.ID() == "block" {
			<-release
		}
		processed.Add(1)
	})
	pool.Start()
	defer pool.Stop()

	if !pool.IsRunning() {
		t.Fatal("pool should be running")
	}

	pool.Submit(NewTaskFunc("block", "lane", 1, nil))
	time.Sleep(20 * time.Millisecond)
	if pool.TrySubmit(NewTaskFunc("later", "lane", 1, nil)) {
		t.Fatal("TrySubmit should fail while single worker is blocked")
	}

	close(release)
	pool.Submit(NewTaskFunc("panic", "lane", 1, nil))
	pool.Submit(NewTaskFunc("ok", "lane", 1, nil))

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if pool.TasksProcessed() >= 2 && processed.Load() >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if pool.TasksProcessed() < 2 {
		t.Fatalf("expected at least 2 processed tasks, got %d", pool.TasksProcessed())
	}
}

func TestDynamicWorkerPoolStartAndScale(t *testing.T) {
	var processed atomic.Int64
	pool := NewDynamicWorkerPool(1, 2, func(task Task) {
		processed.Add(1)
	})
	pool.Start()
	defer pool.Stop()

	if !pool.IsRunning() {
		t.Fatal("dynamic pool should be running after start")
	}

	select {
	case pool.scaleUpCh <- struct{}{}:
	default:
	}

	pool.Submit(NewTaskFunc("d1", "lane", 1, nil))
	pool.Submit(NewTaskFunc("d2", "lane", 1, nil))

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if processed.Load() >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if processed.Load() < 2 {
		t.Fatalf("expected dynamic pool to process tasks, got %d", processed.Load())
	}
}

type errCloseLane struct {
	name string
}

func (l *errCloseLane) Name() string                       { return l.name }
func (l *errCloseLane) Submit(context.Context, Task) error { return nil }
func (l *errCloseLane) TrySubmit(Task) bool                { return true }
func (l *errCloseLane) Stats() Stats                       { return Stats{Name: l.name} }
func (l *errCloseLane) Close(context.Context) error        { return errors.New("close failed") }
func (l *errCloseLane) IsClosed() bool                     { return false }

func TestManagerExtraPaths(t *testing.T) {
	m := NewManager()

	if err := m.RegisterLane(nil); err == nil {
		t.Fatal("expected error when registering nil lane")
	}

	mem, err := New(&Config{Name: "m1", Capacity: 2, MaxConcurrency: 1, Backpressure: Block})
	if err != nil {
		t.Fatalf("new memory lane failed: %v", err)
	}
	if err := m.RegisterLane(mem); err != nil {
		t.Fatalf("register memory lane failed: %v", err)
	}
	if err := m.RegisterLane(mem); err == nil {
		t.Fatal("expected duplicate register error")
	}

	if m.TrySubmit(nil) {
		t.Fatal("TrySubmit(nil) should return false")
	}
	if m.TrySubmit(NewTaskFunc("x", "", 1, nil)) {
		t.Fatal("TrySubmit with empty lane should return false")
	}
	if err := m.Submit(context.Background(), nil); err == nil {
		t.Fatal("Submit(nil) should fail")
	}
	if err := m.Unregister(context.Background(), "not-found"); err == nil {
		t.Fatal("unregister unknown lane should fail")
	}
	if err := m.Unregister(context.Background(), "m1"); err != nil {
		t.Fatalf("unregister existing lane failed: %v", err)
	}

	if err := m.RegisterLane(&errCloseLane{name: "bad-close"}); err != nil {
		t.Fatalf("register errClose lane failed: %v", err)
	}
	if err := m.Close(context.Background()); err == nil {
		t.Fatal("expected manager close aggregation error")
	}
}
