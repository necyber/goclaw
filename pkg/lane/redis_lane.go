package lane

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisMetricsRecorder interface {
	SetRedisQueueDepth(laneName string, depth float64)
	RecordRedisSubmitDuration(laneName string, duration time.Duration)
	RecordRedisThroughput(laneName string)
}

type redisOwnershipMetricsRecorder interface {
	RecordRedisOwnershipDecision(laneName string, decision string)
}

// RedisOwnershipGuard validates ownership before dequeue and execution in distributed mode.
type RedisOwnershipGuard interface {
	CanConsume(ctx context.Context, laneName string) (bool, error)
	ValidateFencing(ctx context.Context, shardKey string, fencingToken uint64) error
}

// DistributedTaskMetadata exposes shard/fencing metadata used in distributed ownership flows.
type DistributedTaskMetadata interface {
	ShardKey() string
	FencingToken() uint64
}

// RedisTaskPayload is the JSON structure for tasks stored in Redis.
type RedisTaskPayload struct {
	ID         string            `json:"id"`
	Lane       string            `json:"lane"`
	ShardKey   string            `json:"shard_key,omitempty"`
	Fencing    uint64            `json:"fencing_token,omitempty"`
	Priority   int               `json:"priority"`
	Payload    json.RawMessage   `json:"payload,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	EnqueuedAt time.Time         `json:"enqueued_at"`
}

// RedisLane implements the Lane interface using Redis as the backing store.
type RedisLane struct {
	config *RedisConfig
	client redis.Cmdable

	// Redis keys
	queueKey string // List for FIFO or Sorted Set for priority
	dedupKey string // Set for deduplication
	statsKey string // Hash for stats

	// State
	closed    atomic.Bool
	closeCh   chan struct{}
	closeOnce sync.Once

	// Statistics
	pending   atomic.Int32
	running   atomic.Int32
	completed atomic.Int64
	failed    atomic.Int64
	dropped   atomic.Int64
	accepted  atomic.Int64
	rejected  atomic.Int64
	redirected atomic.Int64

	// Worker management
	taskHandler func(ctx context.Context, payload *RedisTaskPayload) error
	workerWg    sync.WaitGroup

	// For redirect strategy
	manager *Manager

	// Metrics
	metrics MetricsRecorder

	// Optional distributed ownership guard.
	ownershipGuard RedisOwnershipGuard
}

// NewRedisLane creates a new Redis-backed Lane.
func NewRedisLane(client redis.Cmdable, config *RedisConfig) (*RedisLane, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}

	prefix := config.KeyPrefix + config.Name
	l := &RedisLane{
		config:   config,
		client:   client,
		queueKey: prefix + ":queue",
		dedupKey: prefix + ":dedup",
		statsKey: prefix + ":stats",
		closeCh:  make(chan struct{}),
		metrics:  &nopMetrics{},
	}

	return l, nil
}

// Name returns the lane name.
func (l *RedisLane) Name() string {
	return l.config.Name
}

// Submit submits a task to the Redis queue.
func (l *RedisLane) Submit(ctx context.Context, task Task) (err error) {
	start := time.Now()
	if l.closed.Load() {
		l.recordRejected()
		return &LaneClosedError{LaneName: l.config.Name}
	}
	if task == nil {
		l.recordRejected()
		return fmt.Errorf("task cannot be nil")
	}

	dedupAdded := false
	// Check dedup
	if l.config.EnableDedup {
		added, sErr := l.client.SAdd(ctx, l.dedupKey, task.ID()).Result()
		if sErr != nil {
			l.recordRejected()
			return fmt.Errorf("dedup check failed: %w", sErr)
		}
		if added == 0 {
			l.recordRejected()
			return &TaskDuplicateError{LaneName: l.config.Name, TaskID: task.ID()}
		}
		dedupAdded = true
		if l.config.DedupTTL > 0 {
			_ = l.client.Expire(ctx, l.dedupKey, l.config.DedupTTL).Err()
		}
	}

	defer func() {
		if err != nil && dedupAdded {
			_ = l.removeDedup(context.Background(), task.ID())
		}
	}()

	// Check capacity and apply backpressure.
	// Fast path uses local counters to avoid a Redis round-trip on every submit.
	queueLen := int64(l.pending.Load())
	if queueLen >= int64(l.config.Capacity) {
		var err error
		queueLen, err = l.queueLength(ctx)
		if err != nil {
			l.recordRejected()
			return fmt.Errorf("failed to check queue length: %w", err)
		}
	}

	if queueLen >= int64(l.config.Capacity) {
		switch l.config.Backpressure {
		case Drop:
			l.dropped.Add(1)
			l.recordDropped()
			return &TaskDroppedError{LaneName: l.config.Name, TaskID: task.ID()}
		case Redirect:
			if l.manager != nil {
				redirectLane, rerr := l.manager.GetLane(l.config.RedirectLane)
				if rerr == nil {
					_ = l.removeDedup(context.Background(), task.ID())
					dedupAdded = false
					if submitErr := redirectLane.Submit(ctx, task); submitErr == nil {
						l.recordRedirected()
						return nil
					}
					if ctx.Err() != nil {
						l.recordRejected()
						return ctx.Err()
					}
				}
			}
			l.dropped.Add(1)
			l.recordDropped()
			return &LaneFullError{LaneName: l.config.Name, Capacity: l.config.Capacity}
		case Block:
			// Poll until space available
			for queueLen >= int64(l.config.Capacity) {
				select {
				case <-ctx.Done():
					l.recordRejected()
					return ctx.Err()
				case <-l.closeCh:
					l.recordRejected()
					return &LaneClosedError{LaneName: l.config.Name}
				case <-time.After(100 * time.Millisecond):
					var err error
					queueLen, err = l.queueLength(ctx)
					if err != nil {
						l.recordRejected()
						return err
					}
				}
			}
		}
	}

	// Serialize and enqueue
	payload := RedisTaskPayload{
		ID:         task.ID(),
		Lane:       task.Lane(),
		Priority:   task.Priority(),
		EnqueuedAt: time.Now(),
	}
	if distributedTask, ok := task.(DistributedTaskMetadata); ok {
		payload.ShardKey = distributedTask.ShardKey()
		payload.Fencing = distributedTask.FencingToken()
	}

	data, err := json.Marshal(payload)
	if err != nil {
		l.recordRejected()
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	if l.config.EnablePriority {
		err = l.client.ZAdd(ctx, l.queueKey, redis.Z{
			Score:  float64(task.Priority()),
			Member: string(data),
		}).Err()
	} else {
		err = l.client.LPush(ctx, l.queueKey, data).Err()
	}

	if err != nil {
		l.recordRejected()
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	l.pending.Add(1)
	l.recordAccepted()
	l.metrics.IncQueueDepth(l.config.Name)
	if recorder, ok := l.metrics.(redisMetricsRecorder); ok {
		recorder.SetRedisQueueDepth(l.config.Name, float64(l.pending.Load()))
		recorder.RecordRedisSubmitDuration(l.config.Name, time.Since(start))
	}
	return nil
}

// TrySubmit attempts to submit a task without blocking.
func (l *RedisLane) TrySubmit(task Task) bool {
	if l.closed.Load() {
		l.recordRejected()
		return false
	}
	if task == nil {
		l.recordRejected()
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	queueLen, err := l.queueLength(ctx)
	if err != nil || queueLen >= int64(l.config.Capacity) {
		l.recordRejected()
		return false
	}

	err = l.Submit(ctx, task)
	return err == nil
}

// Stats returns current lane statistics.
func (l *RedisLane) Stats() Stats {
	return Stats{
		Name:           l.config.Name,
		Pending:        int(l.pending.Load()),
		Running:        int(l.running.Load()),
		Completed:      l.completed.Load(),
		Failed:         l.failed.Load(),
		Dropped:        l.dropped.Load(),
		Accepted:       l.accepted.Load(),
		Rejected:       l.rejected.Load(),
		Redirected:     l.redirected.Load(),
		Capacity:       l.config.Capacity,
		MaxConcurrency: l.config.MaxConcurrency,
	}
}

// Close gracefully shuts down the Redis lane.
func (l *RedisLane) Close(ctx context.Context) error {
	var err error
	l.closeOnce.Do(func() {
		l.closed.Store(true)
		close(l.closeCh)

		// Wait for workers to finish
		done := make(chan struct{})
		go func() {
			l.workerWg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})
	return err
}

// IsClosed returns true if the lane is closed.
func (l *RedisLane) IsClosed() bool {
	return l.closed.Load()
}

// SetManager sets the lane manager for redirect strategy.
func (l *RedisLane) SetManager(m *Manager) {
	l.manager = m
}

// SetMetrics sets the metrics recorder.
func (l *RedisLane) SetMetrics(m MetricsRecorder) {
	if m != nil {
		l.metrics = m
	}
}

func (l *RedisLane) recordAccepted() {
	l.accepted.Add(1)
	l.recordOutcome("accepted")
}

func (l *RedisLane) recordRejected() {
	l.rejected.Add(1)
	l.recordOutcome("rejected")
}

func (l *RedisLane) recordRedirected() {
	l.redirected.Add(1)
	l.recordOutcome("redirected")
}

func (l *RedisLane) recordDropped() {
	l.recordOutcome("dropped")
}

func (l *RedisLane) recordOutcome(outcome string) {
	if recorder, ok := l.metrics.(submissionOutcomeRecorder); ok {
		recorder.RecordSubmissionOutcome(l.config.Name, outcome)
	}
}

// SetOwnershipGuard sets an optional ownership guard for distributed consumption.
func (l *RedisLane) SetOwnershipGuard(guard RedisOwnershipGuard) {
	l.ownershipGuard = guard
}

// SetTaskHandler sets the function that processes dequeued tasks.
func (l *RedisLane) SetTaskHandler(handler func(ctx context.Context, payload *RedisTaskPayload) error) {
	l.taskHandler = handler
}

// Run starts the worker pool consuming from the Redis queue.
func (l *RedisLane) Run() {
	for i := 0; i < l.config.MaxConcurrency; i++ {
		l.workerWg.Add(1)
		go l.worker()
	}
}

func (l *RedisLane) worker() {
	defer l.workerWg.Done()

	for {
		select {
		case <-l.closeCh:
			return
		default:
		}

		ctx := context.Background()
		if l.ownershipGuard != nil {
			allowed, gErr := l.ownershipGuard.CanConsume(ctx, l.config.Name)
			if recorder, ok := l.metrics.(redisOwnershipMetricsRecorder); ok {
				decision := "allow"
				if gErr != nil {
					decision = "error"
				} else if !allowed {
					decision = "deny"
				}
				recorder.RecordRedisOwnershipDecision(l.config.Name, decision)
			}
			if gErr != nil || !allowed {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		payload, err := l.dequeue(ctx)
		if err != nil {
			// Timeout or error — retry
			continue
		}
		if payload == nil {
			continue
		}

		l.pending.Add(-1)
		l.running.Add(1)
		l.metrics.DecQueueDepth(l.config.Name)
		if recorder, ok := l.metrics.(redisMetricsRecorder); ok {
			recorder.SetRedisQueueDepth(l.config.Name, float64(l.pending.Load()))
		}

		start := time.Now()
		if l.ownershipGuard != nil && payload.Fencing > 0 {
			shardKey := payload.ShardKey
			if shardKey == "" {
				shardKey = l.config.Name
			}
			if ferr := l.ownershipGuard.ValidateFencing(ctx, shardKey, payload.Fencing); ferr != nil {
				l.failed.Add(1)
				l.running.Add(-1)
				continue
			}
		}
		if l.taskHandler != nil {
			if herr := l.taskHandler(ctx, payload); herr != nil {
				l.failed.Add(1)
			} else {
				l.completed.Add(1)
			}
		} else {
			l.completed.Add(1)
		}
		_ = l.removeDedup(context.Background(), payload.ID)

		l.running.Add(-1)
		l.metrics.RecordWaitDuration(l.config.Name, time.Since(payload.EnqueuedAt))
		l.metrics.RecordThroughput(l.config.Name)
		if recorder, ok := l.metrics.(redisMetricsRecorder); ok {
			recorder.RecordRedisThroughput(l.config.Name)
		}
		_ = start // suppress unused if metrics not recording process time
	}
}

func (l *RedisLane) dequeue(ctx context.Context) (*RedisTaskPayload, error) {
	var data string

	if l.config.EnablePriority {
		// ZPOPMAX dequeues the highest score first (higher priority first).
		results, err := l.client.ZPopMax(ctx, l.queueKey, 1).Result()
		if err != nil {
			return nil, err
		}
		if len(results) == 0 {
			// No items, wait briefly
			time.Sleep(100 * time.Millisecond)
			return nil, nil
		}
		data = results[0].Member.(string)
	} else {
		// BRPOP for FIFO
		result, err := l.client.BRPop(ctx, l.config.BlockTimeout, l.queueKey).Result()
		if err != nil {
			if err == redis.Nil {
				return nil, nil
			}
			return nil, err
		}
		if len(result) < 2 {
			return nil, nil
		}
		data = result[1]
	}

	var payload RedisTaskPayload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &payload, nil
}

func (l *RedisLane) queueLength(ctx context.Context) (int64, error) {
	if l.config.EnablePriority {
		return l.client.ZCard(ctx, l.queueKey).Result()
	}
	return l.client.LLen(ctx, l.queueKey).Result()
}

func (l *RedisLane) removeDedup(ctx context.Context, taskID string) error {
	if !l.config.EnableDedup || taskID == "" {
		return nil
	}
	return l.client.SRem(ctx, l.dedupKey, taskID).Err()
}
