// Package models defines API request/response data structures.
package models

import "time"

// WorkflowRequest represents a workflow submission request.
type WorkflowRequest struct {
	// Name is the workflow name.
	Name string `json:"name" validate:"required,min=1,max=100" example:"data-processing-workflow"`

	// Description is an optional workflow description.
	Description string `json:"description,omitempty" validate:"max=500" example:"Process customer data and generate reports"`

	// Tasks is the list of tasks in the workflow.
	Tasks []TaskDefinition `json:"tasks" validate:"required,min=1,dive"`

	// Metadata holds optional key-value pairs.
	Metadata map[string]string `json:"metadata,omitempty" example:"environment:production,team:data-engineering"`
}

// TaskDefinition defines a single task in a workflow.
type TaskDefinition struct {
	// ID is the unique task identifier within the workflow.
	ID string `json:"id" validate:"required,min=1,max=100" example:"task-1"`

	// Name is the task name.
	Name string `json:"name" validate:"required,min=1,max=100" example:"Fetch data from API"`

	// Type is the task type (e.g., "http", "script", "function").
	Type string `json:"type" validate:"required,oneof=http script function" example:"http"`

	// DependsOn lists task IDs that must complete before this task.
	DependsOn []string `json:"depends_on,omitempty" example:"task-0"`

	// Config holds task-specific configuration.
	Config map[string]interface{} `json:"config,omitempty"`

	// Timeout is the maximum execution time in seconds.
	Timeout int `json:"timeout,omitempty" validate:"omitempty,min=1,max=3600" example:"300"`

	// Retries is the number of retry attempts on failure.
	Retries int `json:"retries,omitempty" validate:"omitempty,min=0,max=5" example:"3"`
}

// WorkflowResponse represents a workflow submission response.
type WorkflowResponse struct {
	// ID is the unique workflow identifier.
	ID string `json:"id"`

	// Name is the workflow name.
	Name string `json:"name"`

	// Status is the current workflow status.
	Status string `json:"status"`

	// CreatedAt is the workflow creation timestamp.
	CreatedAt time.Time `json:"created_at"`

	// Message provides additional information.
	Message string `json:"message,omitempty"`
}

// WorkflowStatusResponse represents a workflow status query response.
type WorkflowStatusResponse struct {
	// ID is the workflow identifier.
	ID string `json:"id"`

	// Name is the workflow name.
	Name string `json:"name"`

	// Status is the current workflow status.
	Status string `json:"status"`

	// CreatedAt is the workflow creation timestamp.
	CreatedAt time.Time `json:"created_at"`

	// StartedAt is when the workflow started executing.
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt is when the workflow completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Tasks is the list of task statuses.
	Tasks []TaskStatus `json:"tasks"`

	// Metadata holds workflow metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Error holds error information if the workflow failed.
	Error string `json:"error,omitempty"`
}

// TaskStatus represents the status of a single task.
type TaskStatus struct {
	// ID is the task identifier.
	ID string `json:"id"`

	// Name is the task name.
	Name string `json:"name"`

	// Status is the current task status.
	Status string `json:"status"`

	// StartedAt is when the task started.
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt is when the task completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error holds error information if the task failed.
	Error string `json:"error,omitempty"`

	// Result holds the task result data.
	Result interface{} `json:"result,omitempty"`
}

// WorkflowListResponse represents a paginated list of workflows.
type WorkflowListResponse struct {
	// Workflows is the list of workflow summaries.
	Workflows []WorkflowSummary `json:"workflows"`

	// Total is the total number of workflows matching the filter.
	Total int `json:"total"`

	// Limit is the maximum number of results returned.
	Limit int `json:"limit"`

	// Offset is the starting position in the result set.
	Offset int `json:"offset"`
}

// WorkflowSummary provides a brief overview of a workflow.
type WorkflowSummary struct {
	// ID is the workflow identifier.
	ID string `json:"id"`

	// Name is the workflow name.
	Name string `json:"name"`

	// Status is the current workflow status.
	Status string `json:"status"`

	// CreatedAt is the workflow creation timestamp.
	CreatedAt time.Time `json:"created_at"`

	// CompletedAt is when the workflow completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// TaskCount is the total number of tasks.
	TaskCount int `json:"task_count"`
}

// WorkflowFilter defines filtering options for listing workflows.
type WorkflowFilter struct {
	// Status filters by workflow status.
	Status string `json:"status,omitempty"`

	// Limit is the maximum number of results to return.
	Limit int `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`

	// Offset is the starting position in the result set.
	Offset int `json:"offset,omitempty" validate:"omitempty,min=0"`
}

// TaskResultResponse represents a task result query response.
type TaskResultResponse struct {
	// WorkflowID is the workflow identifier.
	WorkflowID string `json:"workflow_id"`

	// TaskID is the task identifier.
	TaskID string `json:"task_id"`

	// Status is the task status.
	Status string `json:"status"`

	// Result holds the task result data.
	Result interface{} `json:"result,omitempty"`

	// Error holds error information if the task failed.
	Error string `json:"error,omitempty"`

	// CompletedAt is when the task completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
