package dag

import (
	"testing"
)

func TestGraph_TopologicalSort_Linear(t *testing.T) {
	g := NewGraph()

	// Linear: a -> b -> c
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"b"}})

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(order))
	}

	// Check order: a should come before b, b before c
	if !g.IsTopologicalOrder(order) {
		t.Error("expected valid topological order")
	}
}

func TestGraph_TopologicalSort_Diamond(t *testing.T) {
	g := NewGraph()

	// Diamond: a -> b, a -> c, b -> d, c -> d
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "d", Name: "D", Agent: "test", Deps: []string{"b", "c"}})

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 4 {
		t.Errorf("expected 4 tasks, got %d", len(order))
	}

	// d must be last, a must be first
	if order[0] != "a" {
		t.Errorf("expected 'a' first, got %s", order[0])
	}
	if order[3] != "d" {
		t.Errorf("expected 'd' last, got %s", order[3])
	}
}

func TestGraph_TopologicalSort_WithCycle(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.tasks["a"].Deps = append(g.tasks["a"].Deps, "b") // Create cycle
	g.dirty = true

	_, err := g.TopologicalSort()
	if err == nil {
		t.Fatal("expected error for cyclic graph")
	}

	if _, ok := err.(*CyclicDependencyError); !ok {
		t.Errorf("expected CyclicDependencyError, got %T", err)
	}
}

func TestGraph_TopologicalSort_Empty(t *testing.T) {
	g := NewGraph()

	order, err := g.TopologicalSort()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(order) != 0 {
		t.Errorf("expected empty order, got %v", order)
	}
}

func TestGraph_TopologicalSort_Caching(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})

	// First call should compute
	order1, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call should use cache
	order2, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Results should be equal
	if len(order1) != len(order2) {
		t.Error("cached result should have same length")
	}
}

func TestGraph_TopologicalSort_CacheInvalidation(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})

	order1, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order1) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(order1))
	}

	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"b"}})

	order2, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error after mutation: %v", err)
	}
	if len(order2) != 3 {
		t.Fatalf("expected 3 tasks after mutation, got %d", len(order2))
	}
	if !g.IsTopologicalOrder(order2) {
		t.Error("expected valid topological order after mutation")
	}
}

func TestGraph_TopologicalSortDFS(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"a"}})

	order, err := g.TopologicalSortDFS()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(order))
	}

	// 'a' should be first (comes before its dependents)
	if order[0] != "a" {
		t.Errorf("expected 'a' first in DFS order, got %s", order[0])
	}
}

func TestGraph_IsTopologicalOrder(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"b"}})

	// Valid order
	if !g.IsTopologicalOrder([]string{"a", "b", "c"}) {
		t.Error("expected valid order")
	}

	// Invalid order (b before a)
	if g.IsTopologicalOrder([]string{"b", "a", "c"}) {
		t.Error("expected invalid order")
	}

	// Wrong length
	if g.IsTopologicalOrder([]string{"a", "b"}) {
		t.Error("expected invalid order (wrong length)")
	}
}

func TestGraph_Levels(t *testing.T) {
	g := NewGraph()

	// Create graph:
	// a -> b -> d
	// a -> c -> d
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "d", Name: "D", Agent: "test", Deps: []string{"b", "c"}})

	levels, err := g.Levels()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(levels) != 3 {
		t.Errorf("expected 3 levels, got %d", len(levels))
	}

	// Level 0 should have 'a'
	if len(levels[0]) != 1 || levels[0][0] != "a" {
		t.Errorf("expected level 0 = [a], got %v", levels[0])
	}

	// Level 1 should have 'b' and 'c'
	if len(levels[1]) != 2 {
		t.Errorf("expected level 1 to have 2 tasks, got %d", len(levels[1]))
	}

	// Level 2 should have 'd'
	if len(levels[2]) != 1 || levels[2][0] != "d" {
		t.Errorf("expected level 2 = [d], got %v", levels[2])
	}
}

func TestGraph_Levels_WithCycle(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.tasks["a"].Deps = append(g.tasks["a"].Deps, "b")
	g.dirty = true

	_, err := g.Levels()
	if err == nil {
		t.Fatal("expected error for cyclic graph")
	}
}

func TestGraph_Levels_Empty(t *testing.T) {
	g := NewGraph()

	levels, err := g.Levels()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(levels) != 0 {
		t.Errorf("expected empty levels, got %v", levels)
	}
}
