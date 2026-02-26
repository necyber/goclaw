package lane

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestFallbackLane_Integration_DegradeOnRedisOutage(t *testing.T) {
	client := newMockRedisClient(t)

	redisCfg := DefaultRedisConfig("fallback-int")
	redisCfg.KeyPrefix = uniqueKeyPrefix("fallback-int")
	redisCfg.Capacity = 16
	redisCfg.MaxConcurrency = 2
	redisCfg.BlockTimeout = 50 * time.Millisecond

	primary, err := NewRedisLane(client, redisCfg)
	if err != nil {
		t.Fatalf("new redis lane failed: %v", err)
	}
	fallback, err := New(&Config{
		Name:           "fallback-int",
		Capacity:       16,
		MaxConcurrency: 2,
		Backpressure:   Block,
	})
	if err != nil {
		t.Fatalf("new fallback channel lane failed: %v", err)
	}

	fl, err := NewFallbackLane(primary, fallback, &FallbackConfig{CheckInterval: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("NewFallbackLane failed: %v", err)
	}
	t.Cleanup(func() {
		_ = fl.Close(context.Background())
	})
	fl.Run()

	// Simulate Redis outage.
	client.SetDown(true)
	outageAt := time.Now()

	var executed atomic.Bool
	task := NewTaskFunc(fmt.Sprintf("fallback-%d", time.Now().UnixNano()), "fallback-int", 1, func(ctx context.Context) error {
		executed.Store(true)
		return nil
	})
	if err := fl.Submit(context.Background(), task); err != nil {
		t.Fatalf("submit via fallback lane failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var degradedAt time.Time
	for time.Now().Before(deadline) {
		if fl.IsDegraded() && degradedAt.IsZero() {
			degradedAt = time.Now()
		}
		if !degradedAt.IsZero() && executed.Load() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !fl.IsDegraded() {
		t.Fatal("expected fallback lane to enter degraded mode")
	}
	if !executed.Load() {
		t.Fatal("expected task to execute via local fallback after degradation")
	}
	if degradedAt.IsZero() {
		t.Fatal("expected to observe degrade timestamp")
	}
	if degradedAt.Sub(outageAt) >= time.Second {
		t.Fatalf("expected degrade switch time < 1s, got %s", degradedAt.Sub(outageAt))
	}
}
