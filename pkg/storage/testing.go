package storage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
)

// StorageTestSuite defines a test suite that can be run against any Storage implementation.
type StorageTestSuite struct {
	NewStorage func(t *testing.T) Storage
}

// RunAllTests runs all storage tests against the provided storage implementation.
func (s *StorageTestSuite) RunAllTests(t *testing.T) {
	t.Run("WorkflowCRUD", s.TestWorkflowCRUD)
	t.Run("TaskPersistence", s.TestTaskPersistence)
	t.Run("ListWorkflowsWithFilter", s.TestListWorkflowsWithFilter)
	t.Run("ListWorkflowsWithPagination", s.TestListWorkflowsWithPagination)
	t.Run("DeleteWorkflowCascade", s.TestDeleteWorkflowCascade)
	t.Run("ConcurrentAccess", s.TestConcurrentAccess)
	t.Run("ErrorHandling", s.TestErrorHandling)
	t.Run("WorkflowNotFound", s.TestWorkflowNotFound)
	t.Run("TaskNotFound", s.TestTaskNotFound)
}

// TestWorkflowCRUD tests basic workflow CRUD operations.
func (s *StorageTestSuite) TestWorkflowCRUD(t *testing.T) {
	store := s.NewStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Create workflow
	wf := &WorkflowState{
		ID:          "wf-1",
		Name:        "Test Workflow",
		Description: "Test Description",
		Status:      "pending",
		Tasks: []models.TaskDefinition{
			{ID: "task-1", Name: "Task 1"},
		},
		TaskStatus: map[string]*TaskState{
			"task-1": {
				ID:     "task-1",
				Name:   "Task 1",
				Status: "pending",
			},
		},
		Metadata:  map[string]string{"key": "value"},
		CreatedAt: time.Now(),
	}

	// Save workflow
	err := store.SaveWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Get workflow
	retrieved, err := store.GetWorkflow(ctx, "wf-1")
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}

	if retrieved.ID != wf.ID {
		t.Errorf("expected ID %s, got %s", wf.ID, retrieved.ID)
	}
	if retrieved.Name != wf.Name {
		t.Errorf("expected Name %s, got %s", wf.Name, retrieved.Name)
	}
	if retrieved.Status != wf.Status {
		t.Errorf("expected Status %s, got %s", wf.Status, retrieved.Status)
	}

	// Update workflow
	retrieved.Status = "running"
	now := time.Now()
	retrieved.StartedAt = &now

	err = store.SaveWorkflow(ctx, retrieved)
	if err != nil {
		t.Fatalf("SaveWorkflow (update) failed: %v", err)
	}

	// Verify update
	updated, err := store.GetWorkflow(ctx, "wf-1")
	if err != nil {
		t.Fatalf("GetWorkflow (after update) failed: %v", err)
	}

	if updated.Status != "running" {
		t.Errorf("expected Status running, got %s", updated.Status)
	}
	if updated.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}

	// Delete workflow
	err = store.DeleteWorkflow(ctx, "wf-1")
	if err != nil {
		t.Fatalf("DeleteWorkflow failed: %v", err)
	}

	// Verify deletion
	_, err = store.GetWorkflow(ctx, "wf-1")
	if err == nil {
		t.Error("expected error when getting deleted workflow")
	}
}

// TestTaskPersistence tests task save and retrieval.
func (s *StorageTestSuite) TestTaskPersistence(t *testing.T) {
	store := s.NewStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Create workflow first
	wf := &WorkflowState{
		ID:         "wf-2",
		Name:       "Task Test Workflow",
		Status:     "pending",
		Tasks:      []models.TaskDefinition{{ID: "task-1", Name: "Task 1"}},
		TaskStatus: map[string]*TaskState{},
		CreatedAt:  time.Now(),
	}

	err := store.SaveWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Save task
	task := &TaskState{
		ID:     "task-1",
		Name:   "Task 1",
		Status: "completed",
		Result: map[string]interface{}{"output": "success"},
	}

	err = store.SaveTask(ctx, "wf-2", task)
	if err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// Get task
	retrieved, err := store.GetTask(ctx, "wf-2", "task-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("expected ID %s, got %s", task.ID, retrieved.ID)
	}
	if retrieved.Status != task.Status {
		t.Errorf("expected Status %s, got %s", task.Status, retrieved.Status)
	}

	// List tasks
	tasks, err := store.ListTasks(ctx, "wf-2")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

// TestListWorkflowsWithFilter tests workflow listing with status filter.
func (s *StorageTestSuite) TestListWorkflowsWithFilter(t *testing.T) {
	store := s.NewStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Create workflows with different statuses
	statuses := []string{"pending", "running", "completed", "failed"}
	for i, status := range statuses {
		wf := &WorkflowState{
			ID:         string(rune('a' + i)),
			Name:       "Workflow " + status,
			Status:     status,
			Tasks:      []models.TaskDefinition{},
			TaskStatus: map[string]*TaskState{},
			CreatedAt:  time.Now(),
		}
		if err := store.SaveWorkflow(ctx, wf); err != nil {
			t.Fatalf("SaveWorkflow failed: %v", err)
		}
	}

	// Filter by status
	filter := &WorkflowFilter{
		Status: []string{"pending", "running"},
		Limit:  10,
		Offset: 0,
	}

	workflows, total, err := store.ListWorkflows(ctx, filter)
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	if len(workflows) != 2 {
		t.Errorf("expected 2 workflows, got %d", len(workflows))
	}

	// Verify filtered results
	for _, wf := range workflows {
		if wf.Status != "pending" && wf.Status != "running" {
			t.Errorf("unexpected status %s in filtered results", wf.Status)
		}
	}
}

// TestListWorkflowsWithPagination tests workflow listing with pagination.
func (s *StorageTestSuite) TestListWorkflowsWithPagination(t *testing.T) {
	store := s.NewStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Create 10 workflows
	for i := 0; i < 10; i++ {
		wf := &WorkflowState{
			ID:         string(rune('a' + i)),
			Name:       "Workflow " + string(rune('a'+i)),
			Status:     "pending",
			Tasks:      []models.TaskDefinition{},
			TaskStatus: map[string]*TaskState{},
			CreatedAt:  time.Now(),
		}
		if err := store.SaveWorkflow(ctx, wf); err != nil {
			t.Fatalf("SaveWorkflow failed: %v", err)
		}
	}

	// First page
	filter := &WorkflowFilter{
		Limit:  3,
		Offset: 0,
	}

	workflows, total, err := store.ListWorkflows(ctx, filter)
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}

	if total != 10 {
		t.Errorf("expected total 10, got %d", total)
	}

	if len(workflows) != 3 {
		t.Errorf("expected 3 workflows, got %d", len(workflows))
	}

	// Second page
	filter.Offset = 3
	workflows, total, err = store.ListWorkflows(ctx, filter)
	if err != nil {
		t.Fatalf("ListWorkflows (page 2) failed: %v", err)
	}

	if total != 10 {
		t.Errorf("expected total 10, got %d", total)
	}

	if len(workflows) != 3 {
		t.Errorf("expected 3 workflows, got %d", len(workflows))
	}
}

// TestDeleteWorkflowCascade tests that deleting a workflow also deletes its tasks.
func (s *StorageTestSuite) TestDeleteWorkflowCascade(t *testing.T) {
	store := s.NewStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Create workflow with tasks
	wf := &WorkflowState{
		ID:     "wf-cascade",
		Name:   "Cascade Test",
		Status: "pending",
		Tasks: []models.TaskDefinition{
			{ID: "task-1", Name: "Task 1"},
			{ID: "task-2", Name: "Task 2"},
		},
		TaskStatus: map[string]*TaskState{},
		CreatedAt:  time.Now(),
	}

	err := store.SaveWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Save tasks
	for _, taskDef := range wf.Tasks {
		task := &TaskState{
			ID:     taskDef.ID,
			Name:   taskDef.Name,
			Status: "pending",
		}
		if err := store.SaveTask(ctx, "wf-cascade", task); err != nil {
			t.Fatalf("SaveTask failed: %v", err)
		}
	}

	// Delete workflow
	err = store.DeleteWorkflow(ctx, "wf-cascade")
	if err != nil {
		t.Fatalf("DeleteWorkflow failed: %v", err)
	}

	// Verify tasks are also deleted
	_, err = store.GetTask(ctx, "wf-cascade", "task-1")
	if err == nil {
		t.Error("expected error when getting task from deleted workflow")
	}
}

// TestConcurrentAccess tests concurrent read/write operations.
func (s *StorageTestSuite) TestConcurrentAccess(t *testing.T) {
	store := s.NewStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Create initial workflow
	wf := &WorkflowState{
		ID:         "wf-concurrent",
		Name:       "Concurrent Test",
		Status:     "pending",
		Tasks:      []models.TaskDefinition{},
		TaskStatus: map[string]*TaskState{},
		CreatedAt:  time.Now(),
	}

	err := store.SaveWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Concurrent updates
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Read
			retrieved, err := store.GetWorkflow(ctx, "wf-concurrent")
			if err != nil {
				errors <- err
				return
			}

			// Modify
			retrieved.Metadata = map[string]string{"iteration": string(rune('0' + idx))}

			// Write
			if err := store.SaveWorkflow(ctx, retrieved); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}

	// Verify workflow still exists
	_, err = store.GetWorkflow(ctx, "wf-concurrent")
	if err != nil {
		t.Errorf("GetWorkflow after concurrent updates failed: %v", err)
	}
}

// TestErrorHandling tests error conditions.
func (s *StorageTestSuite) TestErrorHandling(t *testing.T) {
	store := s.NewStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Test getting non-existent workflow
	_, err := store.GetWorkflow(ctx, "non-existent")
	if err == nil {
		t.Error("expected error when getting non-existent workflow")
	}

	// Test deleting non-existent workflow
	err = store.DeleteWorkflow(ctx, "non-existent")
	if err == nil {
		t.Error("expected error when deleting non-existent workflow")
	}

	// Test getting task from non-existent workflow
	_, err = store.GetTask(ctx, "non-existent", "task-1")
	if err == nil {
		t.Error("expected error when getting task from non-existent workflow")
	}
}

// TestWorkflowNotFound tests NotFoundError for workflows.
func (s *StorageTestSuite) TestWorkflowNotFound(t *testing.T) {
	store := s.NewStorage(t)
	defer store.Close()

	ctx := context.Background()

	_, err := store.GetWorkflow(ctx, "missing-workflow")
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}

	// Check if it's a NotFoundError (implementation-specific)
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

// TestTaskNotFound tests NotFoundError for tasks.
func (s *StorageTestSuite) TestTaskNotFound(t *testing.T) {
	store := s.NewStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Create workflow first
	wf := &WorkflowState{
		ID:         "wf-task-test",
		Name:       "Task Not Found Test",
		Status:     "pending",
		Tasks:      []models.TaskDefinition{},
		TaskStatus: map[string]*TaskState{},
		CreatedAt:  time.Now(),
	}

	err := store.SaveWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Try to get non-existent task
	_, err = store.GetTask(ctx, "wf-task-test", "missing-task")
	if err == nil {
		t.Fatal("expected error for missing task")
	}
}
