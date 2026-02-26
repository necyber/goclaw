package dag

import (
	"fmt"
	"strings"
)

// DAGError is the base interface for all DAG-related errors.
type DAGError interface {
	error
	// TaskID returns the task ID associated with the error, if any.
	TaskID() string
}

// TaskNotFoundError is returned when a referenced task does not exist.
type TaskNotFoundError struct {
	ID string
}

func (e *TaskNotFoundError) Error() string {
	return fmt.Sprintf("task not found: %s", e.ID)
}

// TaskID returns the task ID.
func (e *TaskNotFoundError) TaskID() string {
	return e.ID
}

// DuplicateTaskError is returned when a task with the same ID is added twice.
type DuplicateTaskError struct {
	ID string
}

func (e *DuplicateTaskError) Error() string {
	return fmt.Sprintf("duplicate task ID: %s", e.ID)
}

// TaskID returns the task ID.
func (e *DuplicateTaskError) TaskID() string {
	return e.ID
}

// DependencyNotFoundError is returned when a dependency references a non-existent task.
type DependencyNotFoundError struct {
	SrcTask string // 源任务ID
	DepID   string // 依赖的任务ID
}

func (e *DependencyNotFoundError) Error() string {
	return fmt.Sprintf("task %s depends on non-existent task: %s", e.SrcTask, e.DepID)
}

// TaskID returns the task ID.
func (e *DependencyNotFoundError) TaskID() string {
	return e.SrcTask
}

// CyclicDependencyError is returned when a cycle is detected in the DAG.
type CyclicDependencyError struct {
	// Path is the cycle path (e.g., ["A", "B", "C", "A"])
	Path []string
}

func (e *CyclicDependencyError) Error() string {
	if len(e.Path) == 0 {
		return "cyclic dependency detected"
	}
	return fmt.Sprintf("cyclic dependency detected: %s", strings.Join(e.Path, " → "))
}

// TaskID returns the first task ID in the cycle.
func (e *CyclicDependencyError) TaskID() string {
	if len(e.Path) > 0 {
		return e.Path[0]
	}
	return ""
}

// SelfDependencyError is returned when a task depends on itself.
type SelfDependencyError struct {
	ID string
}

func (e *SelfDependencyError) Error() string {
	return fmt.Sprintf("task %s cannot depend on itself", e.ID)
}

// TaskID returns the task ID.
func (e *SelfDependencyError) TaskID() string {
	return e.ID
}

// InvalidEdgeError is returned when an invalid edge is added.
type InvalidEdgeError struct {
	From   string
	To     string
	Reason string
}

func (e *InvalidEdgeError) Error() string {
	return fmt.Sprintf("invalid edge from %s to %s: %s", e.From, e.To, e.Reason)
}

// TaskID returns the "from" task ID.
func (e *InvalidEdgeError) TaskID() string {
	return e.From
}
