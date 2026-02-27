package engine

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const runtimeTracerName = "goclaw.runtime"

const (
	spanWorkflowExecute = "workflow.execute"
	spanWorkflowLayer   = "workflow.layer"
	spanTaskSchedule    = "workflow.task.schedule"
	spanTaskRun         = "workflow.task.run"
	spanLaneWait        = "lane.wait"
)

func runtimeTracer() trace.Tracer {
	return otel.Tracer(runtimeTracerName)
}
