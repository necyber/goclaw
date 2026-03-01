package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/storage"
	"github.com/goclaw/goclaw/pkg/storage/memory"
)

type failingSaveWorkflowStorage struct {
	storage.Storage
	failWorkflowID string
}

func (s *failingSaveWorkflowStorage) SaveWorkflow(ctx context.Context, wf *storage.WorkflowState) error {
	if wf.ID == s.failWorkflowID {
		return errors.New("forced save workflow failure")
	}
	return s.Storage.SaveWorkflow(ctx, wf)
}

type orderedListStorage struct {
	storage.Storage
}

func (s *orderedListStorage) ListWorkflows(ctx context.Context, filter *storage.WorkflowFilter) ([]*storage.WorkflowState, int, error) {
	status := []string(nil)
	limit := 0
	offset := 0
	if filter != nil {
		status = filter.Status
		limit = filter.Limit
		offset = filter.Offset
	}

	all, _, err := s.Storage.ListWorkflows(ctx, &storage.WorkflowFilter{
		Status: status,
		Limit:  0,
		Offset: 0,
	})
	if err != nil {
		return nil, 0, err
	}

	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	total := len(all)
	if limit <= 0 {
		return all, total, nil
	}

	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func TestListRecoverableWorkflows_PaginatesResults(t *testing.T) {
	ctx := context.Background()
	store := &orderedListStorage{Storage: memory.NewMemoryStorage()}
	eng, err := New(minConfig(), nil, store)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	for i := 0; i < 125; i++ {
		id := fmt.Sprintf("recoverable-%03d", i)
		if err := store.SaveWorkflow(ctx, testWorkflowState(id, workflowStatusPending)); err != nil {
			t.Fatalf("seed workflow %s failed: %v", id, err)
		}
	}
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("terminal-%03d", i)
		if err := store.SaveWorkflow(ctx, testWorkflowState(id, workflowStatusCompleted)); err != nil {
			t.Fatalf("seed terminal workflow %s failed: %v", id, err)
		}
	}

	workflows, err := eng.listRecoverableWorkflows(ctx, 50)
	if err != nil {
		t.Fatalf("listRecoverableWorkflows() error = %v", err)
	}
	if got, want := len(workflows), 125; got != want {
		t.Fatalf("recoverable workflows = %d, want %d", got, want)
	}
}

func TestRecoverWorkflows_ResubmitsRecoveredWorkflow(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemoryStorage()
	wf := testWorkflowState("wf-recover", workflowStatusRunning)
	wf.StartedAt = timePtr(time.Now().UTC())
	if err := store.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("seed workflow failed: %v", err)
	}

	eng, err := New(minConfig(), nil, store)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	waitForWorkflowStatus(t, store, wf.ID, workflowStatusCompleted, 3*time.Second)
}

func TestRecoverWorkflows_SkipsAlreadyExecutingWorkflow(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemoryStorage()
	eng, err := New(minConfig(), nil, store)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	wf := testWorkflowState("wf-active", workflowStatusPending)
	if err := store.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("seed workflow failed: %v", err)
	}
	exec := &workflowExecution{
		workflowID: wf.ID,
		cancel:     func() {},
		done:       make(chan struct{}),
		wfState:    wf,
	}
	eng.registerExecution(exec)
	defer eng.unregisterExecution(wf.ID)

	if err := eng.RecoverWorkflows(ctx); err != nil {
		t.Fatalf("RecoverWorkflows() error = %v", err)
	}

	persisted, err := store.GetWorkflow(ctx, wf.ID)
	if err != nil {
		t.Fatalf("GetWorkflow() error = %v", err)
	}
	if persisted.Status != workflowStatusPending {
		t.Fatalf("workflow status = %s, want %s", persisted.Status, workflowStatusPending)
	}
}

func TestRecoverWorkflows_AggregatesErrorsAndContinues(t *testing.T) {
	ctx := context.Background()
	base := memory.NewMemoryStorage()
	eng, err := New(minConfig(), nil, base)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(context.Background())

	good := testWorkflowState("wf-good", workflowStatusPending)
	bad := testWorkflowState("wf-bad", workflowStatusPending)
	if err := base.SaveWorkflow(ctx, good); err != nil {
		t.Fatalf("seed good workflow failed: %v", err)
	}
	if err := base.SaveWorkflow(ctx, bad); err != nil {
		t.Fatalf("seed bad workflow failed: %v", err)
	}

	eng.storage = &failingSaveWorkflowStorage{
		Storage:        base,
		failWorkflowID: bad.ID,
	}

	err = eng.RecoverWorkflows(ctx)
	if err == nil {
		t.Fatal("expected recovery error for forced save failure")
	}

	waitForWorkflowStatus(t, base, good.ID, workflowStatusCompleted, 3*time.Second)

	persistedBad, err := base.GetWorkflow(ctx, bad.ID)
	if err != nil {
		t.Fatalf("GetWorkflow(bad) error = %v", err)
	}
	if persistedBad.Status != workflowStatusPending {
		t.Fatalf("bad workflow status = %s, want %s", persistedBad.Status, workflowStatusPending)
	}
}

func testWorkflowState(id, status string) *storage.WorkflowState {
	now := time.Now().UTC()
	task := &storage.TaskState{
		ID:     "task-1",
		Name:   "task-1",
		Status: taskStatusPending,
	}
	if status == workflowStatusRunning {
		task.Status = taskStatusRunning
		task.StartedAt = timePtr(now)
	}
	return &storage.WorkflowState{
		ID:     id,
		Name:   id,
		Status: status,
		Tasks: []models.TaskDefinition{
			{ID: "task-1", Name: "task-1", Type: "function"},
		},
		TaskStatus: map[string]*storage.TaskState{
			task.ID: task,
		},
		CreatedAt: now,
	}
}

func waitForWorkflowStatus(t *testing.T, store storage.Storage, workflowID, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		wf, err := store.GetWorkflow(context.Background(), workflowID)
		if err == nil && wf.Status == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	wf, err := store.GetWorkflow(context.Background(), workflowID)
	if err != nil {
		t.Fatalf("workflow %s not found: %v", workflowID, err)
	}
	t.Fatalf("workflow %s status = %s, want %s", workflowID, wf.Status, want)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
