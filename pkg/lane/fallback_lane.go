package lane

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// FallbackLane wraps a RedisLane with automatic degradation to a local ChannelLane
// when Redis becomes unavailable. It monitors Redis health in the background and
// switches back when Redis recovers.
type FallbackLane struct {
	primary   *RedisLane
	fallback  *ChannelLane
	logger    *slog.Logger

	// State: 0 = using primary (Redis), 1 = using fallback (local)
	degraded atomic.Bool

	// Background health check
	checkInterval time.Duration
	closeCh       chan struct{}
	closeOnce     sync.Once

	// Metrics
	metrics          MetricsRecorder
	degradeCount     atomic.Int64
	recoverCount     atomic.Int64
	lastDegradeTime  atomic.Int64 // unix nano
	lastRecoverTime  atomic.Int64 // unix nano
}

// FallbackConfig holds configuration for a FallbackLane.
type FallbackConfig struct {
	// CheckInterval is how often to check Redis health when degraded.
	CheckInterval time.Duration

	// Logger is the structured logger.
	Logger *slog.Logger
}

// NewFallbackLane creates a FallbackLane that wraps a RedisLane with a local ChannelLane fallback.
func NewFallbackLane(primary *RedisLane, fallback *ChannelLane, cfg *FallbackConfig) (*FallbackLane, error) {
	if primary == nil {
		return nil, fmt.Errorf("primary RedisLane cannot be nil")
	}
	if fallback == nil {
		return nil, fmt.Errorf("fallback ChannelLane cannot be nil")
	}

	checkInterval := 5 * time.Second
	var logger *slog.Logger
	if cfg != nil {
		if cfg.CheckInterval > 0 {
			checkInterval = cfg.CheckInterval
		}
		logger = cfg.Logger
	}
	if logger == nil {
		logger = slog.Default()
	}

	fl := &FallbackLane{
		primary:       primary,
		fallback:      fallback,
		logger:        logger,
		checkInterval: checkInterval,
		closeCh:       make(chan struct{}),
		metrics:       &nopMetrics{},
	}

	return fl, nil
}

// Name returns the lane name (same as the primary lane).
func (fl *FallbackLane) Name() string {
	return fl.primary.Name()
}

// Submit submits a task, using the primary lane or falling back to local.
func (fl *FallbackLane) Submit(ctx context.Context, task Task) error {
	if fl.degraded.Load() {
		return fl.fallback.Submit(ctx, task)
	}

	err := fl.tryPrimarySubmit(ctx, task)
	if err != nil && fl.isRedisError(err) {
		fl.degrade("submit error: " + err.Error())
		return fl.fallback.Submit(ctx, task)
	}
	return err
}

// tryPrimarySubmit attempts to submit via the primary lane, recovering from panics
// caused by nil Redis client or broken connections.
func (fl *FallbackLane) tryPrimarySubmit(ctx context.Context, task Task) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("redis panic: %v", r)
		}
	}()
	return fl.primary.Submit(ctx, task)
}

// TrySubmit attempts a non-blocking submit.
func (fl *FallbackLane) TrySubmit(task Task) bool {
	if fl.degraded.Load() {
		return fl.fallback.TrySubmit(task)
	}

	ok, panicked := fl.tryPrimaryTrySubmit(task)
	if panicked {
		fl.degrade("TrySubmit panic")
		return fl.fallback.TrySubmit(task)
	}
	return ok
}

func (fl *FallbackLane) tryPrimaryTrySubmit(task Task) (ok bool, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	return fl.primary.TrySubmit(task), false
}

// Stats returns combined statistics.
func (fl *FallbackLane) Stats() Stats {
	if fl.degraded.Load() {
		stats := fl.fallback.Stats()
		stats.Name = fl.primary.Name()
		return stats
	}
	return fl.primary.Stats()
}

// Close gracefully shuts down both lanes and stops the health checker.
func (fl *FallbackLane) Close(ctx context.Context) error {
	var errs []error
	fl.closeOnce.Do(func() {
		close(fl.closeCh)
	})

	if err := fl.primary.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("primary close: %w", err))
	}
	if err := fl.fallback.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("fallback close: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("fallback lane close errors: %v", errs)
	}
	return nil
}

// IsClosed returns true if the lane is closed.
func (fl *FallbackLane) IsClosed() bool {
	return fl.primary.IsClosed() && fl.fallback.IsClosed()
}

// SetManager sets the manager on both lanes.
func (fl *FallbackLane) SetManager(m *Manager) {
	fl.primary.SetManager(m)
	fl.fallback.SetManager(m)
}

// SetMetrics sets the metrics recorder on both lanes.
func (fl *FallbackLane) SetMetrics(m MetricsRecorder) {
	if m != nil {
		fl.metrics = m
		fl.primary.SetMetrics(m)
		fl.fallback.SetMetrics(m)
	}
}

// Run starts both lanes and the background health checker.
func (fl *FallbackLane) Run() {
	fl.primary.Run()
	fl.fallback.Run()
	go fl.healthCheckLoop()
}

// IsDegraded returns true if currently using the fallback lane.
func (fl *FallbackLane) IsDegraded() bool {
	return fl.degraded.Load()
}

// DegradeCount returns the number of times the lane has degraded.
func (fl *FallbackLane) DegradeCount() int64 {
	return fl.degradeCount.Load()
}

// RecoverCount returns the number of times the lane has recovered.
func (fl *FallbackLane) RecoverCount() int64 {
	return fl.recoverCount.Load()
}

func (fl *FallbackLane) degrade(reason string) {
	if fl.degraded.CompareAndSwap(false, true) {
		fl.degradeCount.Add(1)
		fl.lastDegradeTime.Store(time.Now().UnixNano())
		fl.logger.Warn("Redis lane degraded to local fallback",
			"lane", fl.primary.Name(),
			"reason", reason,
		)
	}
}

func (fl *FallbackLane) recover() {
	if fl.degraded.CompareAndSwap(true, false) {
		fl.recoverCount.Add(1)
		fl.lastRecoverTime.Store(time.Now().UnixNano())
		fl.logger.Info("Redis lane recovered from fallback",
			"lane", fl.primary.Name(),
		)
	}
}

func (fl *FallbackLane) healthCheckLoop() {
	ticker := time.NewTicker(fl.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fl.closeCh:
			return
		case <-ticker.C:
			if !fl.degraded.Load() {
				// Not degraded — do a quick health check to detect failures
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				err := PingRedis(ctx, fl.primary.client)
				cancel()
				if err != nil {
					fl.degrade("health check failed: " + err.Error())
				}
				continue
			}

			// Degraded — check if Redis is back
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := PingRedis(ctx, fl.primary.client)
			cancel()
			if err == nil {
				fl.recover()
			}
		}
	}
}

// isRedisError checks if an error is likely a Redis connectivity issue
// rather than a normal lane error (full, closed, etc.).
func (fl *FallbackLane) isRedisError(err error) bool {
	if err == nil {
		return false
	}
	// Lane-level errors are not Redis errors
	if IsLaneClosedError(err) || IsTaskDroppedError(err) || IsLaneFullError(err) {
		return false
	}
	// Context errors are not Redis errors
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}
	return true
}
