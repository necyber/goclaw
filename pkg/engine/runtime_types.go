package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/goclaw/goclaw/pkg/storage"
)

const (
	workflowStatusPending   = "pending"
	workflowStatusScheduled = "scheduled"
	workflowStatusRunning   = "running"
	workflowStatusCompleted = "completed"
	workflowStatusFailed    = "failed"
	workflowStatusCancelled = "cancelled"

	taskStatusPending   = "pending"
	taskStatusScheduled = "scheduled"
	taskStatusRunning   = "running"
	taskStatusCompleted = "completed"
	taskStatusFailed    = "failed"
	taskStatusCancelled = "cancelled"
)

// SubmissionMode controls runtime submit semantics.
type SubmissionMode string

const (
	SubmissionModeSync  SubmissionMode = "sync"
	SubmissionModeAsync SubmissionMode = "async"
)

type workflowExecution struct {
	workflowID string
	cancel     context.CancelFunc
	done       chan struct{}
	mu         sync.Mutex
	wfState    *storage.WorkflowState
}

var allowedWorkflowTransitions = map[string]map[string]struct{}{
	workflowStatusPending: {
		workflowStatusScheduled: {},
		workflowStatusFailed:    {},
		workflowStatusCancelled: {},
	},
	workflowStatusScheduled: {
		workflowStatusRunning:   {},
		workflowStatusFailed:    {},
		workflowStatusCancelled: {},
	},
	workflowStatusRunning: {
		workflowStatusCompleted: {},
		workflowStatusFailed:    {},
		workflowStatusCancelled: {},
	},
}

var allowedTaskTransitions = map[string]map[string]struct{}{
	taskStatusPending: {
		taskStatusScheduled: {},
		taskStatusFailed:    {},
		taskStatusCancelled: {},
	},
	taskStatusScheduled: {
		taskStatusRunning:   {},
		taskStatusFailed:    {},
		taskStatusCancelled: {},
	},
	taskStatusRunning: {
		taskStatusScheduled: {},
		taskStatusCompleted: {},
		taskStatusFailed:    {},
		taskStatusCancelled: {},
	},
}

func isTerminalWorkflowStatus(status string) bool {
	return status == workflowStatusCompleted || status == workflowStatusFailed || status == workflowStatusCancelled
}

func isTerminalTaskStatus(status string) bool {
	return status == taskStatusCompleted || status == taskStatusFailed || status == taskStatusCancelled
}

func validateWorkflowTransition(oldStatus, newStatus string) error {
	if oldStatus == "" && newStatus == workflowStatusPending {
		return nil
	}
	if oldStatus == newStatus {
		return nil
	}
	if isTerminalWorkflowStatus(oldStatus) {
		return fmt.Errorf("illegal workflow transition %q -> %q: terminal state is immutable", oldStatus, newStatus)
	}
	allowed, ok := allowedWorkflowTransitions[oldStatus]
	if !ok {
		return fmt.Errorf("illegal workflow transition %q -> %q", oldStatus, newStatus)
	}
	if _, ok := allowed[newStatus]; !ok {
		return fmt.Errorf("illegal workflow transition %q -> %q", oldStatus, newStatus)
	}
	return nil
}

func validateTaskTransition(oldStatus, newStatus string) error {
	if oldStatus == "" && newStatus == taskStatusPending {
		return nil
	}
	if oldStatus == newStatus {
		return nil
	}
	if isTerminalTaskStatus(oldStatus) {
		return fmt.Errorf("illegal task transition %q -> %q: terminal state is immutable", oldStatus, newStatus)
	}
	allowed, ok := allowedTaskTransitions[oldStatus]
	if !ok {
		return fmt.Errorf("illegal task transition %q -> %q", oldStatus, newStatus)
	}
	if _, ok := allowed[newStatus]; !ok {
		return fmt.Errorf("illegal task transition %q -> %q", oldStatus, newStatus)
	}
	return nil
}

func (e *Engine) registerExecution(exec *workflowExecution) {
	e.execMu.Lock()
	defer e.execMu.Unlock()
	e.executions[exec.workflowID] = exec
}

func (e *Engine) getExecution(workflowID string) (*workflowExecution, bool) {
	e.execMu.RLock()
	defer e.execMu.RUnlock()
	exec, ok := e.executions[workflowID]
	return exec, ok
}

func (e *Engine) unregisterExecution(workflowID string) {
	e.execMu.Lock()
	defer e.execMu.Unlock()
	delete(e.executions, workflowID)
}
