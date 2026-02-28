// Package lane provides the Lane Queue system for task scheduling and resource management.
//
// A Lane represents a resource-constrained execution queue with:
//   - Buffered Channel for task queuing
//   - Worker Pool for concurrent execution
//   - Rate Limiting for flow control
//   - Backpressure strategies for overload handling
//
// Basic usage:
//
//	config := &lane.Config{
//	    Name:           "cpu",
//	    Capacity:       100,
//	    MaxConcurrency: 8,
//	    Backpressure:   lane.Block,
//	}
//	l, err := lane.New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer l.Close(context.Background())
//
//	// Submit a task
//	task := lane.NewTaskFunc("task-1", "cpu", 1, func(ctx context.Context) error {
//	    // Do work
//	    return nil
//	})
//	err = l.Submit(context.Background(), task)
package lane

import (
	"context"
	"fmt"
	"time"
)

// Task represents a unit of work that can be submitted to a Lane.
type Task interface {
	// ID returns the unique identifier of the task.
	ID() string

	// Priority returns the priority of the task (higher = more important).
	Priority() int

	// Lane returns the lane name this task should be executed in.
	Lane() string
}

// TaskFunc is a function type that implements the Task interface.
type TaskFunc struct {
	id       string
	priority int
	lane     string
	fn       func(ctx context.Context) error
}

// NewTaskFunc creates a new TaskFunc.
func NewTaskFunc(id, lane string, priority int, fn func(ctx context.Context) error) *TaskFunc {
	return &TaskFunc{
		id:       id,
		lane:     lane,
		priority: priority,
		fn:       fn,
	}
}

// ID implements Task.ID.
func (t *TaskFunc) ID() string {
	return t.id
}

// Priority implements Task.Priority.
func (t *TaskFunc) Priority() int {
	return t.priority
}

// Lane implements Task.Lane.
func (t *TaskFunc) Lane() string {
	return t.lane
}

// Execute executes the task function.
func (t *TaskFunc) Execute(ctx context.Context) error {
	if t.fn == nil {
		return fmt.Errorf("task function is nil")
	}
	return t.fn(ctx)
}

// BackpressureStrategy defines how to handle overload situations.
type BackpressureStrategy int

const (
	// Block blocks the submitter until space is available.
	Block BackpressureStrategy = iota
	// Drop drops the new task when the queue is full.
	Drop
	// Redirect redirects the task to another lane or handler.
	Redirect
)

// String returns the string representation of BackpressureStrategy.
func (s BackpressureStrategy) String() string {
	switch s {
	case Block:
		return "block"
	case Drop:
		return "drop"
	case Redirect:
		return "redirect"
	default:
		return "unknown"
	}
}

// Config holds the configuration for a Lane.
type Config struct {
	// Name is the unique name of the lane.
	Name string

	// Capacity is the maximum number of tasks in the queue.
	Capacity int

	// MaxConcurrency is the maximum number of concurrent workers.
	MaxConcurrency int

	// EnableDynamicWorkers enables optional dynamic worker scaling.
	// Default is false, which keeps a fixed-size worker pool.
	EnableDynamicWorkers bool

	// MinConcurrency is the minimum number of workers when dynamic scaling is enabled.
	// Ignored when EnableDynamicWorkers is false.
	MinConcurrency int

	// Backpressure is the strategy when the queue is full.
	Backpressure BackpressureStrategy

	// RedirectLane is the lane name to redirect to when Backpressure is Redirect.
	RedirectLane string

	// EnablePriority enables priority queue support.
	EnablePriority bool

	// RateLimit enables rate limiting (tasks per second, 0 = unlimited).
	RateLimit float64
}

// Validate validates the lane configuration.
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("lane name cannot be empty")
	}
	if c.Capacity <= 0 {
		return fmt.Errorf("lane capacity must be positive, got %d", c.Capacity)
	}
	if c.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive, got %d", c.MaxConcurrency)
	}
	if c.EnableDynamicWorkers {
		if c.MinConcurrency <= 0 {
			return fmt.Errorf("min concurrency must be positive when dynamic workers are enabled, got %d", c.MinConcurrency)
		}
		if c.MinConcurrency > c.MaxConcurrency {
			return fmt.Errorf("min concurrency (%d) cannot exceed max concurrency (%d)", c.MinConcurrency, c.MaxConcurrency)
		}
	}
	if c.Backpressure == Redirect && c.RedirectLane == "" {
		return fmt.Errorf("redirect lane must be specified when using redirect strategy")
	}
	if c.RateLimit < 0 {
		return fmt.Errorf("rate limit cannot be negative")
	}
	return nil
}

// Lane represents a resource-constrained execution queue.
type Lane interface {
	// Name returns the lane name.
	Name() string

	// Submit submits a task to the lane.
	// The behavior when full depends on the backpressure strategy.
	Submit(ctx context.Context, task Task) error

	// TrySubmit attempts to submit a task without blocking.
	// Returns true if the task was accepted, false otherwise.
	TrySubmit(task Task) bool

	// Stats returns current lane statistics.
	Stats() Stats

	// Close gracefully shuts down the lane.
	Close(ctx context.Context) error

	// IsClosed returns true if the lane is closed.
	IsClosed() bool
}

// Stats holds statistics for a Lane.
type Stats struct {
	// Name is the lane name.
	Name string

	// Pending is the number of pending tasks in the queue.
	Pending int

	// Running is the number of currently running tasks.
	Running int

	// Completed is the total number of completed tasks.
	Completed int64

	// Failed is the total number of failed tasks.
	Failed int64

	// Dropped is the total number of dropped tasks.
	Dropped int64

	// Accepted is the total number of directly accepted submissions.
	Accepted int64

	// Rejected is the total number of submissions rejected before admission.
	Rejected int64

	// Redirected is the total number of submissions redirected to other lanes.
	Redirected int64

	// Capacity is the queue capacity.
	Capacity int

	// MaxConcurrency is the maximum concurrency.
	MaxConcurrency int

	// WaitTime is the average wait time in the queue.
	WaitTime time.Duration

	// ProcessTime is the average processing time.
	ProcessTime time.Duration
}

// Utilization returns the current utilization ratio (0.0 - 1.0).
func (s Stats) Utilization() float64 {
	if s.Capacity == 0 {
		return 0
	}
	return float64(s.Pending+s.Running) / float64(s.Capacity+s.MaxConcurrency)
}

// IsFull returns true if the lane is at capacity.
func (s Stats) IsFull() bool {
	return s.Pending >= s.Capacity
}

// String returns a human-readable string representation of Stats.
func (s Stats) String() string {
	return fmt.Sprintf(
		"Stats{Name: %s, Pending: %d, Running: %d, Completed: %d, Failed: %d, Dropped: %d, Accepted: %d, Rejected: %d, Redirected: %d, Utilization: %.2f%%}",
		s.Name, s.Pending, s.Running, s.Completed, s.Failed, s.Dropped, s.Accepted, s.Rejected, s.Redirected, s.Utilization()*100,
	)
}
