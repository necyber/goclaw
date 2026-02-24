// Package dag provides Directed Acyclic Graph (DAG) functionality for workflow orchestration.
package dag

import (
	"fmt"
	"time"
)

// TaskStatus represents the execution status of a task.
type TaskStatus int

const (
	// TaskPending indicates the task is waiting to be scheduled.
	TaskPending TaskStatus = iota
	// TaskScheduled indicates the task has been scheduled for execution.
	TaskScheduled
	// TaskRunning indicates the task is currently executing.
	TaskRunning
	// TaskCompleted indicates the task completed successfully.
	TaskCompleted
	// TaskFailed indicates the task failed during execution.
	TaskFailed
	// TaskCancelled indicates the task was cancelled.
	TaskCancelled
)

// String returns the string representation of TaskStatus.
func (s TaskStatus) String() string {
	switch s {
	case TaskPending:
		return "pending"
	case TaskScheduled:
		return "scheduled"
	case TaskRunning:
		return "running"
	case TaskCompleted:
		return "completed"
	case TaskFailed:
		return "failed"
	case TaskCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Task represents a unit of work in the workflow.
type Task struct {
	// ID is the unique identifier for the task.
	ID string `json:"id" yaml:"id"`

	// Name is a human-readable name for the task.
	Name string `json:"name" yaml:"name"`

	// Agent is the type of agent that should execute this task.
	Agent string `json:"agent" yaml:"agent"`

	// Lane is the resource lane this task should run in (e.g., "cpu", "io", "gpu").
	Lane string `json:"lane" yaml:"lane"`

	// Deps is a list of task IDs that this task depends on.
	Deps []string `json:"deps,omitempty" yaml:"deps,omitempty"`

	// Timeout is the maximum duration allowed for task execution.
	// Zero means no timeout.
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Retries is the number of times to retry on failure.
	// Zero means no retries.
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`

	// Metadata contains arbitrary key-value pairs for the task.
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Input defines the expected input schema (optional).
	Input interface{} `json:"input,omitempty" yaml:"input,omitempty"`

	// Output defines the expected output schema (optional).
	Output interface{} `json:"output,omitempty" yaml:"output,omitempty"`
}

// Validate checks if the task definition is valid.
func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if t.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if t.Agent == "" {
		return fmt.Errorf("task agent cannot be empty")
	}
	if t.Timeout < 0 {
		return fmt.Errorf("task timeout cannot be negative")
	}
	if t.Retries < 0 {
		return fmt.Errorf("task retries cannot be negative")
	}
	return nil
}

// Clone creates a deep copy of the task.
func (t *Task) Clone() *Task {
	cloned := &Task{
		ID:       t.ID,
		Name:     t.Name,
		Agent:    t.Agent,
		Lane:     t.Lane,
		Timeout:  t.Timeout,
		Retries:  t.Retries,
		Input:    t.Input,
		Output:   t.Output,
	}

	if t.Deps != nil {
		cloned.Deps = make([]string, len(t.Deps))
		copy(cloned.Deps, t.Deps)
	}

	if t.Metadata != nil {
		cloned.Metadata = make(map[string]string, len(t.Metadata))
		for k, v := range t.Metadata {
			cloned.Metadata[k] = v
		}
	}

	return cloned
}

// String returns a string representation of the task.
func (t *Task) String() string {
	return fmt.Sprintf("Task{ID: %s, Name: %s, Agent: %s, Deps: %v}",
		t.ID, t.Name, t.Agent, t.Deps)
}

// HasDependency checks if the task depends on the given task ID.
func (t *Task) HasDependency(taskID string) bool {
	for _, dep := range t.Deps {
		if dep == taskID {
			return true
		}
	}
	return false
}

// AddDependency adds a dependency to the task.
func (t *Task) AddDependency(taskID string) error {
	if taskID == t.ID {
		return fmt.Errorf("task cannot depend on itself")
	}
	if t.HasDependency(taskID) {
		return fmt.Errorf("task already depends on %s", taskID)
	}
	t.Deps = append(t.Deps, taskID)
	return nil
}

// RemoveDependency removes a dependency from the task.
func (t *Task) RemoveDependency(taskID string) bool {
	for i, dep := range t.Deps {
		if dep == taskID {
			t.Deps = append(t.Deps[:i], t.Deps[i+1:]...)
			return true
		}
	}
	return false
}
