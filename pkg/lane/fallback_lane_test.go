package lane

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestFallbackLane_NewValidation(t *testing.T) {
	_, err := NewFallbackLane(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil primary")
	}

	redisLane := &RedisLane{config: &RedisConfig{Name: "test", Capacity: 10, MaxConcurrency: 2, KeyPrefix: "goclaw:lane:", BlockTimeout: time.Second}}
	_, err = NewFallbackLane(redisLane, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil fallback")
	}
}

// mockRedisCmd implements redis.Cmdable minimally for testing fallback behavior.
// We use a flag to simulate Redis being up or down.
type failingRedisLane struct {
	RedisLane
	failSubmit atomic.Bool
}

func newTestFallbackLane(t *testing.T) (*FallbackLane, *ChannelLane) {
	t.Helper()

	// Create fallback (local) lane
	localCfg := &Config{
		Name:           "test-fallback",
		Capacity:       100,
		MaxConcurrency: 2,
		Backpressure:   Block,
	}
	local, err := New(localCfg)
	if err != nil {
		t.Fatalf("failed to create local lane: %v", err)
	}

	// Create a RedisLane with a nil client — we'll test degradation behavior
	redisCfg := &RedisConfig{
		Name:           "test-redis",
		Capacity:       100,
		MaxConcurrency: 2,
		Backpressure:   Block,
		KeyPrefix:      "goclaw:lane:",
		BlockTimeout:   time.Second,
	}
	// We can't create a real RedisLane without a client, so we build one manually
	rl := &RedisLane{
		config:   redisCfg,
		queueKey: redisCfg.KeyPrefix + redisCfg.Name + ":queue",
		dedupKey: redisCfg.KeyPrefix + redisCfg.Name + ":dedup",
		statsKey: redisCfg.KeyPrefix + redisCfg.Name + ":stats",
		closeCh:  make(chan struct{}),
		metrics:  &nopMetrics{},
	}

	fl, err := NewFallbackLane(rl, local, &FallbackConfig{
		CheckInterval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("failed to create fallback lane: %v", err)
	}

	return fl, local
}

func TestFallbackLane_Name(t *testing.T) {
	fl, _ := newTestFallbackLane(t)
	defer fl.Close(context.Background())

	if fl.Name() != "test-redis" {
		t.Errorf("expected name 'test-redis', got '%s'", fl.Name())
	}
}

func TestFallbackLane_DegradeOnRedisError(t *testing.T) {
	fl, local := newTestFallbackLane(t)
	local.Run()
	defer fl.Close(context.Background())

	// The primary has no Redis client, so Submit will fail with a nil pointer or error.
	// This should trigger degradation to the fallback lane.
	if fl.IsDegraded() {
		t.Fatal("should not be degraded initially")
	}

	var executed atomic.Bool
	task := NewTaskFunc("task-1", "test-redis", 1, func(ctx context.Context) error {
		executed.Store(true)
		return nil
	})

	// Submit should degrade and use fallback
	ctx := context.Background()
	err := fl.Submit(ctx, task)
	if err != nil {
		t.Fatalf("Submit should succeed via fallback, got: %v", err)
	}

	if !fl.IsDegraded() {
		t.Error("should be degraded after Redis error")
	}

	if fl.DegradeCount() != 1 {
		t.Errorf("expected degrade count 1, got %d", fl.DegradeCount())
	}

	// Wait for task execution
	time.Sleep(100 * time.Millisecond)
	if !executed.Load() {
		t.Error("task should have been executed via fallback")
	}
}

func TestFallbackLane_StatsWhenDegraded(t *testing.T) {
	fl, local := newTestFallbackLane(t)
	local.Run()
	defer fl.Close(context.Background())

	// Force degradation
	fl.degrade("test")

	stats := fl.Stats()
	// Stats should come from fallback but with primary's name
	if stats.Name != "test-redis" {
		t.Errorf("expected stats name 'test-redis', got '%s'", stats.Name)
	}
}

func TestFallbackLane_TrySubmitWhenDegraded(t *testing.T) {
	fl, local := newTestFallbackLane(t)
	local.Run()
	defer fl.Close(context.Background())

	// Force degradation
	fl.degrade("test")

	task := NewTaskFunc("task-1", "test-redis", 1, func(ctx context.Context) error {
		return nil
	})

	ok := fl.TrySubmit(task)
	if !ok {
		t.Error("TrySubmit should succeed via fallback when degraded")
	}
}

func TestFallbackLane_IsRedisError(t *testing.T) {
	fl, _ := newTestFallbackLane(t)
	defer fl.Close(context.Background())

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"lane closed", &LaneClosedError{LaneName: "test"}, false},
		{"task dropped", &TaskDroppedError{LaneName: "test", TaskID: "t1"}, false},
		{"lane full", &LaneFullError{LaneName: "test", Capacity: 10}, false},
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, false},
		{"redis error", fmt.Errorf("connection refused"), true},
		{"redis timeout", fmt.Errorf("i/o timeout"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fl.isRedisError(tt.err)
			if got != tt.want {
				t.Errorf("isRedisError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestFallbackLane_DegradeAndRecover(t *testing.T) {
	fl, _ := newTestFallbackLane(t)
	defer fl.Close(context.Background())

	// Initially not degraded
	if fl.IsDegraded() {
		t.Fatal("should not be degraded initially")
	}

	// Degrade
	fl.degrade("test reason")
	if !fl.IsDegraded() {
		t.Fatal("should be degraded")
	}
	if fl.DegradeCount() != 1 {
		t.Errorf("expected degrade count 1, got %d", fl.DegradeCount())
	}

	// Degrade again (should be idempotent)
	fl.degrade("another reason")
	if fl.DegradeCount() != 1 {
		t.Errorf("expected degrade count still 1, got %d", fl.DegradeCount())
	}

	// Recover
	fl.recover()
	if fl.IsDegraded() {
		t.Fatal("should not be degraded after recovery")
	}
	if fl.RecoverCount() != 1 {
		t.Errorf("expected recover count 1, got %d", fl.RecoverCount())
	}

	// Recover again (should be idempotent)
	fl.recover()
	if fl.RecoverCount() != 1 {
		t.Errorf("expected recover count still 1, got %d", fl.RecoverCount())
	}
}

func TestFallbackLane_Close(t *testing.T) {
	fl, local := newTestFallbackLane(t)
	local.Run()

	// Start the health check loop
	go fl.healthCheckLoop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := fl.Close(ctx)
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !fl.IsClosed() {
		t.Error("expected lane to be closed")
	}
}

func TestFallbackLane_MultipleSubmitsWhileDegraded(t *testing.T) {
	fl, local := newTestFallbackLane(t)
	local.Run()
	defer fl.Close(context.Background())

	// Force degradation
	fl.degrade("test")

	var counter atomic.Int32
	for i := 0; i < 10; i++ {
		task := NewTaskFunc(fmt.Sprintf("task-%d", i), "test-redis", 1, func(ctx context.Context) error {
			counter.Add(1)
			return nil
		})
		err := fl.Submit(context.Background(), task)
		if err != nil {
			t.Fatalf("Submit %d failed: %v", i, err)
		}
	}

	// Wait for tasks to complete
	time.Sleep(200 * time.Millisecond)

	if counter.Load() != 10 {
		t.Errorf("expected 10 tasks executed, got %d", counter.Load())
	}
}
