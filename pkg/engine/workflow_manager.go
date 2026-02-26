// Package engine provides the core orchestration engine for multi-agent systems.
package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/storage"
	"github.com/google/uuid"
)

// SubmitWorkflowRequest submits a workflow and returns its ID.
func (e *Engine) SubmitWorkflowRequest(ctx context.Context, req *models.WorkflowRequest) (string, error) {
	// Generate unique workflow ID
	id := uuid.New().String()

	// Initialize task status map
	taskStatus := make(map[string]*storage.TaskState)
	for _, task := range req.Tasks {
		taskStatus[task.ID] = &storage.TaskState{
			ID:     task.ID,
			Name:   task.Name,
			Status: "pending",
		}
	}

	// Create workflow state
	wfState := &storage.WorkflowState{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Status:      "pending",
		Tasks:       req.Tasks,
		TaskStatus:  taskStatus,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
	}

	// Save to storage
	if err := e.storage.SaveWorkflow(ctx, wfState); err != nil {
		return "", fmt.Errorf("failed to save workflow: %w", err)
	}
	e.emitWorkflowStateChanged(wfState.ID, wfState.Name, "", wfState.Status)

	e.logger.Info("workflow submitted", "id", wfState.ID, "name", wfState.Name, "tasks", len(wfState.Tasks))

	return wfState.ID, nil
}

// GetWorkflowStatusResponse retrieves workflow status.
func (e *Engine) GetWorkflowStatusResponse(ctx context.Context, id string) (*models.WorkflowStatusResponse, error) {
	wfState, err := e.storage.GetWorkflow(ctx, id)
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
	// Convert models.WorkflowFilter to storage.WorkflowFilter
	storageFilter := &storage.WorkflowFilter{
		Status: []string{},
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}
	if filter.Status != "" {
		storageFilter.Status = []string{filter.Status}
	}

	workflows, total, err := e.storage.ListWorkflows(ctx, storageFilter)
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
	wfState, err := e.storage.GetWorkflow(ctx, id)
	if err != nil {
		return err
	}

	// Check if workflow can be cancelled
	if wfState.Status == "completed" || wfState.Status == "failed" || wfState.Status == "cancelled" {
		return fmt.Errorf("workflow cannot be cancelled: already %s", wfState.Status)
	}

	// Update status to cancelled
	oldStatus := wfState.Status
	wfState.Status = "cancelled"
	now := time.Now()
	if wfState.CompletedAt == nil {
		wfState.CompletedAt = &now
	}

	if err := e.storage.SaveWorkflow(ctx, wfState); err != nil {
		return err
	}
	e.emitWorkflowStateChanged(wfState.ID, wfState.Name, oldStatus, wfState.Status)

	e.logger.Info("workflow cancelled", "id", id)
	return nil
}

// GetTaskResultResponse retrieves a task's result.
func (e *Engine) GetTaskResultResponse(ctx context.Context, workflowID, taskID string) (*models.TaskResultResponse, error) {
	taskState, err := e.storage.GetTask(ctx, workflowID, taskID)
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
