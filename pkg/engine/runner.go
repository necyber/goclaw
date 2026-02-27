package engine

import (
	"context"
	"errors"
	"time"

	"github.com/goclaw/goclaw/pkg/dag"
)

// taskRunner wraps a dag.Task to implement the lane.Task interface,
// and drives execution with retry logic.
type taskRunner struct {
	task    *dag.Task
	tracker *StateTracker
	fn      func(ctx context.Context) error
}

// newTaskRunner creates a taskRunner for the given dag.Task.
// fn is the actual work function; pass nil to use a no-op (useful in tests).
func newTaskRunner(task *dag.Task, tracker *StateTracker, fn func(ctx context.Context) error) *taskRunner {
	if fn == nil {
		fn = func(ctx context.Context) error { return nil }
	}
	return &taskRunner{task: task, tracker: tracker, fn: fn}
}

// ID implements lane.Task.
func (r *taskRunner) ID() string { return r.task.ID }

// Priority implements lane.Task.
func (r *taskRunner) Priority() int { return 1 }

// Lane implements lane.Task.
func (r *taskRunner) Lane() string {
	if r.task.Lane == "" {
		return defaultLaneName
	}
	return r.task.Lane
}

// Execute runs the task function with retry logic and updates the StateTracker.
func (r *taskRunner) Execute(ctx context.Context) error {
	maxAttempts := r.task.Retries + 1
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			r.tracker.SetState(r.task.ID, TaskStateRetrying)
		}
		r.tracker.SetState(r.task.ID, TaskStateRunning)

		// Apply per-task timeout if configured.
		runCtx := ctx
		var cancel context.CancelFunc
		if r.task.Timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, r.task.Timeout)
		}

		lastErr = r.fn(runCtx)

		if cancel != nil {
			cancel()
		}

		if lastErr == nil {
			if ctx.Err() != nil {
				lastErr = ctx.Err()
				break
			}
			r.tracker.SetState(r.task.ID, TaskStateCompleted)
			return nil
		}

		// Check if context was cancelled - no point retrying.
		if ctx.Err() != nil {
			break
		}

		// Back off briefly between retries (simple fixed delay).
		if attempt < maxAttempts-1 {
			select {
			case <-ctx.Done():
				lastErr = ctx.Err()
				goto done
			case <-time.After(100 * time.Millisecond):
			}
		}
	}

done:
	if errors.Is(lastErr, context.Canceled) || errors.Is(lastErr, context.DeadlineExceeded) || ctx.Err() != nil {
		r.tracker.SetState(r.task.ID, TaskStateCancelled)
		return &TaskExecutionError{TaskID: r.task.ID, Retries: r.task.Retries, Cause: lastErr}
	}
	r.tracker.SetFailed(r.task.ID, lastErr, r.task.Retries)
	return &TaskExecutionError{TaskID: r.task.ID, Retries: r.task.Retries, Cause: lastErr}
}
