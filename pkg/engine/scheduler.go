package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/lane"
	"github.com/goclaw/goclaw/pkg/signal"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Scheduler executes an ExecutionPlan layer by layer.
type Scheduler struct {
	tracker     *StateTracker
	logger      appLogger
	signalBus   signal.Bus
	laneManager *lane.Manager
}

// newScheduler creates a new Scheduler.
func newScheduler(tracker *StateTracker, logger appLogger, bus signal.Bus, laneManager *lane.Manager) *Scheduler {
	return &Scheduler{tracker: tracker, logger: logger, signalBus: bus, laneManager: laneManager}
}

func (s *Scheduler) attachSignalChannel(ctx context.Context, taskID string) (context.Context, func()) {
	if s.signalBus == nil {
		return ctx, nil
	}

	ch, err := s.signalBus.Subscribe(ctx, taskID)
	if err != nil {
		s.logger.Warn("failed to subscribe signal channel", "task_id", taskID, "error", err)
		return ctx, nil
	}

	return signal.WithSignalChannel(ctx, ch), func() {
		if err := s.signalBus.Unsubscribe(taskID); err != nil {
			s.logger.Warn("failed to unsubscribe signal channel", "task_id", taskID, "error", err)
		}
	}
}

type scheduledTaskResult struct {
	taskID string
	err    error
}

// Schedule executes the plan layer by layer.
// All tasks within a layer run concurrently; the next layer starts only after
// every task in the current layer has completed.
func (s *Scheduler) Schedule(ctx context.Context, plan *dag.ExecutionPlan, taskFns map[string]func(context.Context) error) error {
	if s.laneManager == nil {
		return fmt.Errorf("lane manager is not configured")
	}

	for layerIdx, layer := range plan.Layers {
		layerCtx, layerSpan := runtimeTracer().Start(ctx, spanWorkflowLayer)
		layerSpan.SetAttributes(
			attribute.Int("workflow.layer_index", layerIdx),
			attribute.Int("workflow.layer_size", len(layer)),
		)

		if ctx.Err() != nil {
			for _, taskID := range layer {
				s.tracker.SetState(taskID, TaskStateCancelled)
			}
			layerSpan.RecordError(ctx.Err())
			layerSpan.SetStatus(otelcodes.Error, "cancelled")
			layerSpan.End()
			return ctx.Err()
		}

		s.logger.Debug("scheduling layer", "layer", layerIdx, "tasks", layer)

		resultCh := make(chan scheduledTaskResult, len(layer))
		submitted := 0
		firstErr := error(nil)

		for idx, taskID := range layer {
			if ctx.Err() != nil {
				for _, remainingTaskID := range layer[idx:] {
					s.tracker.SetState(remainingTaskID, TaskStateCancelled)
				}
				firstErr = ctx.Err()
				break
			}

			dagTask, ok := plan.GetTask(taskID)
			if !ok {
				for _, remainingTaskID := range layer[idx:] {
					s.tracker.SetState(remainingTaskID, TaskStateFailed)
				}
				firstErr = fmt.Errorf("task %q not found in execution plan", taskID)
				break
			}

			fn := taskFns[taskID]
			runner := newTaskRunner(dagTask, s.tracker, fn)
			s.tracker.SetState(taskID, TaskStateScheduled)

			submitCtx, submitSpan := runtimeTracer().Start(layerCtx, spanTaskSchedule)
			submitSpan.SetAttributes(
				attribute.String("task.id", taskID),
				attribute.String("lane.name", runner.Lane()),
				attribute.Int("workflow.layer_index", layerIdx),
			)
			submittedAt := time.Now()

			laneTask := lane.NewTaskFunc(taskID, runner.Lane(), runner.Priority(), func(_ context.Context) error {
				taskCtx, cleanup := s.attachSignalChannel(submitCtx, taskID)
				if cleanup != nil {
					defer cleanup()
				}

				waitCtx, waitSpan := runtimeTracer().Start(
					taskCtx,
					spanLaneWait,
					trace.WithTimestamp(submittedAt),
				)
				waitSpan.SetAttributes(
					attribute.String("task.id", taskID),
					attribute.String("lane.name", runner.Lane()),
				)
				waitSpan.SetStatus(otelcodes.Ok, "ok")
				waitSpan.End()

				err := runner.Execute(waitCtx)
				resultCh <- scheduledTaskResult{taskID: taskID, err: err}
				return err
			})

			if err := s.laneManager.Submit(ctx, laneTask); err != nil {
				submitSpan.RecordError(err)
				submitSpan.SetStatus(otelcodes.Error, "submit_failed")
				submitSpan.End()
				s.tracker.SetFailed(taskID, err, dagTask.Retries)
				for _, remainingTaskID := range layer[idx+1:] {
					s.tracker.SetState(remainingTaskID, TaskStateCancelled)
				}
				firstErr = fmt.Errorf("lane submit failed for task %s: %w", taskID, err)
				break
			}
			submitSpan.SetStatus(otelcodes.Ok, "submitted")
			submitSpan.End()
			submitted++
		}

		for i := 0; i < submitted; i++ {
			res := <-resultCh
			if firstErr == nil && res.err != nil {
				firstErr = res.err
			}
		}

		if firstErr != nil {
			layerSpan.RecordError(firstErr)
			layerSpan.SetStatus(otelcodes.Error, "layer_failed")
			layerSpan.End()
			return firstErr
		}

		s.logger.Debug("layer complete", "layer", layerIdx)
		layerSpan.SetStatus(otelcodes.Ok, "completed")
		layerSpan.End()
	}

	return nil
}
