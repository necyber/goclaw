package signal

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func BenchmarkSignalBusLatency(b *testing.B) {
	b.Run("local", func(b *testing.B) {
		benchmarkSignalBusLatency(b, NewLocalBus(1024))
	})

	b.Run("redis", func(b *testing.B) {
		client := requireRedisBusClient(b)
		bus := NewRedisBus(client, "goclaw:bench:signal:", 1024)
		benchmarkSignalBusLatency(b, bus)
	})
}

func benchmarkSignalBusLatency(b *testing.B, bus Bus) {
	b.Helper()
	defer bus.Close()

	taskID := "signal-bench-task"
	ch, err := bus.Subscribe(context.Background(), taskID)
	if err != nil {
		b.Fatalf("subscribe failed: %v", err)
	}
	b.Cleanup(func() {
		_ = bus.Unsubscribe(taskID)
	})

	payload, _ := json.Marshal(map[string]float64{"rate": 0.7})
	var totalLatency time.Duration
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sig := &Signal{
			Type:    SignalSteer,
			TaskID:  taskID,
			Payload: payload,
			SentAt:  time.Now(),
		}

		start := time.Now()
		if err := bus.Publish(context.Background(), sig); err != nil {
			b.Fatalf("publish failed: %v", err)
		}

		select {
		case <-ch:
			totalLatency += time.Since(start)
		case <-time.After(2 * time.Second):
			b.Fatalf("receive timeout in iteration %d", i)
		}
	}
	b.StopTimer()

	if b.N > 0 {
		avgNS := float64(totalLatency.Nanoseconds()) / float64(b.N)
		b.ReportMetric(avgNS, "avg_latency_ns")
	}
}

func requireRedisBusClient(tb testing.TB) redis.UniversalClient {
	tb.Helper()

	addr := os.Getenv("GOCLAW_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		tb.Skipf("redis is not available at %s: %v", addr, err)
	}

	tb.Cleanup(func() {
		_ = client.Close()
	})

	return client
}
