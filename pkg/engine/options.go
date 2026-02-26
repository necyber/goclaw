package engine

import (
	"github.com/goclaw/goclaw/pkg/signal"
	"github.com/redis/go-redis/v9"
)

// Option is a functional option for configuring the Engine.
type Option func(*Engine)

// WithMetrics sets the metrics recorder for the engine.
func WithMetrics(metrics MetricsRecorder) Option {
	return func(e *Engine) {
		if metrics != nil {
			e.metrics = metrics
		}
	}
}

// WithMemoryHub sets the memory hub for the engine.
func WithMemoryHub(hub MemoryHub) Option {
	return func(e *Engine) {
		if hub != nil {
			e.memoryHub = hub
		}
	}
}

// WithSignalBus sets the signal bus for the engine.
func WithSignalBus(bus signal.Bus) Option {
	return func(e *Engine) {
		if bus != nil {
			e.signalBus = bus
		}
	}
}

// WithRedisClient sets the shared Redis client used by Redis-backed lanes.
func WithRedisClient(client redis.Cmdable) Option {
	return func(e *Engine) {
		if client != nil {
			e.redisClient = client
		}
	}
}

// WithEventBroadcaster sets an event broadcaster for workflow/task state changes.
func WithEventBroadcaster(broadcaster EventBroadcaster) Option {
	return func(e *Engine) {
		if broadcaster != nil {
			e.events = broadcaster
		}
	}
}
