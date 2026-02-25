// Package storage provides persistent storage abstraction for workflows and tasks.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
)

// Storage defines the interface for persistent storage operations.
type Storage interface {
	// Workflow operations
	SaveWorkflow(ctx context.Context, wf *WorkflowState) error
	GetWorkflow(ctx context.Context, id string) (*WorkflowState, error)
	ListWorkflows(ctx context.Context, filter *WorkflowFilter) ([]*WorkflowState, int, error)
	DeleteWorkflow(ctx context.Context, id string) error

	// Task operations
	SaveTask(ctx context.Context, workflowID string, task *TaskState) error
	GetTask(ctx context.Context, workflowID, taskID string) (*TaskState, error)
	ListTasks(ctx context.Context, workflowID string) ([]*TaskState, error)

	// Lifecycle
	Close() error
}

// WorkflowState represents the persisted state of a workflow.
type WorkflowState struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Status      string                  `json:"status"`
	Tasks       []models.TaskDefinition `json:"tasks"`
	TaskStatus  map[string]*TaskState   `json:"task_status"`
	Metadata    map[string]string       `json:"metadata"`
	CreatedAt   time.Time               `json:"created_at"`
	StartedAt   *time.Time              `json:"started_at,omitempty"`
	CompletedAt *time.Time              `json:"completed_at,omitempty"`
	Error       string                  `json:"error,omitempty"`
}

// TaskState represents the persisted state of a task.
type TaskState struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Status      string      `json:"status"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Error       string      `json:"error,omitempty"`
	Result      interface{} `json:"result,omitempty"`
}

// WorkflowFilter defines filtering options for listing workflows.
type WorkflowFilter struct {
	Status []string `json:"status,omitempty"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}

// NotFoundError indicates that the requested entity was not found.
type NotFoundError struct {
	EntityType string
	ID         string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.EntityType, e.ID)
}

// DuplicateKeyError indicates that an entity with the given ID already exists.
type DuplicateKeyError struct {
	EntityType string
	ID         string
}

func (e *DuplicateKeyError) Error() string {
	return fmt.Sprintf("%s already exists: %s", e.EntityType, e.ID)
}

// StorageUnavailableError indicates that the storage backend is unavailable.
type StorageUnavailableError struct {
	Cause error
}

func (e *StorageUnavailableError) Error() string {
	return fmt.Sprintf("storage unavailable: %v", e.Cause)
}

// SerializationError indicates a failure in data serialization/deserialization.
type SerializationError struct {
	Operation string
	Cause     error
}

func (e *SerializationError) Error() string {
	return fmt.Sprintf("serialization error during %s: %v", e.Operation, e.Cause)
}
