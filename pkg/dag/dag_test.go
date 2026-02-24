package dag

import (
	"testing"
	"time"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph()
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if !g.IsEmpty() {
		t.Error("expected empty graph")
	}
}

func TestGraph_AddTask(t *testing.T) {
	g := NewGraph()

	// Valid task
	task := &Task{
		ID:    "task1",
		Name:  "Task 1",
		Agent: "test",
	}
	if err := g.AddTask(task); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Duplicate task
	if err := g.AddTask(task); err == nil {
		t.Error("expected error for duplicate task")
	} else if _, ok := err.(*DuplicateTaskError); !ok {
		t.Errorf("expected DuplicateTaskError, got %T", err)
	}

	// Invalid task (no ID)
	invalidTask := &Task{Name: "Invalid"}
	if err := g.AddTask(invalidTask); err == nil {
		t.Error("expected error for invalid task")
	}

	// Self-dependency via Deps
	selfDepTask := &Task{
		ID:    "self",
		Name:  "Self",
		Agent: "test",
		Deps:  []string{"self"},
	}
	if err := g.AddTask(selfDepTask); err == nil {
		t.Error("expected error for self-dependency")
	}
}

func TestGraph_AddEdge(t *testing.T) {
	g := NewGraph()

	// Add tasks
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test"})

	// Valid edge
	if err := g.AddEdge("a", "b"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Edge to non-existent task
	if err := g.AddEdge("a", "c"); err == nil {
		t.Error("expected error for non-existent target")
	}

	// Edge from non-existent task
	if err := g.AddEdge("c", "b"); err == nil {
		t.Error("expected error for non-existent source")
	}

	// Self edge
	if err := g.AddEdge("a", "a"); err == nil {
		t.Error("expected error for self-edge")
	}
}

func TestGraph_GetTask(t *testing.T) {
	g := NewGraph()
	task := &Task{ID: "task1", Name: "Task 1", Agent: "test"}
	g.AddTask(task)

	// Get existing task
	got, ok := g.GetTask("task1")
	if !ok {
		t.Error("expected to find task")
	}
	if got.ID != "task1" {
		t.Errorf("expected task1, got %s", got.ID)
	}

	// Get non-existent task
	_, ok = g.GetTask("nonexistent")
	if ok {
		t.Error("expected not to find non-existent task")
	}
}

func TestGraph_Tasks(t *testing.T) {
	g := NewGraph()

	if len(g.Tasks()) != 0 {
		t.Error("expected no tasks")
	}

	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test"})
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})

	tasks := g.Tasks()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}

	// Check sorted order
	if tasks[0].ID != "a" || tasks[1].ID != "b" {
		t.Error("expected tasks to be sorted by ID")
	}
}

func TestGraph_RootsAndLeaves(t *testing.T) {
	g := NewGraph()

	// Linear chain: a -> b -> c
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"b"}})

	roots := g.Roots()
	if len(roots) != 1 || roots[0].ID != "a" {
		t.Errorf("expected root [a], got %v", roots)
	}

	leaves := g.Leaves()
	if len(leaves) != 1 || leaves[0].ID != "c" {
		t.Errorf("expected leaf [c], got %v", leaves)
	}
}

func TestGraph_Dependencies(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"a", "b"}})

	// Task with dependencies
	deps, err := g.Dependencies("c")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}

	// Task without dependencies
	deps, err = g.Dependencies("a")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(deps))
	}

	// Non-existent task
	_, err = g.Dependencies("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}

func TestGraph_RemoveTask(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})

	if err := g.RemoveTask("a"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if g.HasTask("a") {
		t.Error("expected task 'a' to be removed")
	}

	// Check that b's dependency was updated
	b, _ := g.GetTask("b")
	if len(b.Deps) != 0 {
		t.Error("expected b's dependencies to be updated")
	}

	// Remove non-existent
	if err := g.RemoveTask("nonexistent"); err == nil {
		t.Error("expected error for non-existent task")
	}
}

func TestGraph_Clear(t *testing.T) {
	g := NewGraph()
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test"})

	g.Clear()

	if !g.IsEmpty() {
		t.Error("expected empty graph after clear")
	}
}

func TestTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    *Task
		wantErr bool
	}{
		{
			name:    "valid task",
			task:    &Task{ID: "t1", Name: "Task", Agent: "test"},
			wantErr: false,
		},
		{
			name:    "missing ID",
			task:    &Task{Name: "Task", Agent: "test"},
			wantErr: true,
		},
		{
			name:    "missing Name",
			task:    &Task{ID: "t1", Agent: "test"},
			wantErr: true,
		},
		{
			name:    "missing Agent",
			task:    &Task{ID: "t1", Name: "Task"},
			wantErr: true,
		},
		{
			name:    "negative timeout",
			task:    &Task{ID: "t1", Name: "Task", Agent: "test", Timeout: -1 * time.Second},
			wantErr: true,
		},
		{
			name:    "negative retries",
			task:    &Task{ID: "t1", Name: "Task", Agent: "test", Retries: -1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTask_Clone(t *testing.T) {
	original := &Task{
		ID:       "t1",
		Name:     "Task",
		Agent:    "test",
		Deps:     []string{"a", "b"},
		Metadata: map[string]string{"key": "value"},
	}

	cloned := original.Clone()

	// Modify original
	original.Deps[0] = "modified"
	original.Metadata["key"] = "modified"

	// Check clone is not affected
	if cloned.Deps[0] == "modified" {
		t.Error("clone Deps should not be affected by original modification")
	}
	if cloned.Metadata["key"] == "modified" {
		t.Error("clone Metadata should not be affected by original modification")
	}
}

func TestTask_HasDependency(t *testing.T) {
	task := &Task{ID: "t1", Name: "Task", Agent: "test", Deps: []string{"a", "b"}}

	if !task.HasDependency("a") {
		t.Error("expected to have dependency 'a'")
	}
	if task.HasDependency("c") {
		t.Error("expected not to have dependency 'c'")
	}
}

func TestTask_AddDependency(t *testing.T) {
	task := &Task{ID: "t1", Name: "Task", Agent: "test"}

	if err := task.AddDependency("a"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Add duplicate
	if err := task.AddDependency("a"); err == nil {
		t.Error("expected error for duplicate dependency")
	}

	// Self dependency
	if err := task.AddDependency("t1"); err == nil {
		t.Error("expected error for self dependency")
	}
}
