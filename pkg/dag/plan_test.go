package dag

import (
	"testing"
)

func TestGraph_Compile_Linear(t *testing.T) {
	g := NewGraph()

	// Linear: a -> b -> c
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"b"}})

	plan, err := g.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.TotalTasks != 3 {
		t.Errorf("expected 3 tasks, got %d", plan.TotalTasks)
	}

	if plan.TotalLayers != 3 {
		t.Errorf("expected 3 layers, got %d", plan.TotalLayers)
	}

	if plan.MaxParallel != 1 {
		t.Errorf("expected max parallel = 1, got %d", plan.MaxParallel)
	}

	// Critical path should be a -> b -> c
	if len(plan.CriticalPath) != 3 {
		t.Errorf("expected critical path length 3, got %d", len(plan.CriticalPath))
	}
}

func TestGraph_Compile_Parallel(t *testing.T) {
	g := NewGraph()

	// Fork-join: a -> (b, c) -> d
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "d", Name: "D", Agent: "test", Deps: []string{"b", "c"}})

	plan, err := g.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.TotalTasks != 4 {
		t.Errorf("expected 4 tasks, got %d", plan.TotalTasks)
	}

	if plan.MaxParallel != 2 {
		t.Errorf("expected max parallel = 2, got %d", plan.MaxParallel)
	}

	// Layer 1 should have b and c
	if len(plan.Layers[1]) != 2 {
		t.Errorf("expected layer 1 to have 2 tasks, got %d", len(plan.Layers[1]))
	}
}

func TestGraph_Compile_WithCycle(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.tasks["a"].Deps = append(g.tasks["a"].Deps, "b")
	g.dirty = true

	_, err := g.Compile()
	if err == nil {
		t.Fatal("expected error for cyclic graph")
	}
}

func TestGraph_Compile_Empty(t *testing.T) {
	g := NewGraph()

	plan, err := g.Compile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.TotalTasks != 0 {
		t.Errorf("expected 0 tasks, got %d", plan.TotalTasks)
	}
}

func TestExecutionPlan_GetTask(t *testing.T) {
	g := NewGraph()
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})

	plan, _ := g.Compile()

	task, ok := plan.GetTask("a")
	if !ok {
		t.Error("expected to find task 'a'")
	}
	if task.ID != "a" {
		t.Errorf("expected task ID 'a', got %s", task.ID)
	}

	_, ok = plan.GetTask("nonexistent")
	if ok {
		t.Error("expected not to find non-existent task")
	}
}

func TestExecutionPlan_GetLayer(t *testing.T) {
	g := NewGraph()
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})

	plan, _ := g.Compile()

	if plan.GetLayer("a") != 0 {
		t.Error("expected 'a' to be in layer 0")
	}

	if plan.GetLayer("b") != 1 {
		t.Error("expected 'b' to be in layer 1")
	}

	if plan.GetLayer("nonexistent") != -1 {
		t.Error("expected -1 for non-existent task")
	}
}

func TestExecutionPlan_CanRunInParallel(t *testing.T) {
	g := NewGraph()
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"a"}})

	plan, _ := g.Compile()

	// b and c can run in parallel
	if !plan.CanRunInParallel("b", "c") {
		t.Error("expected 'b' and 'c' to be parallelizable")
	}

	// a and b cannot (a must complete before b)
	if plan.CanRunInParallel("a", "b") {
		t.Error("expected 'a' and 'b' not to be parallelizable")
	}
}

func TestExecutionPlan_DependenciesOf(t *testing.T) {
	g := NewGraph()
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"a", "b"}})

	plan, _ := g.Compile()

	deps := plan.DependenciesOf("c")
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}

	deps = plan.DependenciesOf("a")
	if len(deps) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(deps))
	}

	deps = plan.DependenciesOf("nonexistent")
	if deps != nil {
		t.Error("expected nil for non-existent task")
	}
}

func TestExecutionPlan_DependentsOf(t *testing.T) {
	g := NewGraph()
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"a"}})

	plan, _ := g.Compile()

	dependents := plan.DependentsOf("a")
	if len(dependents) != 2 {
		t.Errorf("expected 2 dependents, got %d", len(dependents))
	}
}

func TestExecutionPlan_IsReady(t *testing.T) {
	g := NewGraph()
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})

	plan, _ := g.Compile()

	// 'a' is ready (no dependencies)
	if !plan.IsReady("a", map[string]bool{}) {
		t.Error("expected 'a' to be ready")
	}

	// 'b' is not ready (a not completed)
	if plan.IsReady("b", map[string]bool{}) {
		t.Error("expected 'b' not to be ready")
	}

	// 'b' is ready after 'a' completes
	if !plan.IsReady("b", map[string]bool{"a": true}) {
		t.Error("expected 'b' to be ready after 'a' completes")
	}
}

func TestExecutionPlan_Validate(t *testing.T) {
	g := NewGraph()
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})

	plan, _ := g.Compile()

	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestGraph_CriticalPath(t *testing.T) {
	g := NewGraph()

	// Longer path: a -> b -> c
	// Shorter path: a -> d
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"b"}})
	g.AddTask(&Task{ID: "d", Name: "D", Agent: "test", Deps: []string{"a"}})

	plan, _ := g.Compile()

	// Critical path should be a -> b -> c (3 tasks)
	if len(plan.CriticalPath) != 3 {
		t.Errorf("expected critical path length 3, got %d: %v", len(plan.CriticalPath), plan.CriticalPath)
	}

	// Check that it starts with 'a' and ends with 'c'
	if plan.CriticalPath[0] != "a" {
		t.Errorf("expected critical path to start with 'a', got %s", plan.CriticalPath[0])
	}
	if plan.CriticalPath[len(plan.CriticalPath)-1] != "c" {
		t.Errorf("expected critical path to end with 'c', got %s", plan.CriticalPath[len(plan.CriticalPath)-1])
	}
}

func TestExecutionPlan_String(t *testing.T) {
	g := NewGraph()
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})

	plan, _ := g.Compile()

	s := plan.String()
	if s == "" {
		t.Error("expected non-empty string representation")
	}
}
