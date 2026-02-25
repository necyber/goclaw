// Package memory provides an in-memory implementation of the storage interface.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/goclaw/goclaw/pkg/storage"
)

// MemoryStorage implements the Storage interface using in-memory maps.
type MemoryStorage struct {
	mu        sync.RWMutex
	workflows map[string]*storage.WorkflowState
	tasks     map[string]map[string]*storage.TaskState // workflowID -> taskID -> TaskState
}

// NewMemoryStorage creates a new in-memory storage instance.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		workflows: make(map[string]*storage.WorkflowState),
		tasks:     make(map[string]map[string]*storage.TaskState),
	}
}

// SaveWorkflow saves a workflow to memory.
func (m *MemoryStorage) SaveWorkflow(ctx context.Context, wf *storage.WorkflowState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate on create (if workflow doesn't exist yet)
	if _, exists := m.workflows[wf.ID]; !exists && wf.CreatedAt.IsZero() {
		wf.CreatedAt = time.Now()
	}

	// Deep copy to avoid external modifications
	copied := *wf
	if wf.TaskStatus != nil {
		copied.TaskStatus = make(map[string]*storage.TaskState, len(wf.TaskStatus))
		for k, v := range wf.TaskStatus {
			taskCopy := *v
			copied.TaskStatus[k] = &taskCopy
		}
	}

	m.workflows[wf.ID] = &copied
	return nil
}

// GetWorkflow retrieves a workflow by ID.
func (m *MemoryStorage) GetWorkflow(ctx context.Context, id string) (*storage.WorkflowState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wf, exists := m.workflows[id]
	if !exists {
		return nil, &storage.NotFoundError{
			EntityType: "workflow",
			ID:         id,
		}
	}

	// Deep copy to avoid external modifications
	copied := *wf
	if wf.TaskStatus != nil {
		copied.TaskStatus = make(map[string]*storage.TaskState, len(wf.TaskStatus))
		for k, v := range wf.TaskStatus {
			taskCopy := *v
			copied.TaskStatus[k] = &taskCopy
		}
	}

	return &copied, nil
}

// ListWorkflows lists workflows with optional filtering and pagination.
func (m *MemoryStorage) ListWorkflows(ctx context.Context, filter *storage.WorkflowFilter) ([]*storage.WorkflowState, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all workflows
	var all []*storage.WorkflowState
	for _, wf := range m.workflows {
		all = append(all, wf)
	}

	// Filter by status if specified
	var filtered []*storage.WorkflowState
	if filter != nil && len(filter.Status) > 0 {
		statusMap := make(map[string]bool)
		for _, s := range filter.Status {
			statusMap[s] = true
		}
		for _, wf := range all {
			if statusMap[wf.Status] {
				filtered = append(filtered, wf)
			}
		}
	} else {
		filtered = all
	}

	total := len(filtered)

	// Apply pagination
	if filter != nil {
		start := filter.Offset
		end := filter.Offset + filter.Limit

		if start > len(filtered) {
			start = len(filtered)
		}
		if end > len(filtered) {
			end = len(filtered)
		}
		if filter.Limit > 0 {
			filtered = filtered[start:end]
		}
	}

	// Deep copy results
	result := make([]*storage.WorkflowState, len(filtered))
	for i, wf := range filtered {
		copied := *wf
		if wf.TaskStatus != nil {
			copied.TaskStatus = make(map[string]*storage.TaskState, len(wf.TaskStatus))
			for k, v := range wf.TaskStatus {
				taskCopy := *v
				copied.TaskStatus[k] = &taskCopy
			}
		}
		result[i] = &copied
	}

	return result, total, nil
}

// DeleteWorkflow deletes a workflow and all its tasks.
func (m *MemoryStorage) DeleteWorkflow(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.workflows[id]; !exists {
		return &storage.NotFoundError{
			EntityType: "workflow",
			ID:         id,
		}
	}

	delete(m.workflows, id)
	delete(m.tasks, id)
	return nil
}

// SaveTask saves a task state.
func (m *MemoryStorage) SaveTask(ctx context.Context, workflowID string, task *storage.TaskState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify workflow exists
	if _, exists := m.workflows[workflowID]; !exists {
		return &storage.NotFoundError{
			EntityType: "workflow",
			ID:         workflowID,
		}
	}

	// Initialize task map for workflow if needed
	if m.tasks[workflowID] == nil {
		m.tasks[workflowID] = make(map[string]*storage.TaskState)
	}

	// Deep copy task
	copied := *task
	m.tasks[workflowID][task.ID] = &copied

	// Also update in workflow's TaskStatus
	if m.workflows[workflowID].TaskStatus == nil {
		m.workflows[workflowID].TaskStatus = make(map[string]*storage.TaskState)
	}
	m.workflows[workflowID].TaskStatus[task.ID] = &copied

	return nil
}

// GetTask retrieves a task by workflow ID and task ID.
func (m *MemoryStorage) GetTask(ctx context.Context, workflowID, taskID string) (*storage.TaskState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workflowTasks, exists := m.tasks[workflowID]
	if !exists {
		return nil, &storage.NotFoundError{
			EntityType: "workflow",
			ID:         workflowID,
		}
	}

	task, exists := workflowTasks[taskID]
	if !exists {
		return nil, &storage.NotFoundError{
			EntityType: "task",
			ID:         taskID,
		}
	}

	// Deep copy
	copied := *task
	return &copied, nil
}

// ListTasks lists all tasks for a workflow.
func (m *MemoryStorage) ListTasks(ctx context.Context, workflowID string) ([]*storage.TaskState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workflowTasks, exists := m.tasks[workflowID]
	if !exists {
		return nil, &storage.NotFoundError{
			EntityType: "workflow",
			ID:         workflowID,
		}
	}

	result := make([]*storage.TaskState, 0, len(workflowTasks))
	for _, task := range workflowTasks {
		copied := *task
		result = append(result, &copied)
	}

	return result, nil
}

// Close closes the storage (no-op for memory storage).
func (m *MemoryStorage) Close() error {
	return nil
}
