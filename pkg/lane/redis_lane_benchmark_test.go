package lane

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkRedisLane_Throughput(b *testing.B) {
	client := requireRedisClientTB(b)

	cfg := DefaultRedisConfig("bench-redis-lane")
	cfg.KeyPrefix = uniqueKeyPrefix("bench-redis-lane")
	cfg.Capacity = b.N + 64
	cfg.MaxConcurrency = 16
	cfg.BlockTimeout = time.Second

	l, err := NewRedisLane(client, cfg)
	if err != nil {
		b.Fatalf("NewRedisLane failed: %v", err)
	}
	b.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = l.Close(ctx)
	})

	var handled atomic.Int64
	done := make(chan struct{})
	l.SetTaskHandler(func(ctx context.Context, payload *RedisTaskPayload) error {
		if handled.Add(1) == int64(b.N) {
			close(done)
		}
		return nil
	})
	l.Run()

	tasks := make([]Task, b.N)
	for i := 0; i < b.N; i++ {
		tasks[i] = NewTaskFunc(fmt.Sprintf("bench-%d", i), cfg.Name, i, nil)
	}

	b.ResetTimer()
	start := time.Now()
	for i := 0; i < b.N; i++ {
		if err := l.Submit(context.Background(), tasks[i]); err != nil {
			b.Fatalf("submit failed: %v", err)
		}
	}
	elapsed := time.Since(start)
	b.StopTimer()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		b.Fatalf("benchmark timed out, handled=%d want=%d", handled.Load(), b.N)
	}

	if elapsed > 0 {
		b.ReportMetric(float64(b.N)/elapsed.Seconds(), "tasks/s")
	}
}
