package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/storage/memory"
)

type workflowEventRecord struct {
	WorkflowID string
	OldState   string
	NewState   string
}

type taskEventRecord struct {
	WorkflowID string
	TaskID     string
	OldState   string
	NewState   string
}

type mockEventBroadcaster struct {
	mu             sync.Mutex
	workflowEvents []workflowEventRecord
	taskEvents     []taskEventRecord
}

func (m *mockEventBroadcaster) BroadcastWorkflowStateChanged(
	workflowID, _name, oldState, newState string,
	_ time.Time,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflowEvents = append(m.workflowEvents, workflowEventRecord{
		WorkflowID: workflowID,
		OldState:   oldState,
		NewState:   newState,
	})
}

func (m *mockEventBroadcaster) BroadcastTaskStateChanged(
	workflowID, taskID, _taskName, oldState, newState, _errorMessage string,
	_ any,
	_ time.Time,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskEvents = append(m.taskEvents, taskEventRecord{
		WorkflowID: workflowID,
		TaskID:     taskID,
		OldState:   oldState,
		NewState:   newState,
	})
}

func TestEngine_EmitsWorkflowAndTaskEvents(t *testing.T) {
	cfg := minConfig()
	mockEvents := &mockEventBroadcaster{}

	eng, err := New(cfg, nil, memory.NewMemoryStorage(), WithEventBroadcaster(mockEvents))
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	defer eng.Stop(ctx)

	wf := &Workflow{
		ID: "wf-events",
		Tasks: []*dag.Task{
			{ID: "task-1", Name: "task-1", Agent: "test"},
		},
		TaskFns: map[string]func(context.Context) error{
			"task-1": func(context.Context) error { return nil },
		},
	}

	if _, err := eng.Submit(ctx, wf); err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	mockEvents.mu.Lock()
	defer mockEvents.mu.Unlock()

	if len(mockEvents.workflowEvents) < 2 {
		t.Fatalf("expected at least 2 workflow events, got %d", len(mockEvents.workflowEvents))
	}

	hasRunning := false
	hasCompleted := false
	for _, event := range mockEvents.workflowEvents {
		if event.WorkflowID != wf.ID {
			continue
		}
		if event.NewState == "running" {
			hasRunning = true
		}
		if event.NewState == "completed" {
			hasCompleted = true
		}
	}
	if !hasRunning || !hasCompleted {
		t.Fatalf("expected running+completed workflow events, got %+v", mockEvents.workflowEvents)
	}

	if len(mockEvents.taskEvents) == 0 {
		t.Fatal("expected task events to be emitted")
	}
}

func TestEngine_EmitsCancelEvent(t *testing.T) {
	cfg := minConfig()
	mockEvents := &mockEventBroadcaster{}

	eng, err := New(cfg, nil, memory.NewMemoryStorage(), WithEventBroadcaster(mockEvents))
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx := context.Background()
	req := &models.WorkflowRequest{
		Name: "cancel-me",
		Tasks: []models.TaskDefinition{
			{
				ID:   "t1",
				Name: "task-1",
				Type: "function",
			},
		},
	}

	workflowID, err := eng.SubmitWorkflowRequest(ctx, req)
	if err != nil {
		t.Fatalf("submit workflow request failed: %v", err)
	}
	if err := eng.CancelWorkflowRequest(ctx, workflowID); err != nil {
		t.Fatalf("cancel workflow request failed: %v", err)
	}

	mockEvents.mu.Lock()
	defer mockEvents.mu.Unlock()

	foundCancelled := false
	for _, event := range mockEvents.workflowEvents {
		if event.WorkflowID == workflowID && event.NewState == "cancelled" {
			foundCancelled = true
			break
		}
	}
	if !foundCancelled {
		t.Fatalf("expected cancelled event for workflow %s", workflowID)
	}
}
