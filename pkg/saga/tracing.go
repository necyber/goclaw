package saga

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const sagaTracerName = "goclaw.saga"

const (
	spanSagaExecuteForward    = "saga.execute.forward"
	spanSagaStepForward       = "saga.step.forward"
	spanSagaExecuteCompensate = "saga.execute.compensation"
	spanSagaStepCompensate    = "saga.step.compensate"
	spanSagaRecoveryResume    = "saga.recovery.resume"
)

func sagaTracer() trace.Tracer {
	return otel.Tracer(sagaTracerName)
}
