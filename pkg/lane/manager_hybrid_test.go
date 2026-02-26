package lane

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestManager_RegisterSpec_MixedTypes(t *testing.T) {
	manager := NewManager()

	client := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:0",
		DialTimeout:  5 * time.Millisecond,
		ReadTimeout:  5 * time.Millisecond,
		WriteTimeout: 5 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })
	manager.SetRedisClient(client)

	memSpec := &LaneSpec{
		Type: LaneTypeMemory,
		Memory: &Config{
			Name:           "cpu",
			Capacity:       10,
			MaxConcurrency: 1,
			Backpressure:   Block,
		},
	}

	redisSpec := &LaneSpec{
		Type: LaneTypeRedis,
		Redis: &RedisConfig{
			Name:           "io",
			Capacity:       5,
			MaxConcurrency: 1,
			Backpressure:   Block,
			KeyPrefix:      "goclaw:lane:",
			BlockTimeout:   10 * time.Millisecond,
		},
	}

	if _, err := manager.RegisterSpec(memSpec); err != nil {
		t.Fatalf("failed to register memory lane: %v", err)
	}
	if _, err := manager.RegisterSpec(redisSpec); err != nil {
		t.Fatalf("failed to register redis lane: %v", err)
	}

	stats := manager.GetStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 lane stats, got %d", len(stats))
	}
	if _, ok := stats["cpu"]; !ok {
		t.Error("expected stats for memory lane")
	}
	if _, ok := stats["io"]; !ok {
		t.Error("expected stats for redis lane")
	}

	agg := manager.AggregateStats()
	expectedCapacity := memSpec.Memory.Capacity + redisSpec.Redis.Capacity
	if agg.Capacity != expectedCapacity {
		t.Errorf("expected aggregated capacity %d, got %d", expectedCapacity, agg.Capacity)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := manager.Close(ctx); err != nil {
		t.Fatalf("manager close failed: %v", err)
	}
}
