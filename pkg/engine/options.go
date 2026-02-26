package engine

import "github.com/goclaw/goclaw/pkg/signal"

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
