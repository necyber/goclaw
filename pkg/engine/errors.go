package engine

import "fmt"

// WorkflowCompileError is returned when a workflow DAG fails to compile.
type WorkflowCompileError struct {
	WorkflowID string
	Cause      error
}

func (e *WorkflowCompileError) Error() string {
	return fmt.Sprintf("workflow %q compile error: %v", e.WorkflowID, e.Cause)
}

func (e *WorkflowCompileError) Unwrap() error { return e.Cause }

// TaskExecutionError is returned when a task fails after all retries.
type TaskExecutionError struct {
	TaskID  string
	Retries int
	Cause   error
}

func (e *TaskExecutionError) Error() string {
	return fmt.Sprintf("task %q failed after %d retries: %v", e.TaskID, e.Retries, e.Cause)
}

func (e *TaskExecutionError) Unwrap() error { return e.Cause }

// EngineNotRunningError is returned when an operation requires the engine to be running.
type EngineNotRunningError struct{}

func (e *EngineNotRunningError) Error() string {
	return "engine is not running"
}
