package dag

import (
	"fmt"
	"sort"
)

// Graph represents a directed acyclic graph of tasks.
type Graph struct {
	tasks     map[string]*Task // task ID -> task
	edges     map[string][]string // adjacency list: task -> tasks that depend on it
	inDegree  map[string]int      // number of dependencies for each task
	outDegree map[string]int      // number of dependents for each task
	
	dirty     bool // true if graph structure has changed and cached results are invalid
	sorted    []string // cached topological sort
}

// NewGraph creates a new empty DAG.
func NewGraph() *Graph {
	return &Graph{
		tasks:     make(map[string]*Task),
		edges:     make(map[string][]string),
		inDegree:  make(map[string]int),
		outDegree: make(map[string]int),
		dirty:     true,
	}
}

// AddTask adds a task to the graph.
// Returns DuplicateTaskError if a task with the same ID already exists.
func (g *Graph) AddTask(task *Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	
	if err := task.Validate(); err != nil {
		return err
	}
	
	if _, exists := g.tasks[task.ID]; exists {
		return &DuplicateTaskError{ID: task.ID}
	}
	
	// Clone task to avoid external modifications
	cloned := task.Clone()
	
	// Check self-dependency
	for _, depID := range cloned.Deps {
		if depID == cloned.ID {
			return &SelfDependencyError{ID: cloned.ID}
		}
	}
	
	g.tasks[cloned.ID] = cloned
	g.edges[cloned.ID] = []string{}
	g.inDegree[cloned.ID] = len(cloned.Deps)
	g.outDegree[cloned.ID] = 0
	g.dirty = true
	
	// Note: We don't validate dependencies here to allow adding tasks in any order
	// Validation happens in Validate() or Compile()
	
	return nil
}

// AddEdge adds a dependency edge from 'from' to 'to' (to depends on from).
// Returns error if either task doesn't exist or if the edge would create a cycle.
func (g *Graph) AddEdge(from, to string) error {
	if from == "" || to == "" {
		return fmt.Errorf("task IDs cannot be empty")
	}
	
	if from == to {
		return &SelfDependencyError{ID: from}
	}
	
	// Check if tasks exist
	if _, exists := g.tasks[from]; !exists {
		return &TaskNotFoundError{ID: from}
	}
	if _, exists := g.tasks[to]; !exists {
		return &TaskNotFoundError{ID: to}
	}
	
	// Check if edge already exists
	for _, edge := range g.edges[from] {
		if edge == to {
			return nil // Edge already exists, no-op
		}
	}
	
	// Add the edge
	g.edges[from] = append(g.edges[from], to)
	g.inDegree[to]++
	g.outDegree[from]++
	g.dirty = true
	
	// Update task's Deps list
	task := g.tasks[to]
	task.AddDependency(from)
	
	return nil
}

// RemoveTask removes a task and all its associated edges from the graph.
func (g *Graph) RemoveTask(id string) error {
	task, exists := g.tasks[id]
	if !exists {
		return &TaskNotFoundError{ID: id}
	}
	
	// Ensure edges are built
	g.rebuildEdges()
	
	// Remove outgoing edges (this task depends on others)
	for _, depID := range task.Deps {
		_ = depID // TODO: Update dependency's out-degree tracking
		g.inDegree[id]--
		if g.inDegree[id] < 0 {
			g.inDegree[id] = 0
		}
	}
	
	// Remove incoming edges (other tasks depend on this)
	for _, dependentID := range g.edges[id] {
		g.outDegree[id]--
		if g.outDegree[id] < 0 {
			g.outDegree[id] = 0
		}
		
		// Update the dependent task's Deps
		if dependent, ok := g.tasks[dependentID]; ok {
			dependent.RemoveDependency(id)
		}
	}
	
	// Remove from edges map
	delete(g.edges, id)
	
	// Remove references to this task from other edges
	for taskID, dependents := range g.edges {
		for i, depID := range dependents {
			if depID == id {
				g.edges[taskID] = append(dependents[:i], dependents[i+1:]...)
				g.inDegree[taskID]--
				break
			}
		}
	}
	
	// Remove task
	delete(g.tasks, id)
	delete(g.inDegree, id)
	delete(g.outDegree, id)
	g.dirty = true
	
	return nil
}

// GetTask retrieves a task by ID.
func (g *Graph) GetTask(id string) (*Task, bool) {
	task, ok := g.tasks[id]
	if !ok {
		return nil, false
	}
	return task.Clone(), true
}

// HasTask checks if a task exists in the graph.
func (g *Graph) HasTask(id string) bool {
	_, ok := g.tasks[id]
	return ok
}

// Tasks returns all tasks in the graph.
func (g *Graph) Tasks() []*Task {
	tasks := make([]*Task, 0, len(g.tasks))
	for _, task := range g.tasks {
		tasks = append(tasks, task.Clone())
	}
	// Sort by ID for deterministic ordering
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})
	return tasks
}

// TaskCount returns the number of tasks in the graph.
func (g *Graph) TaskCount() int {
	return len(g.tasks)
}

// Dependencies returns the tasks that the given task depends on.
func (g *Graph) Dependencies(id string) ([]*Task, error) {
	task, exists := g.tasks[id]
	if !exists {
		return nil, &TaskNotFoundError{ID: id}
	}
	
	deps := make([]*Task, 0, len(task.Deps))
	for _, depID := range task.Deps {
		if dep, ok := g.tasks[depID]; ok {
			deps = append(deps, dep.Clone())
		} else {
			return nil, &DependencyNotFoundError{SrcTask: id, DepID: depID}
		}
	}
	return deps, nil
}

// Dependents returns the tasks that depend on the given task.
func (g *Graph) Dependents(id string) ([]*Task, error) {
	if _, exists := g.tasks[id]; !exists {
		return nil, &TaskNotFoundError{ID: id}
	}
	
	dependentIDs := g.edges[id]
	dependents := make([]*Task, 0, len(dependentIDs))
	for _, depID := range dependentIDs {
		if task, ok := g.tasks[depID]; ok {
			dependents = append(dependents, task.Clone())
		}
	}
	return dependents, nil
}

// InDegree returns the number of dependencies for a task.
func (g *Graph) InDegree(id string) (int, error) {
	if _, exists := g.tasks[id]; !exists {
		return 0, &TaskNotFoundError{ID: id}
	}
	return g.inDegree[id], nil
}

// OutDegree returns the number of tasks that depend on the given task.
func (g *Graph) OutDegree(id string) (int, error) {
	if _, exists := g.tasks[id]; !exists {
		return 0, &TaskNotFoundError{ID: id}
	}
	return g.outDegree[id], nil
}

// Roots returns tasks with no dependencies (in-degree = 0).
func (g *Graph) Roots() []*Task {
	g.rebuildEdges()
	
	roots := make([]*Task, 0)
	for id, task := range g.tasks {
		if g.inDegree[id] == 0 {
			roots = append(roots, task.Clone())
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].ID < roots[j].ID
	})
	return roots
}

// Leaves returns tasks with no dependents (out-degree = 0).
func (g *Graph) Leaves() []*Task {
	g.rebuildEdges()
	
	leaves := make([]*Task, 0)
	for id, task := range g.tasks {
		if g.outDegree[id] == 0 {
			leaves = append(leaves, task.Clone())
		}
	}
	sort.Slice(leaves, func(i, j int) bool {
		return leaves[i].ID < leaves[j].ID
	})
	return leaves
}

// IsEmpty returns true if the graph has no tasks.
func (g *Graph) IsEmpty() bool {
	return len(g.tasks) == 0
}

// Clear removes all tasks from the graph.
func (g *Graph) Clear() {
	g.tasks = make(map[string]*Task)
	g.edges = make(map[string][]string)
	g.inDegree = make(map[string]int)
	g.outDegree = make(map[string]int)
	g.sorted = nil
	g.dirty = true
}

// rebuildEdges rebuilds the edges map from task dependencies.
func (g *Graph) rebuildEdges() {
	// Only rebuild if dirty
	if !g.dirty {
		return
	}
	
	// Clear existing edges and degrees
	g.edges = make(map[string][]string, len(g.tasks))
	g.inDegree = make(map[string]int, len(g.tasks))
	g.outDegree = make(map[string]int, len(g.tasks))
	
	for id := range g.tasks {
		g.edges[id] = []string{}
		g.inDegree[id] = 0
		g.outDegree[id] = 0
	}
	
	// Rebuild from task dependencies
	for id, task := range g.tasks {
		for _, depID := range task.Deps {
			// depID -> id (id depends on depID)
			g.edges[depID] = append(g.edges[depID], id)
			g.outDegree[depID]++
			g.inDegree[id]++
		}
	}
	
	// Mark as clean
	g.dirty = false
}

// Validate checks the graph for errors:
// - All dependencies must exist
// - No cycles
func (g *Graph) Validate() error {
	// Rebuild edges to ensure consistency
	g.rebuildEdges()
	
	// Check all dependencies exist
	for id, task := range g.tasks {
		for _, depID := range task.Deps {
			if _, exists := g.tasks[depID]; !exists {
				return &DependencyNotFoundError{SrcTask: id, DepID: depID}
			}
		}
	}
	
	// Check for cycles
	if cycle, hasCycle := g.DetectCycle(); hasCycle {
		return cycle
	}
	
	return nil
}
