// Package memory provides an in-memory implementation of the storage interface.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
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

	copied := cloneWorkflowState(wf)

	// Check for duplicate on create (if workflow doesn't exist yet)
	if _, exists := m.workflows[copied.ID]; !exists && copied.CreatedAt.IsZero() {
		copied.CreatedAt = time.Now().UTC()
	}

	m.workflows[copied.ID] = copied
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

	return cloneWorkflowState(wf), nil
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
		result[i] = cloneWorkflowState(wf)
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
	copied := cloneTaskState(task)
	m.tasks[workflowID][task.ID] = copied

	// Also update in workflow's TaskStatus
	if m.workflows[workflowID].TaskStatus == nil {
		m.workflows[workflowID].TaskStatus = make(map[string]*storage.TaskState)
	}
	m.workflows[workflowID].TaskStatus[task.ID] = cloneTaskState(copied)

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

	return cloneTaskState(task), nil
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
		result = append(result, cloneTaskState(task))
	}

	return result, nil
}

// Close closes the storage (no-op for memory storage).
func (m *MemoryStorage) Close() error {
	return nil
}

func cloneWorkflowState(wf *storage.WorkflowState) *storage.WorkflowState {
	if wf == nil {
		return nil
	}

	copied := *wf
	copied.Tasks = cloneTaskDefinitions(wf.Tasks)
	copied.TaskStatus = cloneTaskStatusMap(wf.TaskStatus)
	copied.Metadata = cloneStringMap(wf.Metadata)
	copied.StartedAt = cloneTimePtr(wf.StartedAt)
	copied.CompletedAt = cloneTimePtr(wf.CompletedAt)
	return &copied
}

func cloneTaskState(task *storage.TaskState) *storage.TaskState {
	if task == nil {
		return nil
	}
	copied := *task
	copied.StartedAt = cloneTimePtr(task.StartedAt)
	copied.CompletedAt = cloneTimePtr(task.CompletedAt)
	copied.Result = deepCopyValue(task.Result)
	return &copied
}

func cloneTaskStatusMap(src map[string]*storage.TaskState) map[string]*storage.TaskState {
	if src == nil {
		return nil
	}
	out := make(map[string]*storage.TaskState, len(src))
	for taskID, taskState := range src {
		out[taskID] = cloneTaskState(taskState)
	}
	return out
}

func cloneTaskDefinitions(src []models.TaskDefinition) []models.TaskDefinition {
	if len(src) == 0 {
		return nil
	}
	out := make([]models.TaskDefinition, len(src))
	for i := range src {
		out[i] = src[i]
		out[i].Dependencies = append([]string(nil), src[i].Dependencies...)
		out[i].DependsOn = append([]string(nil), src[i].DependsOn...)
		out[i].Config = cloneAnyMap(src[i].Config)
	}
	return out
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func cloneAnyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = deepCopyValue(v)
	}
	return out
}

func cloneTimePtr(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	t := *src
	return &t
}

func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, inner := range val {
			out[k] = deepCopyValue(inner)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, inner := range val {
			out[i] = deepCopyValue(inner)
		}
		return out
	default:
		return val
	}
}
