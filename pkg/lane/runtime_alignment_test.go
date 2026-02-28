package lane

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type outcomeMetricsStub struct {
	mu       sync.Mutex
	outcomes map[string]int
	waits    int
}

func newOutcomeMetricsStub() *outcomeMetricsStub {
	return &outcomeMetricsStub{
		outcomes: make(map[string]int),
	}
}

func (m *outcomeMetricsStub) IncQueueDepth(string) {}

func (m *outcomeMetricsStub) DecQueueDepth(string) {}

func (m *outcomeMetricsStub) RecordWaitDuration(string, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.waits++
}

func (m *outcomeMetricsStub) RecordThroughput(string) {}

func (m *outcomeMetricsStub) RecordSubmissionOutcome(_ string, outcome string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outcomes[outcome]++
}

func (m *outcomeMetricsStub) count(outcome string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.outcomes[outcome]
}

func (m *outcomeMetricsStub) waitCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.waits
}

func TestPriorityQueue_DeterministicTieBreak(t *testing.T) {
	pq := NewPriorityQueue()
	pq.Push(NewTaskFunc("a", "lane", 10, nil))
	pq.Push(NewTaskFunc("b", "lane", 10, nil))
	pq.Push(NewTaskFunc("c", "lane", 10, nil))

	if task := pq.Pop(); task == nil || task.ID() != "a" {
		t.Fatalf("expected a first, got %+v", task)
	}
	if task := pq.Pop(); task == nil || task.ID() != "b" {
		t.Fatalf("expected b second, got %+v", task)
	}
	if task := pq.Pop(); task == nil || task.ID() != "c" {
		t.Fatalf("expected c third, got %+v", task)
	}
}

func TestChannelLane_BackpressureOutcomeAccounting(t *testing.T) {
	l, err := New(&Config{
		Name:           "drop-accounting",
		Capacity:       1,
		MaxConcurrency: 1,
		Backpressure:   Drop,
	})
	if err != nil {
		t.Fatalf("create lane failed: %v", err)
	}
	defer l.Close(context.Background())

	metrics := newOutcomeMetricsStub()
	l.SetMetrics(metrics)

	if err := l.Submit(context.Background(), NewTaskFunc("ok", "drop-accounting", 1, nil)); err != nil {
		t.Fatalf("first submit failed: %v", err)
	}
	if err := l.Submit(context.Background(), NewTaskFunc("drop", "drop-accounting", 1, nil)); !IsTaskDroppedError(err) {
		t.Fatalf("expected dropped error, got %v", err)
	}
	if l.TrySubmit(nil) {
		t.Fatal("TrySubmit(nil) must fail")
	}

	stats := l.Stats()
	if stats.Accepted != 1 || stats.Dropped != 1 || stats.Rejected != 1 || stats.Redirected != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if metrics.count("accepted") != 1 || metrics.count("dropped") != 1 || metrics.count("rejected") != 1 {
		t.Fatalf("unexpected outcome metrics: accepted=%d dropped=%d rejected=%d",
			metrics.count("accepted"), metrics.count("dropped"), metrics.count("rejected"))
	}
}

func TestChannelLane_RedirectOutcomeAccounting(t *testing.T) {
	target, err := New(&Config{
		Name:           "redirect-target",
		Capacity:       2,
		MaxConcurrency: 1,
		Backpressure:   Block,
	})
	if err != nil {
		t.Fatalf("create target lane failed: %v", err)
	}

	source, err := New(&Config{
		Name:           "redirect-source",
		Capacity:       1,
		MaxConcurrency: 1,
		Backpressure:   Redirect,
		RedirectLane:   "redirect-target",
	})
	if err != nil {
		t.Fatalf("create source lane failed: %v", err)
	}

	mgr := NewManager()
	if err := mgr.RegisterLane(target); err != nil {
		t.Fatalf("register target failed: %v", err)
	}
	if err := mgr.RegisterLane(source); err != nil {
		t.Fatalf("register source failed: %v", err)
	}
	source.SetManager(mgr)

	metrics := newOutcomeMetricsStub()
	source.SetMetrics(metrics)

	if err := source.Submit(context.Background(), NewTaskFunc("s1", "redirect-source", 1, nil)); err != nil {
		t.Fatalf("source first submit failed: %v", err)
	}
	if err := source.Submit(context.Background(), NewTaskFunc("s2", "redirect-source", 1, nil)); err != nil {
		t.Fatalf("source redirect submit failed: %v", err)
	}

	stats := source.Stats()
	if stats.Accepted != 1 || stats.Redirected != 1 || stats.Dropped != 0 {
		t.Fatalf("unexpected source stats: %+v", stats)
	}
	if metrics.count("redirected") != 1 {
		t.Fatalf("expected redirected metric=1, got %d", metrics.count("redirected"))
	}

	if err := mgr.Close(context.Background()); err != nil {
		t.Fatalf("close manager failed: %v", err)
	}
}

func TestChannelLane_WaitDurationRecordedForStandardTask(t *testing.T) {
	l, err := New(&Config{
		Name:           "wait-accounting",
		Capacity:       4,
		MaxConcurrency: 1,
		Backpressure:   Block,
	})
	if err != nil {
		t.Fatalf("create lane failed: %v", err)
	}
	defer l.Close(context.Background())
	l.Run()

	metrics := newOutcomeMetricsStub()
	l.SetMetrics(metrics)

	if err := l.Submit(context.Background(), NewTaskFunc("w1", "wait-accounting", 1, nil)); err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if metrics.waitCount() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected wait duration to be recorded for standard submission")
}

func TestChannelLane_RedirectFailureIsNotClassifiedAsRedirected(t *testing.T) {
	target, err := New(&Config{
		Name:           "redirect-fail-target",
		Capacity:       2,
		MaxConcurrency: 1,
		Backpressure:   Block,
	})
	if err != nil {
		t.Fatalf("create target lane failed: %v", err)
	}

	source, err := New(&Config{
		Name:           "redirect-fail-source",
		Capacity:       1,
		MaxConcurrency: 1,
		Backpressure:   Redirect,
		RedirectLane:   "redirect-fail-target",
	})
	if err != nil {
		t.Fatalf("create source lane failed: %v", err)
	}

	mgr := NewManager()
	if err := mgr.RegisterLane(target); err != nil {
		t.Fatalf("register target failed: %v", err)
	}
	if err := mgr.RegisterLane(source); err != nil {
		t.Fatalf("register source failed: %v", err)
	}
	source.SetManager(mgr)

	metrics := newOutcomeMetricsStub()
	source.SetMetrics(metrics)

	if err := source.Submit(context.Background(), NewTaskFunc("s1", "redirect-fail-source", 1, nil)); err != nil {
		t.Fatalf("source first submit failed: %v", err)
	}
	if err := target.Close(context.Background()); err != nil {
		t.Fatalf("close target failed: %v", err)
	}
	if err := source.Submit(context.Background(), NewTaskFunc("s2", "redirect-fail-source", 1, nil)); !IsTaskDroppedError(err) {
		t.Fatalf("expected dropped error on failed redirect, got %v", err)
	}

	stats := source.Stats()
	if stats.Redirected != 0 || stats.Dropped != 1 {
		t.Fatalf("unexpected source stats: %+v", stats)
	}
	if metrics.count("redirected") != 0 || metrics.count("dropped") != 1 {
		t.Fatalf("unexpected outcome metrics redirected=%d dropped=%d",
			metrics.count("redirected"), metrics.count("dropped"))
	}

	if err := mgr.Close(context.Background()); err != nil {
		t.Fatalf("close manager failed: %v", err)
	}
}

func TestChannelLane_Close_Idempotent(t *testing.T) {
	l, err := New(&Config{
		Name:           "close-idempotent",
		Capacity:       2,
		MaxConcurrency: 1,
		Backpressure:   Block,
	})
	if err != nil {
		t.Fatalf("create lane failed: %v", err)
	}

	if err := l.Close(context.Background()); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := l.Close(context.Background()); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
	if !l.IsClosed() {
		t.Fatal("lane should remain closed after repeated close")
	}
}

func TestManager_ConcurrentSafetyAndCloseInvariants(t *testing.T) {
	mgr := NewManager()
	l, err := New(&Config{
		Name:           "cpu",
		Capacity:       16,
		MaxConcurrency: 2,
		Backpressure:   Block,
	})
	if err != nil {
		t.Fatalf("create lane failed: %v", err)
	}
	if err := mgr.RegisterLane(l); err != nil {
		t.Fatalf("register lane failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			task := NewTaskFunc(fmt.Sprintf("t-%d", i), "cpu", 1, nil)
			for j := 0; j < 200; j++ {
				_, _ = mgr.GetLane("cpu")
				_ = mgr.HasLane("cpu")
				_ = mgr.TrySubmit(task)
			}
		}(i)
	}

	done := make(chan struct{})
	go func() {
		_ = mgr.Close(context.Background())
		close(done)
	}()

	wg.Wait()
	<-done

	if err := mgr.Close(context.Background()); err != nil {
		t.Fatalf("manager second close should be idempotent, got %v", err)
	}
	if _, err := mgr.Register(&Config{
		Name:           "after-close",
		Capacity:       1,
		MaxConcurrency: 1,
		Backpressure:   Block,
	}); err == nil {
		t.Fatal("expected register to fail after manager close")
	}
}

func TestChannelLane_DynamicWorkerConfig_UsesDynamicPool(t *testing.T) {
	l, err := New(&Config{
		Name:                 "dynamic",
		Capacity:             4,
		MaxConcurrency:       3,
		EnableDynamicWorkers: true,
		MinConcurrency:       1,
		Backpressure:         Block,
	})
	if err != nil {
		t.Fatalf("create lane failed: %v", err)
	}
	defer l.Close(context.Background())

	if _, ok := l.workerPool.(*DynamicWorkerPool); !ok {
		t.Fatalf("expected DynamicWorkerPool, got %T", l.workerPool)
	}
}
