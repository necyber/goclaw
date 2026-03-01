package memory

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/storage"
)

// TestMemoryStorageSuite runs the full storage test suite against MemoryStorage.
func TestMemoryStorageSuite(t *testing.T) {
	suite := &storage.StorageTestSuite{
		NewStorage: func(t *testing.T) storage.Storage {
			return NewMemoryStorage()
		},
	}

	suite.RunAllTests(t)
}

func TestMemoryStorage_SaveAndGetWorkflow(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	wf := &storage.WorkflowState{
		ID:          "wf-1",
		Name:        "test-workflow",
		Description: "test description",
		Status:      "pending",
		Tasks:       []models.TaskDefinition{{ID: "task-1", Name: "Task 1", Type: "function"}},
		Metadata:    map[string]string{"key": "value"},
	}

	// Save workflow
	err := s.SaveWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Get workflow
	retrieved, err := s.GetWorkflow(ctx, "wf-1")
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}

	if retrieved.ID != wf.ID {
		t.Errorf("Expected ID %s, got %s", wf.ID, retrieved.ID)
	}
	if retrieved.Name != wf.Name {
		t.Errorf("Expected Name %s, got %s", wf.Name, retrieved.Name)
	}
	if retrieved.Status != wf.Status {
		t.Errorf("Expected Status %s, got %s", wf.Status, retrieved.Status)
	}
}

func TestMemoryStorage_GetWorkflow_NotFound(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	_, err := s.GetWorkflow(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent workflow")
	}

	if _, ok := err.(*storage.NotFoundError); !ok {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

func TestMemoryStorage_ListWorkflows(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	// Save multiple workflows
	workflows := []*storage.WorkflowState{
		{ID: "wf-1", Name: "workflow-1", Status: "pending"},
		{ID: "wf-2", Name: "workflow-2", Status: "running"},
		{ID: "wf-3", Name: "workflow-3", Status: "completed"},
	}

	for _, wf := range workflows {
		if err := s.SaveWorkflow(ctx, wf); err != nil {
			t.Fatalf("SaveWorkflow failed: %v", err)
		}
	}

	// List all workflows
	result, total, err := s.ListWorkflows(ctx, &storage.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}
	if len(result) != 3 {
		t.Errorf("Expected 3 workflows, got %d", len(result))
	}
}

func TestMemoryStorage_ListWorkflows_WithFilter(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	// Save workflows with different statuses
	workflows := []*storage.WorkflowState{
		{ID: "wf-1", Name: "workflow-1", Status: "pending"},
		{ID: "wf-2", Name: "workflow-2", Status: "running"},
		{ID: "wf-3", Name: "workflow-3", Status: "pending"},
	}

	for _, wf := range workflows {
		if err := s.SaveWorkflow(ctx, wf); err != nil {
			t.Fatalf("SaveWorkflow failed: %v", err)
		}
	}

	// Filter by status
	result, total, err := s.ListWorkflows(ctx, &storage.WorkflowFilter{
		Status: []string{"pending"},
	})
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 workflows, got %d", len(result))
	}
}

func TestMemoryStorage_ListWorkflows_WithPagination(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	// Save 5 workflows
	for i := 1; i <= 5; i++ {
		wf := &storage.WorkflowState{
			ID:     string(rune('0' + i)),
			Name:   "workflow",
			Status: "pending",
		}
		if err := s.SaveWorkflow(ctx, wf); err != nil {
			t.Fatalf("SaveWorkflow failed: %v", err)
		}
	}

	// Get page 1 (limit 2)
	result, total, err := s.ListWorkflows(ctx, &storage.WorkflowFilter{
		Limit:  2,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 workflows, got %d", len(result))
	}

	// Get page 2 (limit 2, offset 2)
	result, total, err = s.ListWorkflows(ctx, &storage.WorkflowFilter{
		Limit:  2,
		Offset: 2,
	})
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 workflows, got %d", len(result))
	}
}

func TestMemoryStorage_DeleteWorkflow(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	wf := &storage.WorkflowState{
		ID:     "wf-1",
		Name:   "test-workflow",
		Status: "pending",
	}

	// Save workflow
	if err := s.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Delete workflow
	if err := s.DeleteWorkflow(ctx, "wf-1"); err != nil {
		t.Fatalf("DeleteWorkflow failed: %v", err)
	}

	// Verify deleted
	_, err := s.GetWorkflow(ctx, "wf-1")
	if err == nil {
		t.Fatal("Expected error for deleted workflow")
	}
}

func TestMemoryStorage_DeleteWorkflow_NotFound(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	err := s.DeleteWorkflow(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent workflow")
	}

	if _, ok := err.(*storage.NotFoundError); !ok {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

func TestMemoryStorage_SaveAndGetTask(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	// Create workflow first
	wf := &storage.WorkflowState{
		ID:     "wf-1",
		Name:   "test-workflow",
		Status: "running",
	}
	if err := s.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Save task
	now := time.Now()
	task := &storage.TaskState{
		ID:        "task-1",
		Name:      "Test Task",
		Status:    "running",
		StartedAt: &now,
	}

	if err := s.SaveTask(ctx, "wf-1", task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// Get task
	retrieved, err := s.GetTask(ctx, "wf-1", "task-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
	}
	if retrieved.Status != task.Status {
		t.Errorf("Expected Status %s, got %s", task.Status, retrieved.Status)
	}
}

func TestMemoryStorage_SaveTask_WorkflowNotFound(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	task := &storage.TaskState{
		ID:     "task-1",
		Name:   "Test Task",
		Status: "pending",
	}

	err := s.SaveTask(ctx, "nonexistent", task)
	if err == nil {
		t.Fatal("Expected error for nonexistent workflow")
	}

	if _, ok := err.(*storage.NotFoundError); !ok {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

func TestMemoryStorage_ListTasks(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	// Create workflow
	wf := &storage.WorkflowState{
		ID:     "wf-1",
		Name:   "test-workflow",
		Status: "running",
	}
	if err := s.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Save multiple tasks
	tasks := []*storage.TaskState{
		{ID: "task-1", Name: "Task 1", Status: "completed"},
		{ID: "task-2", Name: "Task 2", Status: "running"},
		{ID: "task-3", Name: "Task 3", Status: "pending"},
	}

	for _, task := range tasks {
		if err := s.SaveTask(ctx, "wf-1", task); err != nil {
			t.Fatalf("SaveTask failed: %v", err)
		}
	}

	// List tasks
	result, err := s.ListTasks(ctx, "wf-1")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(result))
	}
}

func TestMemoryStorage_DeleteWorkflow_CascadesTasks(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	// Create workflow with tasks
	wf := &storage.WorkflowState{
		ID:     "wf-1",
		Name:   "test-workflow",
		Status: "running",
	}
	if err := s.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	task := &storage.TaskState{
		ID:     "task-1",
		Name:   "Test Task",
		Status: "running",
	}
	if err := s.SaveTask(ctx, "wf-1", task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// Delete workflow
	if err := s.DeleteWorkflow(ctx, "wf-1"); err != nil {
		t.Fatalf("DeleteWorkflow failed: %v", err)
	}

	// Verify tasks are also deleted
	_, err := s.ListTasks(ctx, "wf-1")
	if err == nil {
		t.Fatal("Expected error for deleted workflow's tasks")
	}
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			wf := &storage.WorkflowState{
				ID:     string(rune('0' + id)),
				Name:   "workflow",
				Status: "pending",
			}
			s.SaveWorkflow(ctx, wf)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all workflows saved
	result, total, err := s.ListWorkflows(ctx, &storage.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}

	if total != 10 {
		t.Errorf("Expected 10 workflows, got %d", total)
	}
	if len(result) != 10 {
		t.Errorf("Expected 10 workflows in result, got %d", len(result))
	}
}

func TestMemoryStorage_GetWorkflowMutationIsolation(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	started := time.Now().UTC()

	original := &storage.WorkflowState{
		ID:     "wf-isolation",
		Name:   "workflow-isolation",
		Status: "running",
		Tasks: []models.TaskDefinition{
			{
				ID:           "task-1",
				Name:         "task-1",
				Type:         "function",
				Dependencies: []string{"task-0"},
				Config:       map[string]interface{}{"lane": "cpu", "meta": map[string]interface{}{"x": "y"}},
			},
		},
		TaskStatus: map[string]*storage.TaskState{
			"task-1": {
				ID:        "task-1",
				Name:      "task-1",
				Status:    "running",
				StartedAt: &started,
				Result:    map[string]interface{}{"before": "value"},
			},
		},
		Metadata: map[string]string{"owner": "team-a"},
	}
	if err := s.SaveWorkflow(ctx, original); err != nil {
		t.Fatalf("SaveWorkflow() error = %v", err)
	}

	read1, err := s.GetWorkflow(ctx, original.ID)
	if err != nil {
		t.Fatalf("GetWorkflow() error = %v", err)
	}
	read1.Metadata["owner"] = "mutated"
	read1.Tasks[0].Dependencies[0] = "mutated-dep"
	read1.Tasks[0].Config["lane"] = "io"
	read1.TaskStatus["task-1"].Status = "failed"
	if resultMap, ok := read1.TaskStatus["task-1"].Result.(map[string]interface{}); ok {
		resultMap["before"] = "mutated-result"
	}

	read2, err := s.GetWorkflow(ctx, original.ID)
	if err != nil {
		t.Fatalf("GetWorkflow(second) error = %v", err)
	}
	if read2.Metadata["owner"] != "team-a" {
		t.Fatalf("metadata mutated unexpectedly: %v", read2.Metadata)
	}
	if read2.Tasks[0].Dependencies[0] != "task-0" {
		t.Fatalf("task dependency mutated unexpectedly: %v", read2.Tasks[0].Dependencies)
	}
	if read2.Tasks[0].Config["lane"] != "cpu" {
		t.Fatalf("task config mutated unexpectedly: %v", read2.Tasks[0].Config)
	}
	if read2.TaskStatus["task-1"].Status != "running" {
		t.Fatalf("task status mutated unexpectedly: %s", read2.TaskStatus["task-1"].Status)
	}
	if !reflect.DeepEqual(read2.TaskStatus["task-1"].Result, map[string]interface{}{"before": "value"}) {
		t.Fatalf("task result mutated unexpectedly: %v", read2.TaskStatus["task-1"].Result)
	}
}

func TestMemoryStorage_TaskReadMutationIsolation(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	wf := &storage.WorkflowState{
		ID:     "wf-task-isolation",
		Name:   "wf-task-isolation",
		Status: "running",
	}
	if err := s.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow() error = %v", err)
	}
	task := &storage.TaskState{
		ID:     "task-1",
		Name:   "task-1",
		Status: "completed",
		Result: map[string]interface{}{"value": "original"},
	}
	if err := s.SaveTask(ctx, wf.ID, task); err != nil {
		t.Fatalf("SaveTask() error = %v", err)
	}

	readTask, err := s.GetTask(ctx, wf.ID, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if resultMap, ok := readTask.Result.(map[string]interface{}); ok {
		resultMap["value"] = "mutated"
	}

	listedTasks, err := s.ListTasks(ctx, wf.ID)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(listedTasks) != 1 {
		t.Fatalf("ListTasks() returned %d tasks, want 1", len(listedTasks))
	}
	if !reflect.DeepEqual(listedTasks[0].Result, map[string]interface{}{"value": "original"}) {
		t.Fatalf("stored task result mutated unexpectedly: %v", listedTasks[0].Result)
	}
}
