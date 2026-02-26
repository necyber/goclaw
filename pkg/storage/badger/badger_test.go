package badger

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/storage"
)

// TestBadgerStorageSuite runs the full storage test suite against BadgerStorage.
func TestBadgerStorageSuite(t *testing.T) {
	suite := &storage.StorageTestSuite{
		NewStorage: func(t *testing.T) storage.Storage {
			tmpDir, err := os.MkdirTemp("", "badger-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}

			t.Cleanup(func() {
				os.RemoveAll(tmpDir)
			})

			config := &Config{
				Path:              tmpDir,
				SyncWrites:        false,
				ValueLogFileSize:  1 << 20,
				NumVersionsToKeep: 1,
			}

			db, err := NewBadgerStorage(config)
			if err != nil {
				t.Fatalf("Failed to create BadgerStorage: %v", err)
			}

			return db
		},
	}

	suite.RunAllTests(t)
}

func setupTestDB(t *testing.T) (*BadgerStorage, func()) {
	tmpDir, err := os.MkdirTemp("", "badger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	config := &Config{
		Path:              tmpDir,
		SyncWrites:        false,   // Faster for tests
		ValueLogFileSize:  1 << 20, // 1MB
		NumVersionsToKeep: 1,
	}

	db, err := NewBadgerStorage(config)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create BadgerStorage: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestBadgerStorage_SaveAndGetWorkflow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	wf := &storage.WorkflowState{
		ID:          "wf-1",
		Name:        "test-workflow",
		Description: "test description",
		Status:      "pending",
		Tasks:       []models.TaskDefinition{{ID: "task-1", Name: "Task 1", Type: "function"}},
		Metadata:    map[string]string{"key": "value"},
		CreatedAt:   time.Now(),
	}

	// Save workflow
	err := db.SaveWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Get workflow
	retrieved, err := db.GetWorkflow(ctx, "wf-1")
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

func TestBadgerStorage_GetWorkflow_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	_, err := db.GetWorkflow(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent workflow")
	}

	if _, ok := err.(*storage.NotFoundError); !ok {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

func TestBadgerStorage_ListWorkflows(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save multiple workflows
	workflows := []*storage.WorkflowState{
		{ID: "wf-1", Name: "workflow-1", Status: "pending", CreatedAt: time.Now()},
		{ID: "wf-2", Name: "workflow-2", Status: "running", CreatedAt: time.Now()},
		{ID: "wf-3", Name: "workflow-3", Status: "completed", CreatedAt: time.Now()},
	}

	for _, wf := range workflows {
		if err := db.SaveWorkflow(ctx, wf); err != nil {
			t.Fatalf("SaveWorkflow failed: %v", err)
		}
	}

	// List all workflows
	result, total, err := db.ListWorkflows(ctx, &storage.WorkflowFilter{})
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

func TestBadgerStorage_ListWorkflows_WithFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save workflows with different statuses
	workflows := []*storage.WorkflowState{
		{ID: "wf-1", Name: "workflow-1", Status: "pending", CreatedAt: time.Now()},
		{ID: "wf-2", Name: "workflow-2", Status: "running", CreatedAt: time.Now()},
		{ID: "wf-3", Name: "workflow-3", Status: "pending", CreatedAt: time.Now()},
	}

	for _, wf := range workflows {
		if err := db.SaveWorkflow(ctx, wf); err != nil {
			t.Fatalf("SaveWorkflow failed: %v", err)
		}
	}

	// Filter by status
	result, total, err := db.ListWorkflows(ctx, &storage.WorkflowFilter{
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

func TestBadgerStorage_ListWorkflows_WithPagination(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save 5 workflows
	for i := 1; i <= 5; i++ {
		wf := &storage.WorkflowState{
			ID:        string(rune('0' + i)),
			Name:      "workflow",
			Status:    "pending",
			CreatedAt: time.Now(),
		}
		if err := db.SaveWorkflow(ctx, wf); err != nil {
			t.Fatalf("SaveWorkflow failed: %v", err)
		}
	}

	// Get page 1 (limit 2)
	result, total, err := db.ListWorkflows(ctx, &storage.WorkflowFilter{
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
}

func TestBadgerStorage_DeleteWorkflow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	wf := &storage.WorkflowState{
		ID:        "wf-1",
		Name:      "test-workflow",
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	// Save workflow
	if err := db.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Delete workflow
	if err := db.DeleteWorkflow(ctx, "wf-1"); err != nil {
		t.Fatalf("DeleteWorkflow failed: %v", err)
	}

	// Verify deleted
	_, err := db.GetWorkflow(ctx, "wf-1")
	if err == nil {
		t.Fatal("Expected error for deleted workflow")
	}
}

func TestBadgerStorage_SaveAndGetTask(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create workflow first
	wf := &storage.WorkflowState{
		ID:        "wf-1",
		Name:      "test-workflow",
		Status:    "running",
		CreatedAt: time.Now(),
	}
	if err := db.SaveWorkflow(ctx, wf); err != nil {
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

	if err := db.SaveTask(ctx, "wf-1", task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// Get task
	retrieved, err := db.GetTask(ctx, "wf-1", "task-1")
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

func TestBadgerStorage_SaveTask_WorkflowNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	task := &storage.TaskState{
		ID:     "task-1",
		Name:   "Test Task",
		Status: "pending",
	}

	err := db.SaveTask(ctx, "nonexistent", task)
	if err == nil {
		t.Fatal("Expected error for nonexistent workflow")
	}

	if _, ok := err.(*storage.NotFoundError); !ok {
		t.Errorf("Expected NotFoundError, got %T", err)
	}
}

func TestBadgerStorage_ListTasks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create workflow
	wf := &storage.WorkflowState{
		ID:        "wf-1",
		Name:      "test-workflow",
		Status:    "running",
		CreatedAt: time.Now(),
	}
	if err := db.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Save multiple tasks
	tasks := []*storage.TaskState{
		{ID: "task-1", Name: "Task 1", Status: "completed"},
		{ID: "task-2", Name: "Task 2", Status: "running"},
		{ID: "task-3", Name: "Task 3", Status: "pending"},
	}

	for _, task := range tasks {
		if err := db.SaveTask(ctx, "wf-1", task); err != nil {
			t.Fatalf("SaveTask failed: %v", err)
		}
	}

	// List tasks
	result, err := db.ListTasks(ctx, "wf-1")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(result))
	}
}

func TestBadgerStorage_DeleteWorkflow_CascadesTasks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create workflow with tasks
	wf := &storage.WorkflowState{
		ID:        "wf-1",
		Name:      "test-workflow",
		Status:    "running",
		CreatedAt: time.Now(),
	}
	if err := db.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	task := &storage.TaskState{
		ID:     "task-1",
		Name:   "Test Task",
		Status: "running",
	}
	if err := db.SaveTask(ctx, "wf-1", task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// Delete workflow
	if err := db.DeleteWorkflow(ctx, "wf-1"); err != nil {
		t.Fatalf("DeleteWorkflow failed: %v", err)
	}

	// Verify tasks are also deleted
	_, err := db.GetTask(ctx, "wf-1", "task-1")
	if err == nil {
		t.Fatal("Expected error for deleted task")
	}
}

func TestBadgerStorage_UpdateWorkflow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create workflow
	wf := &storage.WorkflowState{
		ID:        "wf-1",
		Name:      "test-workflow",
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	if err := db.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Update workflow status
	wf.Status = "running"
	now := time.Now()
	wf.StartedAt = &now

	if err := db.SaveWorkflow(ctx, wf); err != nil {
		t.Fatalf("SaveWorkflow (update) failed: %v", err)
	}

	// Verify update
	retrieved, err := db.GetWorkflow(ctx, "wf-1")
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}

	if retrieved.Status != "running" {
		t.Errorf("Expected Status running, got %s", retrieved.Status)
	}
	if retrieved.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}
}
