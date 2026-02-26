package dag

import (
	"container/list"
)

// TopologicalSort returns a topological ordering of tasks using Kahn's algorithm.
// Returns CyclicDependencyError if the graph contains a cycle.
// Time complexity: O(V+E)
// Space complexity: O(V)
func (g *Graph) TopologicalSort() ([]string, error) {
	// Ensure edges are built
	g.rebuildEdges()

	// Use cached result if graph hasn't changed
	if !g.dirty && g.sorted != nil {
		sorted := make([]string, len(g.sorted))
		copy(sorted, g.sorted)
		return sorted, nil
	}

	if len(g.tasks) == 0 {
		return []string{}, nil
	}

	// Make a copy of in-degrees since we'll modify them
	inDegree := make(map[string]int, len(g.inDegree))
	for id, degree := range g.inDegree {
		inDegree[id] = degree
	}

	// Initialize queue with all nodes that have no dependencies
	queue := list.New()
	for id, degree := range inDegree {
		if degree == 0 {
			queue.PushBack(id)
		}
	}

	result := make([]string, 0, len(g.tasks))

	// Process queue
	for queue.Len() > 0 {
		// Dequeue
		elem := queue.Front()
		queue.Remove(elem)
		node := elem.Value.(string)

		result = append(result, node)

		// Reduce in-degree of all neighbors
		for _, neighbor := range g.edges[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue.PushBack(neighbor)
			}
		}
	}

	// Check if all nodes were processed
	if len(result) != len(g.tasks) {
		// Graph has a cycle, find and report it
		if cycle, hasCycle := g.DetectCycle(); hasCycle {
			return nil, cycle
		}
		return nil, &CyclicDependencyError{Path: []string{"unknown"}}
	}

	// Cache the result
	g.sorted = make([]string, len(result))
	copy(g.sorted, result)
	g.dirty = false

	return result, nil
}

// TopologicalSortDFS returns a topological ordering using DFS.
// This is an alternative implementation for comparison/testing.
// Time complexity: O(V+E)
func (g *Graph) TopologicalSortDFS() ([]string, error) {
	if len(g.tasks) == 0 {
		return []string{}, nil
	}

	// Check for cycles first
	if cycle, hasCycle := g.DetectCycle(); hasCycle {
		return nil, cycle
	}

	visited := make(map[string]bool)
	result := make([]string, 0, len(g.tasks))

	var dfs func(node string)
	dfs = func(node string) {
		if visited[node] {
			return
		}
		visited[node] = true

		// Visit all dependencies first
		task := g.tasks[node]
		for _, depID := range task.Deps {
			if !visited[depID] {
				dfs(depID)
			}
		}

		result = append(result, node)
	}

	// Visit all nodes
	for id := range g.tasks {
		if !visited[id] {
			dfs(id)
		}
	}

	return result, nil
}

// IsTopologicalOrder checks if the given order is a valid topological ordering.
func (g *Graph) IsTopologicalOrder(order []string) bool {
	if len(order) != len(g.tasks) {
		return false
	}

	// Check all tasks are present
	for _, id := range order {
		if _, exists := g.tasks[id]; !exists {
			return false
		}
	}

	// Check that for every edge u->v, u comes before v
	position := make(map[string]int, len(order))
	for i, id := range order {
		position[id] = i
	}

	for id, task := range g.tasks {
		for _, depID := range task.Deps {
			if position[depID] >= position[id] {
				// Dependency comes after the task
				return false
			}
		}
	}

	return true
}

// Levels returns tasks grouped by their depth (layer) in the DAG.
// Tasks in the same layer can be executed in parallel.
// Layer 0 contains root tasks (no dependencies).
func (g *Graph) Levels() ([][]string, error) {
	if len(g.tasks) == 0 {
		return [][]string{}, nil
	}

	// Check for cycles
	if cycle, hasCycle := g.DetectCycle(); hasCycle {
		return nil, cycle
	}

	// Calculate depth for each task
	depth := make(map[string]int, len(g.tasks))

	// Initialize roots to depth 0
	for id := range g.tasks {
		if g.inDegree[id] == 0 {
			depth[id] = 0
		}
	}

	// Topological sort to process in order
	order, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}

	// Calculate depths
	maxDepth := 0
	for _, id := range order {
		task := g.tasks[id]
		for _, depID := range task.Deps {
			if d, ok := depth[depID]; ok {
				if d+1 > depth[id] {
					depth[id] = d + 1
				}
			}
		}
		if depth[id] > maxDepth {
			maxDepth = depth[id]
		}
	}

	// Group by depth
	levels := make([][]string, maxDepth+1)
	for id, d := range depth {
		levels[d] = append(levels[d], id)
	}

	return levels, nil
}
