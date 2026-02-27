package saga

import "time"

// MetricsRecorder records saga runtime metrics.
type MetricsRecorder interface {
	RecordSagaExecution(status string)
	RecordSagaDuration(status string, duration time.Duration)
	IncActiveSagas()
	DecActiveSagas()
	RecordCompensation(status string)
	RecordCompensationDuration(duration time.Duration)
	RecordCompensationRetry()
	RecordSagaRecovery(status string)
}

type nopMetricsRecorder struct{}

func (n *nopMetricsRecorder) RecordSagaExecution(status string)                        {}
func (n *nopMetricsRecorder) RecordSagaDuration(status string, duration time.Duration) {}
func (n *nopMetricsRecorder) IncActiveSagas()                                          {}
func (n *nopMetricsRecorder) DecActiveSagas()                                          {}
func (n *nopMetricsRecorder) RecordCompensation(status string)                         {}
func (n *nopMetricsRecorder) RecordCompensationDuration(duration time.Duration)        {}
func (n *nopMetricsRecorder) RecordCompensationRetry()                                 {}
func (n *nopMetricsRecorder) RecordSagaRecovery(status string)                         {}
