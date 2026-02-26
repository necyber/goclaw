package lane

import (
	"fmt"
)

// LaneFullError is returned when a lane is at capacity and cannot accept new tasks.
type LaneFullError struct {
	LaneName string
	Capacity int
}

func (e *LaneFullError) Error() string {
	return fmt.Sprintf("lane %s is full (capacity: %d)", e.LaneName, e.Capacity)
}

// LaneClosedError is returned when attempting to submit to a closed lane.
type LaneClosedError struct {
	LaneName string
}

func (e *LaneClosedError) Error() string {
	return fmt.Sprintf("lane %s is closed", e.LaneName)
}

// TaskDroppedError is returned when a task is dropped due to backpressure.
type TaskDroppedError struct {
	LaneName string
	TaskID   string
}

func (e *TaskDroppedError) Error() string {
	return fmt.Sprintf("task %s dropped in lane %s due to backpressure", e.TaskID, e.LaneName)
}

// TaskDuplicateError is returned when a duplicate task is submitted.
type TaskDuplicateError struct {
	LaneName string
	TaskID   string
}

func (e *TaskDuplicateError) Error() string {
	return fmt.Sprintf("task %s is duplicate in lane %s", e.TaskID, e.LaneName)
}

// LaneNotFoundError is returned when a lane is not found.
type LaneNotFoundError struct {
	LaneName string
}

func (e *LaneNotFoundError) Error() string {
	return fmt.Sprintf("lane %s not found", e.LaneName)
}

// DuplicateLaneError is returned when attempting to register a lane that already exists.
type DuplicateLaneError struct {
	LaneName string
}

func (e *DuplicateLaneError) Error() string {
	return fmt.Sprintf("lane %s already exists", e.LaneName)
}

// RateLimitError is returned when rate limit is exceeded.
type RateLimitError struct {
	LaneName string
	WaitTime float64 // seconds
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded in lane %s, retry after %.2f seconds", e.LaneName, e.WaitTime)
}

// IsLaneFullError returns true if the error is a LaneFullError.
func IsLaneFullError(err error) bool {
	_, ok := err.(*LaneFullError)
	return ok
}

// IsLaneClosedError returns true if the error is a LaneClosedError.
func IsLaneClosedError(err error) bool {
	_, ok := err.(*LaneClosedError)
	return ok
}

// IsTaskDroppedError returns true if the error is a TaskDroppedError.
func IsTaskDroppedError(err error) bool {
	_, ok := err.(*TaskDroppedError)
	return ok
}

// IsTaskDuplicateError returns true if the error is a TaskDuplicateError.
func IsTaskDuplicateError(err error) bool {
	_, ok := err.(*TaskDuplicateError)
	return ok
}

// IsLaneNotFoundError returns true if the error is a LaneNotFoundError.
func IsLaneNotFoundError(err error) bool {
	_, ok := err.(*LaneNotFoundError)
	return ok
}
