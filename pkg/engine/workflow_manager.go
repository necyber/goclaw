// Package engine provides the core orchestration engine for multi-agent systems.
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/goclaw/goclaw/pkg/api/models"
)

// WorkflowState represents the internal state of a workflow.
type WorkflowState struct {
	ID          string
	Name        string
	Description string
	Status      string
	Tasks       []models.TaskDefinition
	TaskStatus  map[string]*TaskStatusState
	Metadata    map[string]string
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Error       string
}

// TaskStatusState represents the internal state of a task.
type TaskStatusState struct {
	ID          string
	Name        string
	Status      string
	StartedAt   *time.Time
	CompletedAt *time.Time
	Error       string
	Result      interface{}
}

// WorkflowStore manages workflow state storage.
type WorkflowStore struct {
	mu        sync.RWMutex
	workflows map[string]*WorkflowState
}

// NewWorkflowStore creates a new workflow store.
func NewWorkflowStore() *WorkflowStore {
	return &WorkflowStore{
		workflows: make(map[string]*WorkflowState),
	}
}

// Create creates a new workflow state.
func (s *WorkflowStore) Create(req *models.WorkflowRequest) (*WorkflowState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate unique workflow ID
	id := uuid.New().String()

	// Initialize task status map
	taskStatus := make(map[string]*TaskStatusState)
	for _, task := range req.Tasks {
		taskStatus[task.ID] = &TaskStatusState{
			ID:     task.ID,
			Name:   task.Name,
			Status: "pending",
		}
	}

	// Create workflow state
	wf := &WorkflowState{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Status:      "pending",
		Tasks:       req.Tasks,
		TaskStatus:  taskStatus,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
	}

	s.workflows[id] = wf
	return wf, nil
}

// Get retrieves a workflow by ID.
func (s *WorkflowStore) Get(id string) (*WorkflowState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wf, exists := s.workflows[id]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}

	return wf, nil
}

// List returns a filtered list of workflows.
func (s *WorkflowStore) List(filter models.WorkflowFilter) ([]*WorkflowState, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect all workflows
	var all []*WorkflowState
	for _, wf := range s.workflows {
		// Apply status filter if specified
		if filter.Status != "" && wf.Status != filter.Status {
			continue
		}
		all = append(all, wf)
	}

	total := len(all)

	// Apply pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Calculate slice bounds
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	result := all[start:end]
	return result, total, nil
}

// UpdateStatus updates the workflow status.
func (s *WorkflowStore) UpdateStatus(id string, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wf, exists := s.workflows[id]
	if !exists {
		return fmt.Errorf("workflow not found: %s", id)
	}

	wf.Status = status

	// Update timestamps based on status
	now := time.Now()
	switch status {
	case "running":
		if wf.StartedAt == nil {
			wf.StartedAt = &now
		}
	case "completed", "failed", "cancelled":
		if wf.CompletedAt == nil {
			wf.CompletedAt = &now
		}
	}

	return nil
}

// UpdateTaskStatus updates a task's status within a workflow.
func (s *WorkflowStore) UpdateTaskStatus(workflowID, taskID, status string, result interface{}, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wf, exists := s.workflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	taskState, exists := wf.TaskStatus[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	taskState.Status = status
	taskState.Result = result

	now := time.Now()
	switch status {
	case "running":
		if taskState.StartedAt == nil {
			taskState.StartedAt = &now
		}
	case "completed", "failed":
		if taskState.CompletedAt == nil {
			taskState.CompletedAt = &now
		}
		if err != nil {
			taskState.Error = err.Error()
		}
	}

	return nil
}

// Delete removes a workflow from the store.
func (s *WorkflowStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workflows[id]; !exists {
		return fmt.Errorf("workflow not found: %s", id)
	}

	delete(s.workflows, id)
	return nil
}

// GetTaskResult retrieves a specific task's result.
func (s *WorkflowStore) GetTaskResult(workflowID, taskID string) (*TaskStatusState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wf, exists := s.workflows[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	taskState, exists := wf.TaskStatus[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return taskState, nil
}

// SubmitWorkflowRequest submits a workflow and returns its ID.
func (e *Engine) SubmitWorkflowRequest(ctx context.Context, req *models.WorkflowRequest) (string, error) {
	// Create workflow state
	wfState, err := e.workflowStore.Create(req)
	if err != nil {
		return "", fmt.Errorf("failed to create workflow state: %w", err)
	}

	// Convert to engine.Workflow for execution
	// Note: Full task execution integration will be implemented in Phase 2
	// For now, we create the workflow structure and log submission
	e.logger.Info("workflow submitted", "id", wfState.ID, "name", wfState.Name, "tasks", len(wfState.Tasks))

	return wfState.ID, nil
}

// GetWorkflowStatusResponse retrieves workflow status.
func (e *Engine) GetWorkflowStatusResponse(ctx context.Context, id string) (*models.WorkflowStatusResponse, error) {
	wfState, err := e.workflowStore.Get(id)
	if err != nil {
		return nil, err
	}

	// Convert to response model
	resp := &models.WorkflowStatusResponse{
		ID:          wfState.ID,
		Name:        wfState.Name,
		Status:      wfState.Status,
		CreatedAt:   wfState.CreatedAt,
		StartedAt:   wfState.StartedAt,
		CompletedAt: wfState.CompletedAt,
		Metadata:    wfState.Metadata,
		Error:       wfState.Error,
		Tasks:       make([]models.TaskStatus, 0, len(wfState.TaskStatus)),
	}

	// Convert task statuses
	for _, taskState := range wfState.TaskStatus {
		resp.Tasks = append(resp.Tasks, models.TaskStatus{
			ID:          taskState.ID,
			Name:        taskState.Name,
			Status:      taskState.Status,
			StartedAt:   taskState.StartedAt,
			CompletedAt: taskState.CompletedAt,
			Error:       taskState.Error,
			Result:      taskState.Result,
		})
	}

	return resp, nil
}

// ListWorkflowsResponse lists workflows with filtering.
func (e *Engine) ListWorkflowsResponse(ctx context.Context, filter models.WorkflowFilter) ([]*models.WorkflowStatusResponse, int, error) {
	workflows, total, err := e.workflowStore.List(filter)
	if err != nil {
		return nil, 0, err
	}

	// Convert to response models
	result := make([]*models.WorkflowStatusResponse, 0, len(workflows))
	for _, wf := range workflows {
		resp, err := e.GetWorkflowStatusResponse(ctx, wf.ID)
		if err != nil {
			continue
		}
		result = append(result, resp)
	}

	return result, total, nil
}

// CancelWorkflowRequest cancels a running workflow.
func (e *Engine) CancelWorkflowRequest(ctx context.Context, id string) error {
	wfState, err := e.workflowStore.Get(id)
	if err != nil {
		return err
	}

	// Check if workflow can be cancelled
	if wfState.Status == "completed" || wfState.Status == "failed" || wfState.Status == "cancelled" {
		return fmt.Errorf("workflow cannot be cancelled: already %s", wfState.Status)
	}

	// Update status to cancelled
	if err := e.workflowStore.UpdateStatus(id, "cancelled"); err != nil {
		return err
	}

	e.logger.Info("workflow cancelled", "id", id)
	return nil
}

// GetTaskResultResponse retrieves a task's result.
func (e *Engine) GetTaskResultResponse(ctx context.Context, workflowID, taskID string) (*models.TaskResultResponse, error) {
	taskState, err := e.workflowStore.GetTaskResult(workflowID, taskID)
	if err != nil {
		return nil, err
	}

	resp := &models.TaskResultResponse{
		WorkflowID:  workflowID,
		TaskID:      taskState.ID,
		Status:      taskState.Status,
		Result:      taskState.Result,
		Error:       taskState.Error,
		CompletedAt: taskState.CompletedAt,
	}

	return resp, nil
}

// IsHealthy returns true if the engine is healthy.
func (e *Engine) IsHealthy() bool {
	return engineState(e.state.Load()) == stateRunning
}

// IsReady returns true if the engine is ready to accept requests.
func (e *Engine) IsReady() bool {
	return engineState(e.state.Load()) == stateRunning && e.laneManager != nil
}

// EngineStatus represents the engine's current status.
type EngineStatus struct {
	State   string `json:"state"`
	Uptime  string `json:"uptime,omitempty"`
	Version string `json:"version,omitempty"`
}

// GetStatus returns detailed engine status.
func (e *Engine) GetStatus() *EngineStatus {
	state := engineState(e.state.Load())
	stateStr := "unknown"
	switch state {
	case stateIdle:
		stateStr = "idle"
	case stateRunning:
		stateStr = "running"
	case stateStopped:
		stateStr = "stopped"
	case stateError:
		stateStr = "error"
	}

	return &EngineStatus{
		State:   stateStr,
		Version: e.cfg.App.Version,
	}
}
