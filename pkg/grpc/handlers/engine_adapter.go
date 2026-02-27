package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/storage"
)

// EngineAdapter adapts engine.Engine to gRPC handler interfaces.
type EngineAdapter struct {
	engine     *engine.Engine
	startTime  time.Time
	lastErrMsg string
}

// NewEngineAdapter creates a new runtime adapter for gRPC services.
func NewEngineAdapter(eng *engine.Engine) *EngineAdapter {
	if eng == nil {
		return nil
	}
	return &EngineAdapter{
		engine:    eng,
		startTime: time.Now().UTC(),
	}
}

// SubmitWorkflow submits a workflow in async runtime mode.
func (a *EngineAdapter) SubmitWorkflow(ctx context.Context, name string, tasks []WorkflowTask) (string, error) {
	if a.engine == nil {
		return "", fmt.Errorf("engine adapter is not configured")
	}

	req := &models.WorkflowRequest{
		Name:  name,
		Tasks: make([]models.TaskDefinition, 0, len(tasks)),
		Async: true,
	}
	for _, t := range tasks {
		taskDef := models.TaskDefinition{
			ID:        t.ID,
			Name:      t.Name,
			Type:      "function",
			DependsOn: append([]string(nil), t.Dependencies...),
			Config:    map[string]interface{}{},
		}
		if laneName, ok := t.Metadata["lane"]; ok && laneName != "" {
			taskDef.Config["lane"] = laneName
		}
		req.Tasks = append(req.Tasks, taskDef)
	}

	resp, err := a.engine.SubmitWorkflowRuntime(ctx, req, engine.SubmitWorkflowOptions{
		Mode: engine.SubmissionModeAsync,
	})
	if err != nil {
		a.lastErrMsg = err.Error()
		return "", err
	}
	return resp.ID, nil
}

// GetWorkflowStatus returns persisted workflow status.
func (a *EngineAdapter) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	statusResp, err := a.engine.GetWorkflowStatusResponse(ctx, workflowID)
	if err != nil {
		a.lastErrMsg = err.Error()
		return nil, err
	}

	tasks := make([]*TaskStatus, 0, len(statusResp.Tasks))
	for _, task := range statusResp.Tasks {
		ts := &TaskStatus{
			TaskID:   task.ID,
			Name:     task.Name,
			Status:   task.Status,
			ErrorMsg: task.Error,
		}
		if task.StartedAt != nil {
			ts.StartedAt = task.StartedAt.Unix()
		}
		if task.CompletedAt != nil {
			ts.CompletedAt = task.CompletedAt.Unix()
		}
		tasks = append(tasks, ts)
	}

	ws := &WorkflowStatus{
		WorkflowID: statusResp.ID,
		Name:       statusResp.Name,
		Status:     statusResp.Status,
		Tasks:      tasks,
		CreatedAt:  statusResp.CreatedAt.Unix(),
		UpdatedAt:  chooseWorkflowUpdatedAt(statusResp),
	}
	return ws, nil
}

// ListWorkflows lists workflows with simple offset pagination semantics.
func (a *EngineAdapter) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*WorkflowSummary, string, error) {
	offset := 0
	if filter.PageToken != "" {
		parsed, err := strconv.Atoi(filter.PageToken)
		if err != nil || parsed < 0 {
			return nil, "", fmt.Errorf("invalid page token")
		}
		offset = parsed
	}
	limit := int(filter.PageSize)
	if limit <= 0 {
		limit = 50
	}

	workflows, total, err := a.engine.ListWorkflowsResponse(ctx, models.WorkflowFilter{
		Status: normalizeWorkflowFilterStatus(filter.StatusFilter),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		a.lastErrMsg = err.Error()
		return nil, "", err
	}

	summaries := make([]*WorkflowSummary, 0, len(workflows))
	for _, wf := range workflows {
		summaries = append(summaries, &WorkflowSummary{
			WorkflowID: wf.ID,
			Name:       wf.Name,
			Status:     wf.Status,
			CreatedAt:  wf.CreatedAt.Unix(),
			UpdatedAt:  chooseWorkflowUpdatedAt(wf),
		})
	}

	nextToken := ""
	if offset+len(workflows) < total {
		nextToken = strconv.Itoa(offset + len(workflows))
	}
	return summaries, nextToken, nil
}

// CancelWorkflow cancels a pending/running workflow.
func (a *EngineAdapter) CancelWorkflow(ctx context.Context, workflowID string, force bool) error {
	_ = force
	if err := a.engine.CancelWorkflowRequest(ctx, workflowID); err != nil {
		a.lastErrMsg = err.Error()
		return err
	}
	return nil
}

// GetTaskResult returns persisted task result semantics.
func (a *EngineAdapter) GetTaskResult(ctx context.Context, workflowID, taskID string) (*TaskResult, error) {
	resp, err := a.engine.GetTaskResultResponse(ctx, workflowID, taskID)
	if err != nil {
		a.lastErrMsg = err.Error()
		return nil, err
	}

	var resultData []byte
	if resp.Result != nil {
		resultData, err = json.Marshal(resp.Result)
		if err != nil {
			a.lastErrMsg = err.Error()
			return nil, err
		}
	}

	return &TaskResult{
		TaskID:     resp.TaskID,
		Status:     resp.Status,
		ResultData: resultData,
		ErrorMsg:   resp.Error,
	}, nil
}

// GetEngineState returns engine runtime state.
func (a *EngineAdapter) GetEngineState() string {
	return a.engine.State()
}

// GetEngineMetrics returns lightweight runtime metrics.
func (a *EngineAdapter) GetEngineMetrics() *EngineMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &EngineMetrics{
		MemoryUsageBytes: int64(memStats.Alloc),
		GoroutineCount:   int32(runtime.NumGoroutine()),
	}
}

// GetUptime returns adapter start time.
func (a *EngineAdapter) GetUptime() time.Time {
	return a.startTime
}

// GetLastError returns the most recent adapter error.
func (a *EngineAdapter) GetLastError() string {
	return a.lastErrMsg
}

// IsHealthy proxies engine health.
func (a *EngineAdapter) IsHealthy() bool {
	return a.engine.IsHealthy()
}

// UpdateConfig is not supported yet in local runtime mode.
func (a *EngineAdapter) UpdateConfig(ctx context.Context, updates map[string]string, persist bool) (map[string]string, error) {
	_ = ctx
	_ = persist
	if updates == nil {
		updates = map[string]string{}
	}
	return nil, errors.New("runtime config updates are not supported yet")
}

// ListClusterNodes returns an empty local cluster view.
func (a *EngineAdapter) ListClusterNodes(ctx context.Context) ([]*ClusterNode, error) {
	_ = ctx
	return []*ClusterNode{}, nil
}

// AddClusterNode is not supported yet in local runtime mode.
func (a *EngineAdapter) AddClusterNode(ctx context.Context, nodeID, address string) error {
	_ = ctx
	_ = nodeID
	_ = address
	return errors.New("cluster management is not supported yet")
}

// RemoveClusterNode is not supported yet in local runtime mode.
func (a *EngineAdapter) RemoveClusterNode(ctx context.Context, nodeID string) error {
	_ = ctx
	_ = nodeID
	return errors.New("cluster management is not supported yet")
}

// PauseWorkflows is not supported yet in local runtime mode.
func (a *EngineAdapter) PauseWorkflows(ctx context.Context) (int32, error) {
	_ = ctx
	return 0, errors.New("pause workflows is not supported yet")
}

// ResumeWorkflows is not supported yet in local runtime mode.
func (a *EngineAdapter) ResumeWorkflows(ctx context.Context) (int32, error) {
	_ = ctx
	return 0, errors.New("resume workflows is not supported yet")
}

// PurgeWorkflows is not supported yet in local runtime mode.
func (a *EngineAdapter) PurgeWorkflows(ctx context.Context, ageThresholdHours int32, dryRun bool) (int32, error) {
	_ = ctx
	_ = ageThresholdHours
	_ = dryRun
	return 0, errors.New("purge workflows is not supported yet")
}

// GetLaneStats returns an empty lane stats list for now.
func (a *EngineAdapter) GetLaneStats(ctx context.Context, laneName string) ([]*LaneStats, error) {
	_ = ctx
	_ = laneName
	return []*LaneStats{}, nil
}

// ExportMetrics returns a minimal metrics snapshot.
func (a *EngineAdapter) ExportMetrics(ctx context.Context, format string, prefixFilter string) (string, error) {
	_ = ctx
	_ = prefixFilter
	switch format {
	case "prometheus":
		return "", nil
	default:
		return "{}", nil
	}
}

func chooseWorkflowUpdatedAt(resp *models.WorkflowStatusResponse) int64 {
	if resp.CompletedAt != nil {
		return resp.CompletedAt.Unix()
	}
	if resp.StartedAt != nil {
		return resp.StartedAt.Unix()
	}
	return resp.CreatedAt.Unix()
}

func normalizeWorkflowFilterStatus(status string) string {
	switch status {
	case "WORKFLOW_STATUS_PENDING", "PENDING", "pending":
		return "pending"
	case "WORKFLOW_STATUS_RUNNING", "RUNNING", "running":
		return "running"
	case "WORKFLOW_STATUS_COMPLETED", "COMPLETED", "completed":
		return "completed"
	case "WORKFLOW_STATUS_FAILED", "FAILED", "failed":
		return "failed"
	case "WORKFLOW_STATUS_CANCELLED", "CANCELLED", "cancelled":
		return "cancelled"
	default:
		return ""
	}
}

// IsNotFoundError reports whether an adapter error is a storage not found error.
func IsNotFoundError(err error) bool {
	var notFound *storage.NotFoundError
	return errors.As(err, &notFound)
}
