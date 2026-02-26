package lane

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRedisLane_Integration_SubmitAndConsume(t *testing.T) {
	client := requireRedisClient(t)

	cfg := DefaultRedisConfig("integration-io")
	cfg.KeyPrefix = uniqueKeyPrefix("integration-io")
	cfg.Capacity = 64
	cfg.MaxConcurrency = 4
	cfg.BlockTimeout = time.Second

	l, err := NewRedisLane(client, cfg)
	if err != nil {
		t.Fatalf("NewRedisLane failed: %v", err)
	}
	t.Cleanup(func() {
		_ = l.Close(context.Background())
	})

	l.Run()

	total := 20
	for i := 0; i < total; i++ {
		task := NewTaskFunc(fmt.Sprintf("int-%d-%d", time.Now().UnixNano(), i), "integration-io", i, nil)
		if err := l.Submit(context.Background(), task); err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if l.Stats().Completed >= int64(total) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	stats := l.Stats()
	if stats.Completed < int64(total) {
		t.Fatalf("expected completed >= %d, got %d", total, stats.Completed)
	}
}

func TestManager_Integration_MixedLaneMode(t *testing.T) {
	client := requireRedisClient(t)

	manager := NewManager()
	manager.SetRedisClient(client)

	memSpec := &LaneSpec{
		Type: LaneTypeMemory,
		Memory: &Config{
			Name:           "cpu-int",
			Capacity:       16,
			MaxConcurrency: 2,
			Backpressure:   Block,
		},
	}
	redisSpec := &LaneSpec{
		Type: LaneTypeRedis,
		Redis: &RedisConfig{
			Name:           "io-int",
			Capacity:       32,
			MaxConcurrency: 2,
			Backpressure:   Block,
			KeyPrefix:      uniqueKeyPrefix("io-int"),
			BlockTimeout:   time.Second,
		},
	}

	if _, err := manager.RegisterSpec(memSpec); err != nil {
		t.Fatalf("register memory lane failed: %v", err)
	}
	_, err := manager.RegisterSpec(redisSpec)
	if err != nil {
		t.Fatalf("register redis lane failed: %v", err)
	}

	// Submit into both memory and redis lanes.
	for i := 0; i < 5; i++ {
		if err := manager.Submit(context.Background(), NewTaskFunc(fmt.Sprintf("cpu-%d", i), "cpu-int", i, func(ctx context.Context) error {
			return nil
		})); err != nil {
			t.Fatalf("submit memory failed: %v", err)
		}
		if err := manager.Submit(context.Background(), NewTaskFunc(fmt.Sprintf("io-%d-%d", time.Now().UnixNano(), i), "io-int", i, nil)); err != nil {
			t.Fatalf("submit redis failed: %v", err)
		}
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		stats := manager.GetStats()
		if stats["cpu-int"].Completed >= 5 && stats["io-int"].Completed >= 5 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	stats := manager.GetStats()
	if stats["cpu-int"].Completed < 5 {
		t.Fatalf("expected memory completed >= 5, got %d", stats["cpu-int"].Completed)
	}
	if stats["io-int"].Completed < 5 {
		t.Fatalf("expected redis completed >= 5, got %d", stats["io-int"].Completed)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := manager.Close(ctx); err != nil {
		t.Fatalf("manager close failed: %v", err)
	}
}
