package engine

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/lane"
	"github.com/goclaw/goclaw/pkg/storage/memory"
	"github.com/redis/go-redis/v9"
)

func TestEngine_Submit_RedisQueueExecutesTaskFunction(t *testing.T) {
	client := requireEngineRedisClient(t)
	defer client.Close()

	cfg := minConfig()
	cfg.Orchestration.Queue.Type = "redis"
	cfg.Orchestration.Queue.Size = 64

	eng, err := New(cfg, nil, memory.NewMemoryStorage(), WithRedisClient(client))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer eng.Stop(ctx)

	var executed atomic.Bool
	wf := &Workflow{
		ID: "wf-redis-exec",
		Tasks: []*dag.Task{
			{ID: "redis-task", Name: "redis-task", Agent: "test"},
		},
		TaskFns: map[string]func(context.Context) error{
			"redis-task": func(context.Context) error {
				executed.Store(true)
				return nil
			},
		},
	}

	result, err := eng.Submit(ctx, wf)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if result.Status != WorkflowStatusSuccess {
		t.Fatalf("expected success status, got %v", result.Status)
	}
	if !executed.Load() {
		t.Fatal("expected task function to execute in redis queue mode")
	}
}

func requireEngineRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("GOCLAW_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           15,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("redis is not available at %s: %v", addr, err)
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("redis db flush failed: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Second)
		defer cleanupCancel()
		if err := client.FlushDB(cleanupCtx).Err(); err != nil {
			t.Logf("cleanup redis db flush failed: %v", err)
		}
	})

	return client
}

func TestEngine_Start_RedisQueueWithoutClientFallsBackToMemory(t *testing.T) {
	cfg := minConfig()
	cfg.Orchestration.Queue.Type = "redis"

	eng, err := New(cfg, nil, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer eng.Stop(context.Background())

	l, err := eng.laneManager.GetLane(defaultLaneName)
	if err != nil {
		t.Fatalf("GetLane: %v", err)
	}
	if _, ok := l.(*lane.ChannelLane); !ok {
		t.Fatalf("expected default lane fallback to ChannelLane, got %T", l)
	}
}
