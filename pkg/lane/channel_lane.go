package lane

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsRecorder defines the interface for recording lane metrics.
type MetricsRecorder interface {
	IncQueueDepth(laneName string)
	DecQueueDepth(laneName string)
	RecordWaitDuration(laneName string, duration time.Duration)
	RecordThroughput(laneName string)
}

// ChannelLane implements Lane using Go channels.
type ChannelLane struct {
	config      *Config
	taskCh      chan Task
	workerPool  *WorkerPool
	rateLimiter *TokenBucket
	metrics     MetricsRecorder

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

	// For redirect strategy
	manager *Manager

	// Wait time tracking
	totalProcessTime atomic.Int64 // nanoseconds
	taskCount        atomic.Int64
}

// New creates a new ChannelLane with the given configuration.
func New(config *Config) (*ChannelLane, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	l := &ChannelLane{
		config:  config,
		taskCh:  make(chan Task, config.Capacity),
		closeCh: make(chan struct{}),
		metrics: &nopMetrics{},
	}

	// Initialize rate limiter if configured
	if config.RateLimit > 0 {
		l.rateLimiter = NewTokenBucket(config.RateLimit, config.RateLimit*2)
	}

	// Initialize worker pool
	l.workerPool = NewWorkerPool(config.MaxConcurrency, l.executeTask)
	l.workerPool.Start()

	return l, nil
}

// Name returns the lane name.
func (l *ChannelLane) Name() string {
	return l.config.Name
}

// Submit submits a task to the lane.
// The behavior depends on the backpressure strategy:
//   - Block: waits until space is available or context is cancelled
//   - Drop: returns TaskDroppedError if queue is full
//   - Redirect: redirects to another lane if queue is full
func (l *ChannelLane) Submit(ctx context.Context, task Task) error {
	if l.closed.Load() {
		return &LaneClosedError{LaneName: l.config.Name}
	}

	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	// Check rate limit
	if l.rateLimiter != nil {
		if err := l.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	switch l.config.Backpressure {
	case Block:
		return l.submitBlock(ctx, task)
	case Drop:
		return l.submitDrop(task)
	case Redirect:
		return l.submitRedirect(ctx, task)
	default:
		return l.submitBlock(ctx, task)
	}
}

// submitBlock blocks until the task can be submitted or context is cancelled.
func (l *ChannelLane) submitBlock(ctx context.Context, task Task) error {
	select {
	case l.taskCh <- task:
		l.pending.Add(1)
		l.metrics.IncQueueDepth(l.config.Name)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-l.closeCh:
		return &LaneClosedError{LaneName: l.config.Name}
	}
}

// submitDrop attempts to submit without blocking, drops if full.
func (l *ChannelLane) submitDrop(task Task) error {
	select {
	case l.taskCh <- task:
		l.pending.Add(1)
		l.metrics.IncQueueDepth(l.config.Name)
		return nil
	default:
		l.dropped.Add(1)
		return &TaskDroppedError{LaneName: l.config.Name, TaskID: task.ID()}
	}
}

// submitRedirect redirects to another lane if full.
func (l *ChannelLane) submitRedirect(ctx context.Context, task Task) error {
	select {
	case l.taskCh <- task:
		l.pending.Add(1)
		l.metrics.IncQueueDepth(l.config.Name)
		return nil
	default:
		// Try to redirect
		if l.manager != nil {
			targetLane, err := l.manager.GetLane(l.config.RedirectLane)
			if err == nil {
				return targetLane.Submit(ctx, task)
			}
		}
		// If redirect fails, drop the task
		l.dropped.Add(1)
		return &TaskDroppedError{LaneName: l.config.Name, TaskID: task.ID()}
	}
}

// TrySubmit attempts to submit a task without blocking.
// Returns true if the task was accepted, false otherwise.
func (l *ChannelLane) TrySubmit(task Task) bool {
	if l.closed.Load() {
		return false
	}

	if task == nil {
		return false
	}

	// Check rate limit
	if l.rateLimiter != nil && !l.rateLimiter.Allow() {
		return false
	}

	select {
	case l.taskCh <- task:
		l.pending.Add(1)
		l.metrics.IncQueueDepth(l.config.Name)
		return true
	default:
		return false
	}
}

// executeTask is called by the worker pool to execute a task.
func (l *ChannelLane) executeTask(task Task) {
	l.pending.Add(-1)
	l.metrics.DecQueueDepth(l.config.Name)

	// Record wait duration
	if tw, ok := task.(interface{ EnqueuedAt() time.Time }); ok {
		waitDuration := time.Since(tw.EnqueuedAt())
		l.metrics.RecordWaitDuration(l.config.Name, waitDuration)
	}

	l.running.Add(1)
	defer l.running.Add(-1)

	startTime := time.Now()

	// Execute the task
	var err error
	if taskFunc, ok := task.(*TaskFunc); ok {
		err = taskFunc.Execute(context.Background())
	}

	processTime := time.Since(startTime)
	l.totalProcessTime.Add(int64(processTime))
	l.taskCount.Add(1)

	if err != nil {
		l.failed.Add(1)
	} else {
		l.completed.Add(1)
	}

	// Record throughput
	l.metrics.RecordThroughput(l.config.Name)
}

// Stats returns current lane statistics.
func (l *ChannelLane) Stats() Stats {
	stats := Stats{
		Name:           l.config.Name,
		Pending:        int(l.pending.Load()),
		Running:        int(l.running.Load()),
		Completed:      l.completed.Load(),
		Failed:         l.failed.Load(),
		Dropped:        l.dropped.Load(),
		Capacity:       l.config.Capacity,
		MaxConcurrency: l.config.MaxConcurrency,
	}

	// Calculate average times
	count := l.taskCount.Load()
	if count > 0 {
		stats.ProcessTime = time.Duration(l.totalProcessTime.Load() / count)
	}

	return stats
}

// Close gracefully shuts down the lane.
func (l *ChannelLane) Close(ctx context.Context) error {
	var closeErr error

	l.closeOnce.Do(func() {
		// Mark as closed
		l.closed.Store(true)
		close(l.closeCh)

		// Stop accepting new tasks
		close(l.taskCh)

		// Wait for worker pool to finish with timeout
		done := make(chan struct{})
		go func() {
			l.workerPool.Stop()
			close(done)
		}()

		select {
		case <-done:
			// Successfully stopped
		case <-ctx.Done():
			closeErr = ctx.Err()
		}
	})

	return closeErr
}

// IsClosed returns true if the lane is closed.
func (l *ChannelLane) IsClosed() bool {
	return l.closed.Load()
}

// SetManager sets the manager for redirect strategy.
func (l *ChannelLane) SetManager(m *Manager) {
	l.manager = m
}

// SetMetrics sets the metrics recorder for the lane.
func (l *ChannelLane) SetMetrics(m MetricsRecorder) {
	if m != nil {
		l.metrics = m
	}
}

// Run starts the main loop that distributes tasks to workers.
func (l *ChannelLane) Run() {
	go func() {
		for task := range l.taskCh {
			l.workerPool.Submit(task)
		}
	}()
}

// nopMetrics is a no-op implementation of MetricsRecorder.
type nopMetrics struct{}

func (n *nopMetrics) IncQueueDepth(laneName string)                              {}
func (n *nopMetrics) DecQueueDepth(laneName string)                              {}
func (n *nopMetrics) RecordWaitDuration(laneName string, duration time.Duration) {}
func (n *nopMetrics) RecordThroughput(laneName string)                           {}
