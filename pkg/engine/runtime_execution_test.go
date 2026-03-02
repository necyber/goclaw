package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/storage"
	"github.com/goclaw/goclaw/pkg/storage/memory"
)

type captureMetrics struct {
	mu                 sync.Mutex
	workflowSubmission map[string]int
	taskExecution      map[string]int
	taskTypes          map[string]int
	activeInc          map[string]int
	activeDec          map[string]int
	taskRetryCount     int
}

func newCaptureMetrics() *captureMetrics {
	return &captureMetrics{
		workflowSubmission: make(map[string]int),
		taskExecution:      make(map[string]int),
		taskTypes:          make(map[string]int),
		activeInc:          make(map[string]int),
		activeDec:          make(map[string]int),
	}
}

func (m *captureMetrics) RecordWorkflowSubmission(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflowSubmission[status]++
}

func (m *captureMetrics) RecordWorkflowDuration(status string, duration time.Duration) {
	_ = status
	_ = duration
}
func (m *captureMetrics) IncActiveWorkflows(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeInc[status]++
}
func (m *captureMetrics) DecActiveWorkflows(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeDec[status]++
}
func (m *captureMetrics) RecordTaskExecution(status string, taskType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskExecution[status]++
	m.taskTypes[taskType]++
}
func (m *captureMetrics) RecordTaskDuration(taskType string, duration time.Duration) {
	_ = taskType
	_ = duration
}
func (m *captureMetrics) RecordTaskRetry(taskType string) {
	_ = taskType
	m.mu.Lock()
	m.taskRetryCount++
	m.mu.Unlock()
}
func (m *captureMetrics) IncQueueDepth(laneName string)             { _ = laneName }
func (m *captureMetrics) DecQueueDepth(laneName string)             { _ = laneName }
func (m *captureMetrics) RecordWaitDuration(laneName string, duration time.Duration) {
	_ = laneName
	_ = duration
}
func (m *captureMetrics) RecordThroughput(laneName string) { _ = laneName }
func (m *captureMetrics) workflowCount(status string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.workflowSubmission[status]
}
func (m *captureMetrics) taskCount(status string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.taskExecution[status]
}
func (m *captureMetrics) activeNet(status string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeInc[status] - m.activeDec[status]
}
func (m *captureMetrics) taskTypeCount(taskType string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.taskTypes[taskType]
}

type verifyingBroadcaster struct {
	store          storage.Storage
	mu             sync.Mutex
	workflowEvents []string
	taskEvents     map[string][]string
	violations     []string
}

func newVerifyingBroadcaster(store storage.Storage) *verifyingBroadcaster {
	return &verifyingBroadcaster{
		store:      store,
		taskEvents: make(map[string][]string),
	}
}

func (b *verifyingBroadcaster) BroadcastWorkflowStateChanged(workflowID, _name, _oldState, newState string, _updatedAt time.Time) {
	wf, err := b.store.GetWorkflow(context.Background(), workflowID)
	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		b.violations = append(b.violations, "workflow fetch error")
		return
	}
	if wf.Status != newState {
		b.violations = append(b.violations, "workflow state emitted before persistence")
	}
	b.workflowEvents = append(b.workflowEvents, newState)
}

func (b *verifyingBroadcaster) BroadcastTaskStateChanged(
	workflowID, taskID, _taskName, _oldState, newState, _errorMessage string,
	_result any,
	_updatedAt time.Time,
) {
	task, err := b.store.GetTask(context.Background(), workflowID, taskID)
	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		b.violations = append(b.violations, "task fetch error")
		return
	}
	if task.Status != newState {
		b.violations = append(b.violations, "task state emitted before persistence")
	}
	b.taskEvents[taskID] = append(b.taskEvents[taskID], newState)
}

func TestSubmitWorkflowRuntime_PersistsAndEmitsTransitions(t *testing.T) {
	cfg := minConfig()
	store := memory.NewMemoryStorage()
	metrics := newCaptureMetrics()
	broadcaster := newVerifyingBroadcaster(store)

	eng, err := New(cfg, nil, store, WithMetrics(metrics), WithEventBroadcaster(broadcaster))
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	req := &models.WorkflowRequest{
		Name: "runtime-sync",
		Tasks: []models.TaskDefinition{
			{ID: "t1", Name: "task-1", Type: "function"},
		},
	}

	resp, err := eng.SubmitWorkflowRuntime(context.Background(), req, SubmitWorkflowOptions{
		Mode: SubmissionModeSync,
		TaskFns: map[string]func(context.Context) error{
			"t1": func(context.Context) error { return nil },
		},
	})
	if err != nil {
		t.Fatalf("SubmitWorkflowRuntime() error = %v", err)
	}
	if resp.Status != workflowStatusCompleted {
		t.Fatalf("workflow status = %s, want %s", resp.Status, workflowStatusCompleted)
	}

	persisted, err := store.GetWorkflow(context.Background(), resp.ID)
	if err != nil {
		t.Fatalf("GetWorkflow() error = %v", err)
	}
	if persisted.Status != workflowStatusCompleted {
		t.Fatalf("persisted status = %s, want %s", persisted.Status, workflowStatusCompleted)
	}
	if persisted.StartedAt == nil || persisted.CompletedAt == nil {
		t.Fatal("expected started_at and completed_at to be persisted")
	}

	broadcaster.mu.Lock()
	defer broadcaster.mu.Unlock()
	if len(broadcaster.violations) != 0 {
		t.Fatalf("unexpected persistence violations: %v", broadcaster.violations)
	}
	wantWorkflow := []string{"pending", "scheduled", "running", "completed"}
	if len(broadcaster.workflowEvents) != len(wantWorkflow) {
		t.Fatalf("workflow events = %v, want %v", broadcaster.workflowEvents, wantWorkflow)
	}
	for i := range wantWorkflow {
		if broadcaster.workflowEvents[i] != wantWorkflow[i] {
			t.Fatalf("workflow event[%d] = %s, want %s", i, broadcaster.workflowEvents[i], wantWorkflow[i])
		}
	}
	wantTask := []string{"scheduled", "running", "completed"}
	taskEvents := broadcaster.taskEvents["t1"]
	if len(taskEvents) != len(wantTask) {
		t.Fatalf("task events = %v, want %v", taskEvents, wantTask)
	}
	for i := range wantTask {
		if taskEvents[i] != wantTask[i] {
			t.Fatalf("task event[%d] = %s, want %s", i, taskEvents[i], wantTask[i])
		}
	}

	if metrics.workflowCount("pending") == 0 {
		t.Fatal("expected pending workflow submission metric")
	}
	if metrics.workflowCount("completed") == 0 {
		t.Fatal("expected completed workflow submission metric")
	}
	if metrics.taskCount("completed") != 1 {
		t.Fatalf("expected one completed task metric, got %d", metrics.taskCount("completed"))
	}
	if metrics.taskTypeCount("function") != 1 {
		t.Fatalf("expected task_type=function metric, got %d", metrics.taskTypeCount("function"))
	}
	if metrics.activeNet("pending") != 0 {
		t.Fatalf("expected pending active gauge to balance to 0, got %d", metrics.activeNet("pending"))
	}
	if metrics.activeNet("running") != 0 {
		t.Fatalf("expected running active gauge to balance to 0, got %d", metrics.activeNet("running"))
	}
}

func TestSubmitWorkflowRuntime_CancelPrecedence(t *testing.T) {
	cfg := minConfig()
	store := memory.NewMemoryStorage()
	metrics := newCaptureMetrics()

	eng, err := New(cfg, nil, store, WithMetrics(metrics))
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	req := &models.WorkflowRequest{
		Name: "cancel-precedence",
		Tasks: []models.TaskDefinition{
			{ID: "t1", Name: "task-1", Type: "function"},
		},
	}

	resp, err := eng.SubmitWorkflowRuntime(context.Background(), req, SubmitWorkflowOptions{
		Mode: SubmissionModeAsync,
		TaskFns: map[string]func(context.Context) error{
			"t1": func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("SubmitWorkflowRuntime() error = %v", err)
	}

	if err := waitWorkflowStatus(eng, resp.ID, workflowStatusRunning, 2*time.Second); err != nil {
		t.Fatalf("workflow did not reach running state: %v", err)
	}
	if err := eng.CancelWorkflowRequest(context.Background(), resp.ID); err != nil {
		t.Fatalf("CancelWorkflowRequest() error = %v", err)
	}
	if err := waitWorkflowStatus(eng, resp.ID, workflowStatusCancelled, 2*time.Second); err != nil {
		t.Fatalf("workflow did not reach cancelled state: %v", err)
	}

	taskResp, err := eng.GetTaskResultResponse(context.Background(), resp.ID, "t1")
	if err != nil {
		t.Fatalf("GetTaskResultResponse() error = %v", err)
	}
	if taskResp.Status != taskStatusCancelled {
		t.Fatalf("task status = %s, want %s", taskResp.Status, taskStatusCancelled)
	}
	if metrics.activeNet("pending") != 0 {
		t.Fatalf("expected pending active gauge to balance to 0 after cancel, got %d", metrics.activeNet("pending"))
	}
	if metrics.activeNet("running") != 0 {
		t.Fatalf("expected running active gauge to balance to 0 after cancel, got %d", metrics.activeNet("running"))
	}
}

func waitWorkflowStatus(eng *Engine, workflowID, want string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := eng.GetWorkflowStatusResponse(context.Background(), workflowID)
		if err == nil && status.Status == want {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return context.DeadlineExceeded
}

func TestSubmitWorkflowRuntime_DependencyField_Dependencies(t *testing.T) {
	cfg := minConfig()
	store := memory.NewMemoryStorage()

	eng, err := New(cfg, nil, store)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	done := make(chan struct{})
	req := &models.WorkflowRequest{
		Name: "deps-canonical",
		Tasks: []models.TaskDefinition{
			{ID: "t1", Name: "task-1", Type: "function"},
			{ID: "t2", Name: "task-2", Type: "function", Dependencies: []string{"t1"}},
		},
	}

	resp, err := eng.SubmitWorkflowRuntime(context.Background(), req, SubmitWorkflowOptions{
		Mode: SubmissionModeSync,
		TaskFns: map[string]func(context.Context) error{
			"t1": func(context.Context) error {
				time.Sleep(40 * time.Millisecond)
				close(done)
				return nil
			},
			"t2": func(context.Context) error {
				select {
				case <-done:
					return nil
				default:
					return fmt.Errorf("task t2 started before dependency completed")
				}
			},
		},
	})
	if err != nil {
		t.Fatalf("SubmitWorkflowRuntime() error = %v", err)
	}
	if resp.Status != workflowStatusCompleted {
		t.Fatalf("workflow status = %s, want %s", resp.Status, workflowStatusCompleted)
	}
}

func TestSubmitWorkflowRuntime_DependencyField_DependsOnAlias(t *testing.T) {
	cfg := minConfig()
	store := memory.NewMemoryStorage()

	eng, err := New(cfg, nil, store)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	done := make(chan struct{})
	req := &models.WorkflowRequest{
		Name: "deps-alias",
		Tasks: []models.TaskDefinition{
			{ID: "t1", Name: "task-1", Type: "function"},
			{ID: "t2", Name: "task-2", Type: "function", DependsOn: []string{"t1"}},
		},
	}

	resp, err := eng.SubmitWorkflowRuntime(context.Background(), req, SubmitWorkflowOptions{
		Mode: SubmissionModeSync,
		TaskFns: map[string]func(context.Context) error{
			"t1": func(context.Context) error {
				time.Sleep(40 * time.Millisecond)
				close(done)
				return nil
			},
			"t2": func(context.Context) error {
				select {
				case <-done:
					return nil
				default:
					return fmt.Errorf("task t2 started before dependency completed")
				}
			},
		},
	})
	if err != nil {
		t.Fatalf("SubmitWorkflowRuntime() error = %v", err)
	}
	if resp.Status != workflowStatusCompleted {
		t.Fatalf("workflow status = %s, want %s", resp.Status, workflowStatusCompleted)
	}
}

func TestTransitionTask_PersistsCompletedResultPayload(t *testing.T) {
	cfg := minConfig()
	store := memory.NewMemoryStorage()

	eng, err := New(cfg, nil, store)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	resp, err := eng.SubmitWorkflowRuntime(context.Background(), &models.WorkflowRequest{
		Name: "result-persistence",
		Tasks: []models.TaskDefinition{
			{ID: "t1", Name: "task-1", Type: "function"},
		},
	}, SubmitWorkflowOptions{Mode: SubmissionModeAsync})
	if err != nil {
		t.Fatalf("SubmitWorkflowRuntime() error = %v", err)
	}

	wfState, err := store.GetWorkflow(context.Background(), resp.ID)
	if err != nil {
		t.Fatalf("GetWorkflow() error = %v", err)
	}
	exec := &workflowExecution{workflowID: resp.ID, wfState: wfState}

	if err := eng.transitionTask(exec, "t1", TaskStatePending, TaskStateScheduled, TaskResult{}); err != nil {
		t.Fatalf("transition to scheduled error = %v", err)
	}
	started := time.Now().Add(-200 * time.Millisecond)
	if err := eng.transitionTask(exec, "t1", TaskStateScheduled, TaskStateRunning, TaskResult{StartedAt: started}); err != nil {
		t.Fatalf("transition to running error = %v", err)
	}

	payload := map[string]any{"output": "ok", "count": 1}
	if err := eng.transitionTask(exec, "t1", TaskStateRunning, TaskStateCompleted, TaskResult{
		EndedAt: time.Now(),
		Result:  payload,
	}); err != nil {
		t.Fatalf("transition to completed error = %v", err)
	}

	taskState, err := store.GetTask(context.Background(), resp.ID, "t1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if taskState.Status != taskStatusCompleted {
		t.Fatalf("task status = %s, want %s", taskState.Status, taskStatusCompleted)
	}
	resultMap, ok := taskState.Result.(map[string]any)
	if !ok {
		t.Fatalf("task result type = %T, want map[string]any", taskState.Result)
	}
	if resultMap["output"] != "ok" {
		t.Fatalf("result output = %v, want ok", resultMap["output"])
	}

	resultResp, err := eng.GetTaskResultResponse(context.Background(), resp.ID, "t1")
	if err != nil {
		t.Fatalf("GetTaskResultResponse() error = %v", err)
	}
	if resultResp.Result == nil {
		t.Fatal("expected non-nil task result payload")
	}
}

func TestTransitionTask_ClearsResultForNonCompletedTerminalStates(t *testing.T) {
	cfg := minConfig()
	store := memory.NewMemoryStorage()

	eng, err := New(cfg, nil, store)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	resp, err := eng.SubmitWorkflowRuntime(context.Background(), &models.WorkflowRequest{
		Name: "result-clear",
		Tasks: []models.TaskDefinition{
			{ID: "t1", Name: "task-1", Type: "function"},
		},
	}, SubmitWorkflowOptions{Mode: SubmissionModeAsync})
	if err != nil {
		t.Fatalf("SubmitWorkflowRuntime() error = %v", err)
	}

	wfState, err := store.GetWorkflow(context.Background(), resp.ID)
	if err != nil {
		t.Fatalf("GetWorkflow() error = %v", err)
	}
	wfState.TaskStatus["t1"].Result = map[string]any{"stale": true}
	if err := store.SaveTask(context.Background(), resp.ID, wfState.TaskStatus["t1"]); err != nil {
		t.Fatalf("SaveTask() seed result error = %v", err)
	}

	exec := &workflowExecution{workflowID: resp.ID, wfState: wfState}
	if err := eng.transitionTask(exec, "t1", TaskStatePending, TaskStateScheduled, TaskResult{}); err != nil {
		t.Fatalf("transition to scheduled error = %v", err)
	}
	if err := eng.transitionTask(exec, "t1", TaskStateScheduled, TaskStateRunning, TaskResult{StartedAt: time.Now().Add(-100 * time.Millisecond)}); err != nil {
		t.Fatalf("transition to running error = %v", err)
	}
	if err := eng.transitionTask(exec, "t1", TaskStateRunning, TaskStateFailed, TaskResult{
		EndedAt: time.Now(),
		Error:   errors.New("boom"),
		Result:  map[string]any{"should": "drop"},
	}); err != nil {
		t.Fatalf("transition to failed error = %v", err)
	}

	taskState, err := store.GetTask(context.Background(), resp.ID, "t1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if taskState.Result != nil {
		t.Fatalf("expected nil result for failed task, got %v", taskState.Result)
	}

	resultResp, err := eng.GetTaskResultResponse(context.Background(), resp.ID, "t1")
	if err != nil {
		t.Fatalf("GetTaskResultResponse() error = %v", err)
	}
	if resultResp.Result != nil {
		t.Fatalf("expected nil API result for failed task, got %v", resultResp.Result)
	}
}

func TestCancelWorkflowRequest_GracefulTimeoutExpiresToFailed(t *testing.T) {
	cfg := minConfig()
	cfg.Orchestration.CancellationTimeout = 50 * time.Millisecond
	store := memory.NewMemoryStorage()

	eng, err := New(cfg, nil, store)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	req := &models.WorkflowRequest{
		Name: "cancel-timeout-failed",
		Tasks: []models.TaskDefinition{
			{ID: "t1", Name: "task-1", Type: "function"},
		},
	}
	resp, err := eng.SubmitWorkflowRuntime(context.Background(), req, SubmitWorkflowOptions{
		Mode: SubmissionModeAsync,
		TaskFns: map[string]func(context.Context) error{
			"t1": func(context.Context) error {
				// Simulate non-cooperative task that exceeds graceful cancellation timeout.
				time.Sleep(200 * time.Millisecond)
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("SubmitWorkflowRuntime() error = %v", err)
	}
	if err := waitWorkflowStatus(eng, resp.ID, workflowStatusRunning, 2*time.Second); err != nil {
		t.Fatalf("workflow did not reach running state: %v", err)
	}

	start := time.Now()
	if err := eng.CancelWorkflowRequest(context.Background(), resp.ID); err != nil {
		t.Fatalf("CancelWorkflowRequest() error = %v", err)
	}
	elapsed := time.Since(start)
	if elapsed > 150*time.Millisecond {
		t.Fatalf("cancel exceeded graceful timeout bound, elapsed=%v", elapsed)
	}

	if err := waitWorkflowStatus(eng, resp.ID, workflowStatusFailed, 2*time.Second); err != nil {
		t.Fatalf("workflow did not reach failed state after cancellation timeout: %v", err)
	}
	taskResp, err := eng.GetTaskResultResponse(context.Background(), resp.ID, "t1")
	if err != nil {
		t.Fatalf("GetTaskResultResponse() error = %v", err)
	}
	if taskResp.Status != taskStatusFailed {
		t.Fatalf("task status = %s, want %s", taskResp.Status, taskStatusFailed)
	}
}

func TestCancelWorkflowRequest_CompletesWithinGracefulTimeout(t *testing.T) {
	cfg := minConfig()
	cfg.Orchestration.CancellationTimeout = 500 * time.Millisecond
	store := memory.NewMemoryStorage()

	eng, err := New(cfg, nil, store)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	req := &models.WorkflowRequest{
		Name: "cancel-graceful",
		Tasks: []models.TaskDefinition{
			{ID: "t1", Name: "task-1", Type: "function"},
		},
	}
	resp, err := eng.SubmitWorkflowRuntime(context.Background(), req, SubmitWorkflowOptions{
		Mode: SubmissionModeAsync,
		TaskFns: map[string]func(context.Context) error{
			"t1": func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("SubmitWorkflowRuntime() error = %v", err)
	}
	if err := waitWorkflowStatus(eng, resp.ID, workflowStatusRunning, 2*time.Second); err != nil {
		t.Fatalf("workflow did not reach running state: %v", err)
	}

	start := time.Now()
	if err := eng.CancelWorkflowRequest(context.Background(), resp.ID); err != nil {
		t.Fatalf("CancelWorkflowRequest() error = %v", err)
	}
	elapsed := time.Since(start)
	if elapsed > cfg.Orchestration.CancellationTimeout {
		t.Fatalf("cancel exceeded configured graceful timeout, elapsed=%v timeout=%v", elapsed, cfg.Orchestration.CancellationTimeout)
	}

	if err := waitWorkflowStatus(eng, resp.ID, workflowStatusCancelled, 2*time.Second); err != nil {
		t.Fatalf("workflow did not reach cancelled state: %v", err)
	}
}

func TestTransitionTask_MetricsDistinguishTimeoutFromUserCancellation(t *testing.T) {
	cfg := minConfig()
	store := memory.NewMemoryStorage()
	metrics := newCaptureMetrics()

	eng, err := New(cfg, nil, store, WithMetrics(metrics))
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	started := time.Now().Add(-1 * time.Second).UTC()
	wfState := &storage.WorkflowState{
		ID:        "wf-metric-labels",
		Name:      "wf-metric-labels",
		Status:    workflowStatusRunning,
		CreatedAt: started.Add(-1 * time.Second),
		StartedAt: &started,
		Tasks: []models.TaskDefinition{
			{ID: "user-cancel", Name: "user-cancel", Type: "function"},
			{ID: "timeout-cancel", Name: "timeout-cancel", Type: "function"},
		},
		TaskStatus: map[string]*storage.TaskState{
			"user-cancel": {
				ID:        "user-cancel",
				Name:      "user-cancel",
				Status:    taskStatusRunning,
				StartedAt: &started,
			},
			"timeout-cancel": {
				ID:        "timeout-cancel",
				Name:      "timeout-cancel",
				Status:    taskStatusRunning,
				StartedAt: &started,
			},
		},
	}
	if err := store.SaveWorkflow(context.Background(), wfState); err != nil {
		t.Fatalf("SaveWorkflow() error = %v", err)
	}
	for taskID := range wfState.TaskStatus {
		if err := store.SaveTask(context.Background(), wfState.ID, wfState.TaskStatus[taskID]); err != nil {
			t.Fatalf("SaveTask(%s) error = %v", taskID, err)
		}
	}

	exec := &workflowExecution{workflowID: wfState.ID, wfState: wfState}
	if err := eng.transitionTask(exec, "user-cancel", TaskStateRunning, TaskStateCancelled, TaskResult{
		EndedAt: time.Now().UTC(),
		Error:   context.Canceled,
	}); err != nil {
		t.Fatalf("transitionTask(user-cancel) error = %v", err)
	}
	if err := eng.transitionTask(exec, "timeout-cancel", TaskStateRunning, TaskStateCancelled, TaskResult{
		EndedAt: time.Now().UTC(),
		Error:   context.DeadlineExceeded,
	}); err != nil {
		t.Fatalf("transitionTask(timeout-cancel) error = %v", err)
	}

	if got := metrics.taskCount(taskStatusCancelled); got != 1 {
		t.Fatalf("cancelled metric = %d, want 1", got)
	}
	if got := metrics.taskCount("failed_timeout"); got != 1 {
		t.Fatalf("failed_timeout metric = %d, want 1", got)
	}
	if got := metrics.taskCount(taskStatusFailed); got != 0 {
		t.Fatalf("failed metric = %d, want 0", got)
	}

	// Duplicate terminal callback for the same attempt must not double-count.
	if err := eng.transitionTask(exec, "timeout-cancel", TaskStateRunning, TaskStateCancelled, TaskResult{
		EndedAt: time.Now().UTC(),
		Error:   context.DeadlineExceeded,
	}); err != nil {
		t.Fatalf("duplicate transitionTask(timeout-cancel) error = %v", err)
	}
	if got := metrics.taskCount("failed_timeout"); got != 1 {
		t.Fatalf("failed_timeout metric after duplicate terminal callback = %d, want 1", got)
	}
}
