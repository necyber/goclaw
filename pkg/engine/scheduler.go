package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/signal"
)

// Scheduler executes an ExecutionPlan layer by layer.
type Scheduler struct {
	tracker   *StateTracker
	logger    appLogger
	signalBus signal.Bus
}

// newScheduler creates a new Scheduler.
func newScheduler(tracker *StateTracker, logger appLogger, bus signal.Bus) *Scheduler {
	return &Scheduler{tracker: tracker, logger: logger, signalBus: bus}
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

// Schedule executes the plan layer by layer.
// All tasks within a layer run concurrently; the next layer starts only after
// every task in the current layer has completed (fail-fast on first error).
func (s *Scheduler) Schedule(ctx context.Context, plan *dag.ExecutionPlan, taskFns map[string]func(context.Context) error) error {
	for layerIdx, layer := range plan.Layers {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		s.logger.Debug("scheduling layer", "layer", layerIdx, "tasks", layer)

		var (
			wg       sync.WaitGroup
			mu       sync.Mutex
			firstErr error
		)

		for _, taskID := range layer {
			taskID := taskID

			dagTask, ok := plan.GetTask(taskID)
			if !ok {
				return fmt.Errorf("task %q not found in execution plan", taskID)
			}

			fn := taskFns[taskID]
			runner := newTaskRunner(dagTask, s.tracker, fn)
			s.tracker.SetState(taskID, TaskStateScheduled)

			wg.Add(1)
			go func() {
				defer wg.Done()
				taskCtx, cleanup := s.attachSignalChannel(ctx, taskID)
				if cleanup != nil {
					defer cleanup()
				}
				// Execute directly in this goroutine so we can wait for completion.
				// The lane.Manager is used for resource-constrained workloads in
				// future phases; for now direct execution gives us synchronous results.
				if err := runner.Execute(taskCtx); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		if firstErr != nil {
			return firstErr
		}

		s.logger.Debug("layer complete", "layer", layerIdx)
	}

	return nil
}
