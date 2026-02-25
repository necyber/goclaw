package engine

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
