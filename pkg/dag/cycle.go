package dag

// DetectCycle uses DFS with three-color marking to detect cycles.
// Returns (nil, false) if no cycle exists.
// Returns (CyclicDependencyError, true) if a cycle is found.
// Time complexity: O(V+E)
// Space complexity: O(V)
func (g *Graph) DetectCycle() (*CyclicDependencyError, bool) {
	if len(g.tasks) == 0 {
		return nil, false
	}
	
	// Ensure edges are built from task dependencies
	g.rebuildEdges()
	
	// Three-color marking:
	// white (0): not visited
	// gray (1): being processed (in current DFS path)
	// black (2): finished processing
	color := make(map[string]int, len(g.tasks))
	
	// Initialize all to white
	for id := range g.tasks {
		color[id] = 0
	}
	
	// DFS from each unvisited node
	for id := range g.tasks {
		if color[id] == 0 {
			if cycle := g.dfsCycle(id, color, []string{}); cycle != nil {
				return &CyclicDependencyError{Path: cycle}, true
			}
		}
	}
	
	return nil, false
}

// dfsCycle performs DFS and returns the cycle path if found.
func (g *Graph) dfsCycle(node string, color map[string]int, path []string) []string {
	// Mark as being processed (gray)
	color[node] = 1
	
	// Add to current path
	path = append(path, node)
	
	// Visit all neighbors (tasks that depend on this task)
	for _, neighbor := range g.edges[node] {
		switch color[neighbor] {
		case 0: // white - not visited
			if cycle := g.dfsCycle(neighbor, color, path); cycle != nil {
				return cycle
			}
		case 1: // gray - found a back edge (cycle)
			// Found cycle, construct path
			return g.constructCyclePath(path, neighbor)
		case 2: // black - already processed, skip
			// No cycle through this node
		}
	}
	
	// Mark as finished (black)
	color[node] = 2
	
	return nil
}

// constructCyclePath constructs the cycle path from DFS path.
func (g *Graph) constructCyclePath(path []string, cycleStart string) []string {
	// Find the start of the cycle in the path
	startIdx := -1
	for i, node := range path {
		if node == cycleStart {
			startIdx = i
			break
		}
	}
	
	if startIdx == -1 {
		return []string{cycleStart, cycleStart}
	}
	
	// Extract the cycle and append the start node to close the loop
	cycle := make([]string, len(path)-startIdx+1)
	copy(cycle, path[startIdx:])
	cycle[len(cycle)-1] = cycleStart
	
	return cycle
}

// HasCycle is a convenience method that returns true if the graph has a cycle.
func (g *Graph) HasCycle() bool {
	_, hasCycle := g.DetectCycle()
	return hasCycle
}

// FindAllCycles finds all elementary cycles in the graph.
// Note: This can be expensive for large graphs with many cycles.
// Simplified implementation that finds cycles during DFS.
func (g *Graph) FindAllCycles() [][]string {
	if len(g.tasks) == 0 {
		return nil
	}
	
	g.rebuildEdges()
	
	var cycles [][]string
	
	// Use three-color DFS to find all back edges (cycles)
	color := make(map[string]int) // 0=white, 1=gray, 2=black
	for id := range g.tasks {
		color[id] = 0
	}
	
	var findCycles func(node string, path []string)
	findCycles = func(node string, path []string) {
		color[node] = 1 // gray
		path = append(path, node)
		
		for _, neighbor := range g.edges[node] {
			switch color[neighbor] {
			case 0: // white, continue DFS
				findCycles(neighbor, path)
			case 1: // gray, found cycle
				cycle := g.extractCycle(path, neighbor)
				if len(cycle) > 0 {
					cycles = append(cycles, cycle)
				}
			case 2: // black, already processed
				// No cycle through this path
			}
		}
		
		color[node] = 2 // black
	}
	
	// Run from all unvisited nodes
	for id := range g.tasks {
		if color[id] == 0 {
			findCycles(id, []string{})
		}
	}
	
	return cycles
}

// extractCycle extracts a cycle from the current path ending at target.
func (g *Graph) extractCycle(path []string, target string) []string {
	for i, node := range path {
		if node == target {
			cycle := make([]string, len(path)-i+1)
			copy(cycle, path[i:])
			cycle[len(cycle)-1] = target
			return cycle
		}
	}
	return nil
}
