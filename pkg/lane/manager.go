package lane

import (
	"context"
	"fmt"
	"sync"
)

// Manager manages multiple lanes.
type Manager struct {
	lanes   map[string]Lane
	configs map[string]*Config
	mu      sync.RWMutex
}

// NewManager creates a new Lane Manager.
func NewManager() *Manager {
	return &Manager{
		lanes:   make(map[string]Lane),
		configs: make(map[string]*Config),
	}
}

// Register registers a new lane with the given configuration.
// Returns an error if a lane with the same name already exists.
func (m *Manager) Register(config *Config) (Lane, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.lanes[config.Name]; exists {
		return nil, &DuplicateLaneError{LaneName: config.Name}
	}
	
	lane, err := New(config)
	if err != nil {
		return nil, err
	}
	
	// Set manager for redirect strategy
	if config.Backpressure == Redirect {
		lane.SetManager(m)
	}
	
	m.lanes[config.Name] = lane
	m.configs[config.Name] = config
	
	// Start the lane's main loop
	lane.Run()
	
	return lane, nil
}

// RegisterLane registers an existing lane.
func (m *Manager) RegisterLane(lane Lane) error {
	if lane == nil {
		return fmt.Errorf("lane cannot be nil")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	name := lane.Name()
	if _, exists := m.lanes[name]; exists {
		return &DuplicateLaneError{LaneName: name}
	}
	
	m.lanes[name] = lane
	return nil
}

// GetLane returns a lane by name.
// Returns LaneNotFoundError if the lane doesn't exist.
func (m *Manager) GetLane(name string) (Lane, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	lane, exists := m.lanes[name]
	if !exists {
		return nil, &LaneNotFoundError{LaneName: name}
	}
	
	return lane, nil
}

// Unregister removes a lane from the manager.
// The lane is closed before being removed.
func (m *Manager) Unregister(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	lane, exists := m.lanes[name]
	if !exists {
		return &LaneNotFoundError{LaneName: name}
	}
	
	// Close the lane
	if err := lane.Close(ctx); err != nil {
		return fmt.Errorf("failed to close lane %s: %w", name, err)
	}
	
	delete(m.lanes, name)
	delete(m.configs, name)
	
	return nil
}

// Submit submits a task to the appropriate lane based on task.Lane().
func (m *Manager) Submit(ctx context.Context, task Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	
	laneName := task.Lane()
	if laneName == "" {
		return fmt.Errorf("task lane cannot be empty")
	}
	
	lane, err := m.GetLane(laneName)
	if err != nil {
		return err
	}
	
	return lane.Submit(ctx, task)
}

// TrySubmit attempts to submit a task without blocking.
func (m *Manager) TrySubmit(task Task) bool {
	if task == nil {
		return false
	}
	
	laneName := task.Lane()
	if laneName == "" {
		return false
	}
	
	lane, err := m.GetLane(laneName)
	if err != nil {
		return false
	}
	
	return lane.TrySubmit(task)
}

// GetStats returns statistics for all lanes.
func (m *Manager) GetStats() map[string]Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := make(map[string]Stats, len(m.lanes))
	for name, lane := range m.lanes {
		stats[name] = lane.Stats()
	}
	
	return stats
}

// Close gracefully closes all lanes.
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var errs []error
	for name, lane := range m.lanes {
		if err := lane.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close lane %s: %w", name, err))
		}
	}
	
	// Clear the maps
	m.lanes = make(map[string]Lane)
	m.configs = make(map[string]*Config)
	
	if len(errs) > 0 {
		return fmt.Errorf("errors closing lanes: %v", errs)
	}
	
	return nil
}

// LaneNames returns a list of all lane names.
func (m *Manager) LaneNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.lanes))
	for name := range m.lanes {
		names = append(names, name)
	}
	
	return names
}

// HasLane returns true if a lane with the given name exists.
func (m *Manager) HasLane(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.lanes[name]
	return exists
}
