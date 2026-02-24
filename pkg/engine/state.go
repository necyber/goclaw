package engine

import (
	"sync"
	"time"
)

// TaskState represents the execution state of a task.
type TaskState int

const (
	TaskStatePending TaskState = iota
	TaskStateScheduled
	TaskStateRunning
	TaskStateCompleted
	TaskStateFailed
)

// String returns the string representation of TaskState.
func (s TaskState) String() string {
	switch s {
	case TaskStatePending:
		return "pending"
	case TaskStateScheduled:
		return "scheduled"
	case TaskStateRunning:
		return "running"
	case TaskStateCompleted:
		return "completed"
	case TaskStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// TaskResult holds the execution result of a single task.
type TaskResult struct {
	TaskID    string
	State     TaskState
	Error     error
	StartedAt time.Time
	EndedAt   time.Time
	Retries   int
}

// StateTracker tracks the state of all tasks in a workflow execution.
type StateTracker struct {
	mu      sync.RWMutex
	results map[string]*TaskResult
}

// newStateTracker creates a new StateTracker.
func newStateTracker() *StateTracker {
	return &StateTracker{
		results: make(map[string]*TaskResult),
	}
}

// InitTasks initialises all given task IDs to TaskStatePending.
func (t *StateTracker) InitTasks(taskIDs []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, id := range taskIDs {
		t.results[id] = &TaskResult{
			TaskID: id,
			State:  TaskStatePending,
		}
	}
}

// SetState updates the state of a task.
func (t *StateTracker) SetState(taskID string, state TaskState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	r, ok := t.results[taskID]
	if !ok {
		r = &TaskResult{TaskID: taskID}
		t.results[taskID] = r
	}
	r.State = state
	switch state {
	case TaskStateRunning:
		r.StartedAt = time.Now()
	case TaskStateCompleted, TaskStateFailed:
		r.EndedAt = time.Now()
	}
}

// SetFailed marks a task as failed with the given error and retry count.
func (t *StateTracker) SetFailed(taskID string, err error, retries int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	r, ok := t.results[taskID]
	if !ok {
		r = &TaskResult{TaskID: taskID}
		t.results[taskID] = r
	}
	r.State = TaskStateFailed
	r.Error = err
	r.Retries = retries
	r.EndedAt = time.Now()
}

// GetResult returns a copy of the TaskResult for the given task ID.
func (t *StateTracker) GetResult(taskID string) (*TaskResult, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, ok := t.results[taskID]
	if !ok {
		return nil, false
	}
	copy := *r
	return &copy, true
}

// Results returns a snapshot of all task results.
func (t *StateTracker) Results() map[string]*TaskResult {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make(map[string]*TaskResult, len(t.results))
	for k, v := range t.results {
		copy := *v
		out[k] = &copy
	}
	return out
}
