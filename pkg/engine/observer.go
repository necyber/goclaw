package engine

import (
	"sync"
)

// WorkflowEvent represents a workflow state change event
type WorkflowEvent struct {
	WorkflowID string
	EventType  WorkflowEventType
	Status     string
	Message    string
	Timestamp  int64
}

// WorkflowEventType represents the type of workflow event
type WorkflowEventType int

const (
	WorkflowEventSubmitted WorkflowEventType = iota
	WorkflowEventStarted
	WorkflowEventCompleted
	WorkflowEventFailed
	WorkflowEventCancelled
)

// String returns the string representation of WorkflowEventType
func (t WorkflowEventType) String() string {
	switch t {
	case WorkflowEventSubmitted:
		return "SUBMITTED"
	case WorkflowEventStarted:
		return "STARTED"
	case WorkflowEventCompleted:
		return "COMPLETED"
	case WorkflowEventFailed:
		return "FAILED"
	case WorkflowEventCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

// TaskEvent represents a task state change event
type TaskEvent struct {
	WorkflowID string
	TaskID     string
	EventType  TaskEventType
	Status     string
	Progress   int
	Message    string
	Timestamp  int64
}

// TaskEventType represents the type of task event
type TaskEventType int

const (
	TaskEventStarted TaskEventType = iota
	TaskEventProgress
	TaskEventCompleted
	TaskEventFailed
	TaskEventCancelled
)

// String returns the string representation of TaskEventType
func (t TaskEventType) String() string {
	switch t {
	case TaskEventStarted:
		return "STARTED"
	case TaskEventProgress:
		return "PROGRESS"
	case TaskEventCompleted:
		return "COMPLETED"
	case TaskEventFailed:
		return "FAILED"
	case TaskEventCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

// WorkflowObserver receives workflow state change notifications
type WorkflowObserver interface {
	OnWorkflowEvent(event WorkflowEvent)
	OnTaskEvent(event TaskEvent)
}

// ObserverRegistry manages workflow observers
type ObserverRegistry struct {
	mu        sync.RWMutex
	observers map[string][]WorkflowObserver // workflowID -> observers
	global    []WorkflowObserver            // observers for all workflows
}

// NewObserverRegistry creates a new observer registry
func NewObserverRegistry() *ObserverRegistry {
	return &ObserverRegistry{
		observers: make(map[string][]WorkflowObserver),
		global:    make([]WorkflowObserver, 0),
	}
}

// Subscribe adds an observer for a specific workflow
func (r *ObserverRegistry) Subscribe(workflowID string, observer WorkflowObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.observers[workflowID] = append(r.observers[workflowID], observer)
}

// SubscribeGlobal adds an observer for all workflows
func (r *ObserverRegistry) SubscribeGlobal(observer WorkflowObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.global = append(r.global, observer)
}

// Unsubscribe removes an observer for a specific workflow
func (r *ObserverRegistry) Unsubscribe(workflowID string, observer WorkflowObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()

	observers := r.observers[workflowID]
	for i, obs := range observers {
		if obs == observer {
			r.observers[workflowID] = append(observers[:i], observers[i+1:]...)
			break
		}
	}

	// Clean up empty entries
	if len(r.observers[workflowID]) == 0 {
		delete(r.observers, workflowID)
	}
}

// UnsubscribeGlobal removes a global observer
func (r *ObserverRegistry) UnsubscribeGlobal(observer WorkflowObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, obs := range r.global {
		if obs == observer {
			r.global = append(r.global[:i], r.global[i+1:]...)
			break
		}
	}
}

// NotifyWorkflowEvent notifies all observers of a workflow event
func (r *ObserverRegistry) NotifyWorkflowEvent(event WorkflowEvent) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Notify workflow-specific observers
	for _, observer := range r.observers[event.WorkflowID] {
		go observer.OnWorkflowEvent(event)
	}

	// Notify global observers
	for _, observer := range r.global {
		go observer.OnWorkflowEvent(event)
	}
}

// NotifyTaskEvent notifies all observers of a task event
func (r *ObserverRegistry) NotifyTaskEvent(event TaskEvent) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Notify workflow-specific observers
	for _, observer := range r.observers[event.WorkflowID] {
		go observer.OnTaskEvent(event)
	}

	// Notify global observers
	for _, observer := range r.global {
		go observer.OnTaskEvent(event)
	}
}

// GetObserverCount returns the number of observers for a workflow
func (r *ObserverRegistry) GetObserverCount(workflowID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.observers[workflowID])
}

// GetGlobalObserverCount returns the number of global observers
func (r *ObserverRegistry) GetGlobalObserverCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.global)
}
