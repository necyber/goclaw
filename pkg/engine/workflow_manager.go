// Package engine provides the core orchestration engine for multi-agent systems.
package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/storage"
	"github.com/google/uuid"
)

// SubmitWorkflowOptions configures workflow submit behavior.
type SubmitWorkflowOptions struct {
	Mode    SubmissionMode
	TaskFns map[string]func(context.Context) error
}

// SubmitWorkflowRequest submits a workflow and returns its ID.
func (e *Engine) SubmitWorkflowRequest(ctx context.Context, req *models.WorkflowRequest) (string, error) {
	resp, err := e.SubmitWorkflowRuntime(ctx, req, SubmitWorkflowOptions{Mode: SubmissionModeAsync})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// SubmitWorkflowRuntime submits a workflow using explicit runtime semantics.
func (e *Engine) SubmitWorkflowRuntime(ctx context.Context, req *models.WorkflowRequest, opts SubmitWorkflowOptions) (*models.WorkflowStatusResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("workflow request cannot be nil")
	}

	wfState := newWorkflowState(req)
	if err := e.storage.SaveWorkflow(ctx, wfState); err != nil {
		return nil, fmt.Errorf("failed to save workflow: %w", err)
	}
	for _, taskState := range wfState.TaskStatus {
		if err := e.storage.SaveTask(ctx, wfState.ID, taskState); err != nil {
			return nil, fmt.Errorf("failed to save initial task %s: %w", taskState.ID, err)
		}
	}
	e.metrics.RecordWorkflowSubmission(workflowStatusPending)
	e.emitWorkflowStateChanged(wfState.ID, wfState.Name, "", wfState.Status)

	e.logger.Info("workflow submitted", "id", wfState.ID, "name", wfState.Name, "tasks", len(wfState.Tasks))

	mode := normalizeSubmissionMode(opts.Mode)
	hasTaskFns := len(opts.TaskFns) > 0

	// Without executable task functions, workflow remains persisted pending.
	if !hasTaskFns {
		return e.workflowStateToResponse(wfState), nil
	}

	exec, err := e.startWorkflowExecution(wfState.ID, opts.TaskFns)
	if err != nil {
		if transitionErr := e.markWorkflowFailedFromPending(ctx, wfState.ID, err); transitionErr != nil {
			e.logger.Error("failed to mark workflow failed after start error", "workflow_id", wfState.ID, "error", transitionErr)
		}
		return nil, err
	}

	if mode == SubmissionModeAsync {
		return e.workflowStateToResponse(wfState), nil
	}

	// Sync mode: wait for terminal state or caller cancellation.
	select {
	case <-exec.done:
		return e.GetWorkflowStatusResponse(context.Background(), wfState.ID)
	case <-ctx.Done():
		statusResp, statusErr := e.GetWorkflowStatusResponse(context.Background(), wfState.ID)
		if statusErr != nil {
			return nil, ctx.Err()
		}
		return statusResp, ctx.Err()
	}
}

func normalizeSubmissionMode(mode SubmissionMode) SubmissionMode {
	switch mode {
	case SubmissionModeAsync:
		return SubmissionModeAsync
	case SubmissionModeSync:
		return SubmissionModeSync
	default:
		return SubmissionModeSync
	}
}

func newWorkflowState(req *models.WorkflowRequest) *storage.WorkflowState {
	id := uuid.New().String()
	now := time.Now().UTC()
	taskStatus := make(map[string]*storage.TaskState, len(req.Tasks))
	for _, task := range req.Tasks {
		taskStatus[task.ID] = &storage.TaskState{
			ID:     task.ID,
			Name:   task.Name,
			Status: taskStatusPending,
		}
	}

	return &storage.WorkflowState{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Status:      workflowStatusPending,
		Tasks:       req.Tasks,
		TaskStatus:  taskStatus,
		Metadata:    req.Metadata,
		CreatedAt:   now,
	}
}

func (e *Engine) startWorkflowExecution(workflowID string, taskFns map[string]func(context.Context) error) (*workflowExecution, error) {
	if _, exists := e.getExecution(workflowID); exists {
		return nil, fmt.Errorf("workflow %s is already executing", workflowID)
	}

	wfState, err := e.storage.GetWorkflow(context.Background(), workflowID)
	if err != nil {
		return nil, err
	}
	if wfState.Status != workflowStatusPending {
		return nil, fmt.Errorf("workflow %s is not pending: %s", workflowID, wfState.Status)
	}

	execCtx, cancel := context.WithCancel(context.Background())
	exec := &workflowExecution{
		workflowID: workflowID,
		cancel:     cancel,
		done:       make(chan struct{}),
		wfState:    wfState,
	}
	e.registerExecution(exec)

	go func() {
		defer close(exec.done)
		defer e.unregisterExecution(workflowID)
		e.runWorkflowExecution(execCtx, exec, taskFns)
	}()

	return exec, nil
}

func (e *Engine) runWorkflowExecution(ctx context.Context, exec *workflowExecution, taskFns map[string]func(context.Context) error) {
	wf := e.workflowFromState(exec.wfState, taskFns)

	if err := e.transitionWorkflow(exec, workflowStatusScheduled, ""); err != nil {
		e.logger.Error("failed to transition workflow to scheduled", "workflow_id", exec.workflowID, "error", err)
		_ = e.transitionWorkflow(exec, workflowStatusFailed, err.Error())
		return
	}
	if err := e.transitionWorkflow(exec, workflowStatusRunning, ""); err != nil {
		e.logger.Error("failed to transition workflow to running", "workflow_id", exec.workflowID, "error", err)
		_ = e.transitionWorkflow(exec, workflowStatusFailed, err.Error())
		return
	}

	g := dag.NewGraph()
	for _, t := range wf.Tasks {
		if t.Lane == "" {
			t.Lane = defaultLaneName
		}
		if err := g.AddTask(t); err != nil {
			_ = e.transitionWorkflow(exec, workflowStatusFailed, err.Error())
			return
		}
	}
	plan, err := g.Compile()
	if err != nil {
		_ = e.transitionWorkflow(exec, workflowStatusFailed, err.Error())
		return
	}

	tracker := newStateTracker()
	taskIDs := make([]string, 0, len(wf.Tasks))
	taskNameByID := make(map[string]string, len(wf.Tasks))
	for _, t := range wf.Tasks {
		taskIDs = append(taskIDs, t.ID)
		taskNameByID[t.ID] = t.Name
	}
	tracker.InitTasks(taskIDs)
	tracker.SetOnStateChange(func(taskID string, oldState, newState TaskState, result TaskResult) {
		if err := e.transitionTask(exec, taskID, oldState, newState, result); err != nil {
			e.logger.Error("failed to persist task transition", "workflow_id", exec.workflowID, "task_id", taskID, "error", err)
		}
		_ = taskNameByID
	})

	sched := newScheduler(tracker, e.logger, e.signalBus, e.laneManager)
	err = sched.Schedule(ctx, plan, wf.TaskFns)
	if err != nil {
		if ctx.Err() != nil {
			if transitionErr := e.transitionWorkflow(exec, workflowStatusCancelled, ctx.Err().Error()); transitionErr != nil && !isTerminalWorkflowStatus(exec.wfState.Status) {
				e.logger.Error("failed to transition cancelled workflow", "workflow_id", exec.workflowID, "error", transitionErr)
			}
			return
		}
		if transitionErr := e.transitionWorkflow(exec, workflowStatusFailed, err.Error()); transitionErr != nil && !isTerminalWorkflowStatus(exec.wfState.Status) {
			e.logger.Error("failed to transition failed workflow", "workflow_id", exec.workflowID, "error", transitionErr)
		}
		return
	}

	if transitionErr := e.transitionWorkflow(exec, workflowStatusCompleted, ""); transitionErr != nil && !isTerminalWorkflowStatus(exec.wfState.Status) {
		e.logger.Error("failed to transition completed workflow", "workflow_id", exec.workflowID, "error", transitionErr)
	}
}

func (e *Engine) workflowFromState(state *storage.WorkflowState, taskFns map[string]func(context.Context) error) *Workflow {
	tasks := make([]*dag.Task, 0, len(state.Tasks))
	for _, t := range state.Tasks {
		task := &dag.Task{
			ID:      t.ID,
			Name:    t.Name,
			Agent:   t.Type,
			Deps:    append([]string(nil), t.DependsOn...),
			Retries: t.Retries,
		}
		if task.Agent == "" {
			task.Agent = "function"
		}
		if t.Timeout > 0 {
			task.Timeout = time.Duration(t.Timeout) * time.Second
		}
		if laneName, ok := t.Config["lane"].(string); ok {
			task.Lane = laneName
		}
		tasks = append(tasks, task)
	}

	return &Workflow{
		ID:      state.ID,
		Tasks:   tasks,
		TaskFns: taskFns,
	}
}

func (e *Engine) transitionWorkflow(exec *workflowExecution, newStatus, errMsg string) error {
	exec.mu.Lock()
	defer exec.mu.Unlock()

	oldStatus := exec.wfState.Status
	if oldStatus == newStatus {
		return nil
	}
	if err := validateWorkflowTransition(oldStatus, newStatus); err != nil {
		return err
	}

	now := time.Now().UTC()
	exec.wfState.Status = newStatus
	switch newStatus {
	case workflowStatusRunning:
		if exec.wfState.StartedAt == nil {
			t := now
			exec.wfState.StartedAt = &t
		}
		exec.wfState.Error = ""
	case workflowStatusCompleted:
		t := now
		exec.wfState.CompletedAt = &t
		exec.wfState.Error = ""
	case workflowStatusFailed, workflowStatusCancelled:
		t := now
		exec.wfState.CompletedAt = &t
		exec.wfState.Error = errMsg
	}

	if err := e.storage.SaveWorkflow(context.Background(), exec.wfState); err != nil {
		return err
	}
	e.emitWorkflowStateChanged(exec.wfState.ID, exec.wfState.Name, oldStatus, newStatus)

	if newStatus == workflowStatusRunning {
		e.metrics.IncActiveWorkflows(workflowStatusRunning)
	}
	if oldStatus == workflowStatusRunning && isTerminalWorkflowStatus(newStatus) {
		e.metrics.DecActiveWorkflows(workflowStatusRunning)
		started := exec.wfState.CreatedAt
		if exec.wfState.StartedAt != nil {
			started = *exec.wfState.StartedAt
		}
		e.metrics.RecordWorkflowDuration(workflowMetricLabel(newStatus, errMsg), now.Sub(started))
		e.metrics.RecordWorkflowSubmission(workflowMetricLabel(newStatus, errMsg))
	}

	return nil
}

func workflowMetricLabel(status, errMsg string) string {
	if status == workflowStatusFailed && strings.Contains(strings.ToLower(errMsg), "deadline") {
		return "failed_timeout"
	}
	return status
}

func (e *Engine) transitionTask(exec *workflowExecution, taskID string, oldState, newState TaskState, result TaskResult) error {
	newStatus := mapTaskStateToStatus(newState)
	if newStatus == "" {
		return nil
	}

	exec.mu.Lock()
	defer exec.mu.Unlock()

	taskState, ok := exec.wfState.TaskStatus[taskID]
	if !ok {
		taskState = &storage.TaskState{ID: taskID, Name: taskID, Status: taskStatusPending}
		exec.wfState.TaskStatus[taskID] = taskState
	}

	if exec.wfState.Status == workflowStatusCancelled && (newStatus == taskStatusCompleted || newStatus == taskStatusFailed) {
		newStatus = taskStatusCancelled
		if result.Error == nil {
			result.Error = context.Canceled
		}
	}

	oldStatus := taskState.Status
	if oldStatus == newStatus {
		return nil
	}
	if err := validateTaskTransition(oldStatus, newStatus); err != nil {
		return err
	}

	now := time.Now().UTC()
	taskState.Status = newStatus
	if newStatus == taskStatusRunning {
		started := now
		if !result.StartedAt.IsZero() {
			started = result.StartedAt.UTC()
		}
		taskState.StartedAt = &started
		taskState.Error = ""
	}
	if newStatus == taskStatusScheduled && oldStatus == taskStatusRunning {
		e.metrics.RecordTaskRetry()
	}
	if isTerminalTaskStatus(newStatus) {
		completed := now
		if !result.EndedAt.IsZero() {
			completed = result.EndedAt.UTC()
		}
		taskState.CompletedAt = &completed
		if result.Error != nil {
			taskState.Error = result.Error.Error()
		} else if newStatus != taskStatusCompleted {
			taskState.Error = newStatus
		} else {
			taskState.Error = ""
		}
		if taskState.StartedAt != nil {
			e.metrics.RecordTaskDuration(completed.Sub(*taskState.StartedAt))
		}
		e.metrics.RecordTaskExecution(taskMetricLabel(newStatus, taskState.Error))
	}

	if err := e.storage.SaveTask(context.Background(), exec.workflowID, taskState); err != nil {
		return err
	}
	e.emitTaskStateChanged(exec.workflowID, taskID, taskState.Name, oldStatus, newStatus, taskState.Error, taskState.Result)

	_ = oldState
	return nil
}

func taskMetricLabel(status, errMsg string) string {
	if status == taskStatusFailed && strings.Contains(strings.ToLower(errMsg), "deadline") {
		return "failed_timeout"
	}
	return status
}

func mapTaskStateToStatus(state TaskState) string {
	switch state {
	case TaskStatePending:
		return taskStatusPending
	case TaskStateScheduled, TaskStateRetrying:
		return taskStatusScheduled
	case TaskStateRunning:
		return taskStatusRunning
	case TaskStateCompleted:
		return taskStatusCompleted
	case TaskStateFailed:
		return taskStatusFailed
	case TaskStateCancelled:
		return taskStatusCancelled
	default:
		return ""
	}
}

func (e *Engine) markWorkflowFailedFromPending(ctx context.Context, workflowID string, cause error) error {
	wfState, err := e.storage.GetWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}
	if wfState.Status != workflowStatusPending {
		return nil
	}
	if err := validateWorkflowTransition(wfState.Status, workflowStatusFailed); err != nil {
		return err
	}
	now := time.Now().UTC()
	wfState.Status = workflowStatusFailed
	wfState.CompletedAt = &now
	wfState.Error = cause.Error()
	if err := e.storage.SaveWorkflow(ctx, wfState); err != nil {
		return err
	}
	e.emitWorkflowStateChanged(wfState.ID, wfState.Name, workflowStatusPending, workflowStatusFailed)
	e.metrics.RecordWorkflowSubmission(workflowMetricLabel(workflowStatusFailed, cause.Error()))
	return nil
}

// GetWorkflowStatusResponse retrieves workflow status.
func (e *Engine) GetWorkflowStatusResponse(ctx context.Context, id string) (*models.WorkflowStatusResponse, error) {
	wfState, err := e.storage.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}
	return e.workflowStateToResponse(wfState), nil
}

func (e *Engine) workflowStateToResponse(wfState *storage.WorkflowState) *models.WorkflowStatusResponse {
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

	taskIDs := make([]string, 0, len(wfState.TaskStatus))
	for taskID := range wfState.TaskStatus {
		taskIDs = append(taskIDs, taskID)
	}
	sort.Strings(taskIDs)
	for _, taskID := range taskIDs {
		taskState := wfState.TaskStatus[taskID]
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

	return resp
}

// ListWorkflowsResponse lists workflows with filtering.
func (e *Engine) ListWorkflowsResponse(ctx context.Context, filter models.WorkflowFilter) ([]*models.WorkflowStatusResponse, int, error) {
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

	result := make([]*models.WorkflowStatusResponse, 0, len(workflows))
	for _, wf := range workflows {
		result = append(result, e.workflowStateToResponse(wf))
	}

	return result, total, nil
}

// CancelWorkflowRequest cancels a running or pending workflow.
func (e *Engine) CancelWorkflowRequest(ctx context.Context, id string) error {
	wfState, err := e.storage.GetWorkflow(ctx, id)
	if err != nil {
		return err
	}
	if isTerminalWorkflowStatus(wfState.Status) {
		return fmt.Errorf("workflow cannot be cancelled: already %s", wfState.Status)
	}

	if exec, ok := e.getExecution(id); ok {
		exec.cancel()
		for taskID, taskState := range exec.wfState.TaskStatus {
			if isTerminalTaskStatus(taskState.Status) {
				continue
			}
			if err := e.transitionTask(exec, taskID, TaskStatePending, TaskStateCancelled, TaskResult{Error: context.Canceled}); err != nil {
				e.logger.Warn("failed to cancel task during workflow cancel", "workflow_id", id, "task_id", taskID, "error", err)
			}
		}
		if err := e.transitionWorkflow(exec, workflowStatusCancelled, "cancelled by request"); err != nil && !isTerminalWorkflowStatus(exec.wfState.Status) {
			return err
		}
		return nil
	}

	oldStatus := wfState.Status
	if err := validateWorkflowTransition(oldStatus, workflowStatusCancelled); err != nil {
		return err
	}

	now := time.Now().UTC()
	wfState.Status = workflowStatusCancelled
	wfState.CompletedAt = &now
	wfState.Error = "cancelled by request"
	for _, task := range wfState.TaskStatus {
		if isTerminalTaskStatus(task.Status) {
			continue
		}
		task.Status = taskStatusCancelled
		task.CompletedAt = &now
		task.Error = "cancelled by request"
		if err := e.storage.SaveTask(ctx, wfState.ID, task); err != nil {
			return err
		}
		e.emitTaskStateChanged(wfState.ID, task.ID, task.Name, oldStatus, task.Status, task.Error, task.Result)
	}

	if err := e.storage.SaveWorkflow(ctx, wfState); err != nil {
		return err
	}
	e.emitWorkflowStateChanged(wfState.ID, wfState.Name, oldStatus, wfState.Status)
	e.metrics.RecordWorkflowSubmission(workflowStatusCancelled)

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
		Error:       taskState.Error,
		CompletedAt: taskState.CompletedAt,
	}
	if isTerminalTaskStatus(taskState.Status) {
		resp.Result = taskState.Result
	}
	if !isTerminalTaskStatus(taskState.Status) {
		resp.Result = nil
		resp.Error = ""
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

// SubmitWorkflow executes a runtime workflow request for adapter callers.
func (e *Engine) SubmitWorkflow(ctx context.Context, req *models.WorkflowRequest, opts SubmitWorkflowOptions) (*models.WorkflowStatusResponse, error) {
	return e.SubmitWorkflowRuntime(ctx, req, opts)
}
