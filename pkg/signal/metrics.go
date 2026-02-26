package signal

import (
	"sync"
	"time"
)

// MetricsRecorder defines metrics hooks for signal operations.
type MetricsRecorder interface {
	RecordSignalSent(mode string, signalType string)
	RecordSignalReceived(mode string, signalType string)
	RecordSignalFailed(mode string, signalType string, reason string)
	RecordSignalPattern(pattern string, status string, duration time.Duration)
}

type nopMetrics struct{}

func (n *nopMetrics) RecordSignalSent(mode string, signalType string)                           {}
func (n *nopMetrics) RecordSignalReceived(mode string, signalType string)                       {}
func (n *nopMetrics) RecordSignalFailed(mode string, signalType string, reason string)          {}
func (n *nopMetrics) RecordSignalPattern(pattern string, status string, duration time.Duration) {}

var (
	metricsMu sync.RWMutex
	metrics   MetricsRecorder = &nopMetrics{}
)

// SetMetricsRecorder sets the package-level signal metrics recorder.
func SetMetricsRecorder(recorder MetricsRecorder) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	if recorder == nil {
		metrics = &nopMetrics{}
		return
	}
	metrics = recorder
}

func metricsRecorder() MetricsRecorder {
	metricsMu.RLock()
	defer metricsMu.RUnlock()
	if metrics == nil {
		return &nopMetrics{}
	}
	return metrics
}
