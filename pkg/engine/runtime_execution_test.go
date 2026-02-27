package engine

import (
	"context"
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
	taskRetryCount     int
}

func newCaptureMetrics() *captureMetrics {
	return &captureMetrics{
		workflowSubmission: make(map[string]int),
		taskExecution:      make(map[string]int),
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
func (m *captureMetrics) IncActiveWorkflows(status string) { _ = status }
func (m *captureMetrics) DecActiveWorkflows(status string) { _ = status }
func (m *captureMetrics) RecordTaskExecution(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskExecution[status]++
}
func (m *captureMetrics) RecordTaskDuration(duration time.Duration) { _ = duration }
func (m *captureMetrics) RecordTaskRetry()                          { m.mu.Lock(); m.taskRetryCount++; m.mu.Unlock() }
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
}

func TestSubmitWorkflowRuntime_CancelPrecedence(t *testing.T) {
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
