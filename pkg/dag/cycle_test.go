package dag

import (
	"testing"
)

func TestGraph_DetectCycle_NoCycle(t *testing.T) {
	g := NewGraph()

	// Linear chain: a -> b -> c
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"b"}})

	cycle, hasCycle := g.DetectCycle()
	if hasCycle {
		t.Errorf("expected no cycle, got: %v", cycle)
	}
}

func TestGraph_DetectCycle_SelfLoop(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})

	// Try to add self-loop via AddEdge (should return error)
	err := g.AddEdge("a", "a")
	if err == nil {
		t.Fatal("expected error for self-loop via AddEdge")
	}

	// Create self-loop by manually modifying task (simulating load from invalid config)
	g.tasks["a"].Deps = []string{"a"}
	g.dirty = true

	cycle, hasCycle := g.DetectCycle()
	if !hasCycle {
		t.Fatal("expected cycle")
	}
	if len(cycle.Path) < 1 {
		t.Error("expected cycle path with at least 1 node")
	}
}

func TestGraph_DetectCycle_TwoNodeCycle(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.tasks["a"].Deps = append(g.tasks["a"].Deps, "b") // Creates cycle: a -> b -> a
	g.dirty = true

	cycle, hasCycle := g.DetectCycle()
	if !hasCycle {
		t.Fatal("expected cycle")
	}
	if len(cycle.Path) < 2 {
		t.Errorf("expected cycle path, got: %v", cycle.Path)
	}
}

func TestGraph_DetectCycle_ThreeNodeCycle(t *testing.T) {
	g := NewGraph()

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"b"}})
	g.tasks["a"].Deps = append(g.tasks["a"].Deps, "c") // Creates cycle: a -> b -> c -> a
	g.dirty = true

	cycle, hasCycle := g.DetectCycle()
	if !hasCycle {
		t.Fatal("expected cycle")
	}
	if len(cycle.Path) < 3 {
		t.Errorf("expected cycle path with at least 3 nodes, got: %v", cycle.Path)
	}
}

func TestGraph_DetectCycle_Complex(t *testing.T) {
	g := NewGraph()

	// Create a more complex graph with a cycle
	// a -> b -> c -> d
	//      ^         |
	//      +---------+
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.AddTask(&Task{ID: "c", Name: "C", Agent: "test", Deps: []string{"b"}})
	g.AddTask(&Task{ID: "d", Name: "D", Agent: "test", Deps: []string{"c"}})
	g.tasks["b"].Deps = append(g.tasks["b"].Deps, "d") // Creates cycle: b -> c -> d -> b
	g.dirty = true

	cycle, hasCycle := g.DetectCycle()
	if !hasCycle {
		t.Fatal("expected cycle")
	}
	if len(cycle.Path) < 3 {
		t.Errorf("expected cycle path with at least 3 nodes, got: %v", cycle.Path)
	}

	// Check error message contains path
	errMsg := cycle.Error()
	if errMsg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestGraph_DetectCycle_EmptyGraph(t *testing.T) {
	g := NewGraph()

	cycle, hasCycle := g.DetectCycle()
	if hasCycle {
		t.Errorf("expected no cycle in empty graph, got: %v", cycle)
	}
}

func TestGraph_HasCycle(t *testing.T) {
	g := NewGraph()

	if g.HasCycle() {
		t.Error("expected no cycle in empty graph")
	}

	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})

	if g.HasCycle() {
		t.Error("expected no cycle in linear graph")
	}

	g.tasks["a"].Deps = append(g.tasks["a"].Deps, "b")
	g.dirty = true

	if !g.HasCycle() {
		t.Error("expected cycle")
	}
}

func TestGraph_FindAllCycles(t *testing.T) {
	g := NewGraph()

	// a -> b -> a (one cycle)
	g.AddTask(&Task{ID: "a", Name: "A", Agent: "test"})
	g.AddTask(&Task{ID: "b", Name: "B", Agent: "test", Deps: []string{"a"}})
	g.tasks["a"].Deps = append(g.tasks["a"].Deps, "b")
	g.dirty = true

	cycles := g.FindAllCycles()
	if len(cycles) == 0 {
		t.Error("expected to find cycles")
	}
}

func TestCyclicDependencyError(t *testing.T) {
	err := &CyclicDependencyError{
		Path: []string{"a", "b", "c", "a"},
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("expected non-empty error message")
	}

	if err.TaskID() != "a" {
		t.Errorf("expected TaskID 'a', got %s", err.TaskID())
	}
}

func TestCyclicDependencyError_EmptyPath(t *testing.T) {
	err := &CyclicDependencyError{Path: []string{}}

	if err.Error() == "" {
		t.Error("expected non-empty error message even with empty path")
	}
}
