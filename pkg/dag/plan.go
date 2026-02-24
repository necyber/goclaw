package dag

import (
	"fmt"
	"sort"
	"strings"
)

// ExecutionPlan represents a compiled DAG ready for execution.
type ExecutionPlan struct {
	// Layers contains task IDs grouped by execution layer.
	// Tasks in the same layer can be executed in parallel.
	Layers [][]string `json:"layers"`

	// ParallelGroups contains groups of tasks that can run concurrently.
	ParallelGroups []TaskGroup `json:"parallel_groups"`

	// CriticalPath is the longest path from any root to any leaf.
	// This determines the minimum execution time.
	CriticalPath []string `json:"critical_path"`

	// TotalTasks is the total number of tasks in the plan.
	TotalTasks int `json:"total_tasks"`

	// MaxParallel is the maximum number of tasks that can run simultaneously.
	MaxParallel int `json:"max_parallel"`

	// TotalLayers is the number of execution layers.
	TotalLayers int `json:"total_layers"`

	// taskMap provides quick lookup of tasks by ID (internal use)
	taskMap map[string]*Task
}

// TaskGroup represents a group of tasks that can be executed in parallel.
type TaskGroup struct {
	// Tasks is the list of task IDs in this group.
	Tasks []string `json:"tasks"`

	// Layer is the execution layer index.
	Layer int `json:"layer"`
}

// Compile compiles the DAG into an ExecutionPlan.
// Returns an error if the graph contains a cycle.
func (g *Graph) Compile() (*ExecutionPlan, error) {
	if len(g.tasks) == 0 {
		return &ExecutionPlan{
			Layers:         [][]string{},
			ParallelGroups: []TaskGroup{},
			CriticalPath:   []string{},
			TotalTasks:     0,
			MaxParallel:    0,
			TotalLayers:    0,
			taskMap:        make(map[string]*Task),
		}, nil
	}

	// Validate and check for cycles
	if err := g.Validate(); err != nil {
		return nil, err
	}

	// Build layers
	layers, err := g.Levels()
	if err != nil {
		return nil, err
	}

	// Build parallel groups
	parallelGroups := make([]TaskGroup, len(layers))
	maxParallel := 0
	for i, layer := range layers {
		parallelGroups[i] = TaskGroup{
			Tasks: layer,
			Layer: i,
		}
		if len(layer) > maxParallel {
			maxParallel = len(layer)
		}
	}

	// Calculate critical path
	criticalPath := g.calculateCriticalPath()

	// Build task map
	taskMap := make(map[string]*Task, len(g.tasks))
	for id, task := range g.tasks {
		taskMap[id] = task.Clone()
	}

	return &ExecutionPlan{
		Layers:         layers,
		ParallelGroups: parallelGroups,
		CriticalPath:   criticalPath,
		TotalTasks:     len(g.tasks),
		MaxParallel:    maxParallel,
		TotalLayers:    len(layers),
		taskMap:        taskMap,
	}, nil
}

// calculateCriticalPath finds the longest path in the DAG using dynamic programming.
// Returns the list of task IDs representing the critical path.
func (g *Graph) calculateCriticalPath() []string {
	if len(g.tasks) == 0 {
		return []string{}
	}

	// Get topological order
	order, err := g.TopologicalSort()
	if err != nil {
		return []string{}
	}

	// dist[v] = length of longest path ending at v
	dist := make(map[string]int, len(g.tasks))
	// prev[v] = previous node in the longest path to v
	prev := make(map[string]string, len(g.tasks))

	// Initialize distances
	for _, id := range order {
		dist[id] = 1 // Each task itself counts as 1
		prev[id] = ""
	}

	// Dynamic programming: process in topological order
	maxDist := 0
	maxNode := ""

	for _, id := range order {
		task := g.tasks[id]
		for _, depID := range task.Deps {
			// Edge: depID -> id (id depends on depID)
			// So we update id based on depID
			if dist[depID]+1 > dist[id] {
				dist[id] = dist[depID] + 1
				prev[id] = depID
			}
		}
		if dist[id] > maxDist {
			maxDist = dist[id]
			maxNode = id
		}
	}

	// Reconstruct path by following prev links
	if maxNode == "" {
		return []string{}
	}

	path := []string{}
	for node := maxNode; node != ""; node = prev[node] {
		path = append([]string{node}, path...)
	}

	return path
}

// GetTask retrieves a task from the plan by ID.
func (p *ExecutionPlan) GetTask(id string) (*Task, bool) {
	task, ok := p.taskMap[id]
	if !ok {
		return nil, false
	}
	return task.Clone(), true
}

// GetLayer returns the layer index for a given task ID.
// Returns -1 if the task is not in the plan.
func (p *ExecutionPlan) GetLayer(taskID string) int {
	if p == nil {
		return -1
	}
	for i, layer := range p.Layers {
		for _, id := range layer {
			if id == taskID {
				return i
			}
		}
	}
	return -1
}

// CanRunInParallel checks if two tasks can run in parallel.
func (p *ExecutionPlan) CanRunInParallel(taskID1, taskID2 string) bool {
	layer1 := p.GetLayer(taskID1)
	layer2 := p.GetLayer(taskID2)
	return layer1 == layer2 && layer1 >= 0
}

// DependenciesOf returns all dependencies of a task within the plan.
func (p *ExecutionPlan) DependenciesOf(taskID string) []string {
	task, ok := p.taskMap[taskID]
	if !ok {
		return nil
	}
	deps := make([]string, len(task.Deps))
	copy(deps, task.Deps)
	return deps
}

// DependentsOf returns all tasks that depend on the given task.
func (p *ExecutionPlan) DependentsOf(taskID string) []string {
	dependents := []string{}
	for id, task := range p.taskMap {
		for _, dep := range task.Deps {
			if dep == taskID {
				dependents = append(dependents, id)
				break
			}
		}
	}
	sort.Strings(dependents)
	return dependents
}

// IsReady checks if all dependencies of a task are completed.
func (p *ExecutionPlan) IsReady(taskID string, completed map[string]bool) bool {
	deps := p.DependenciesOf(taskID)
	for _, dep := range deps {
		if !completed[dep] {
			return false
		}
	}
	return true
}

// String returns a string representation of the execution plan.
func (p *ExecutionPlan) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ExecutionPlan{\n"))
	sb.WriteString(fmt.Sprintf("  Total Tasks: %d\n", p.TotalTasks))
	sb.WriteString(fmt.Sprintf("  Total Layers: %d\n", p.TotalLayers))
	sb.WriteString(fmt.Sprintf("  Max Parallel: %d\n", p.MaxParallel))
	sb.WriteString(fmt.Sprintf("  Critical Path: %v\n", p.CriticalPath))
	sb.WriteString(fmt.Sprintf("  Layers:\n"))
	for i, layer := range p.Layers {
		sb.WriteString(fmt.Sprintf("    Layer %d: %v\n", i, layer))
	}
	sb.WriteString(fmt.Sprintf("}"))
	return sb.String()
}

// Validate checks if the execution plan is valid.
func (p *ExecutionPlan) Validate() error {
	// Check total tasks match
	actualTasks := 0
	for _, layer := range p.Layers {
		actualTasks += len(layer)
	}
	if actualTasks != p.TotalTasks {
		return fmt.Errorf("task count mismatch: expected %d, got %d", p.TotalTasks, actualTasks)
	}

	// Check all tasks are accounted for
	seen := make(map[string]bool)
	for _, layer := range p.Layers {
		for _, id := range layer {
			if seen[id] {
				return fmt.Errorf("task %s appears in multiple layers", id)
			}
			seen[id] = true
			if _, ok := p.taskMap[id]; !ok {
				return fmt.Errorf("task %s in layers but not in task map", id)
			}
		}
	}

	return nil
}
