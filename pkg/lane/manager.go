package lane

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Manager manages multiple lanes.
type Manager struct {
	lanes       map[string]Lane
	configs     map[string]*LaneSpec
	redisClient redis.Cmdable
	mu          sync.RWMutex
}

// NewManager creates a new Lane Manager.
func NewManager() *Manager {
	return &Manager{
		lanes:   make(map[string]Lane),
		configs: make(map[string]*LaneSpec),
	}
}

// SetRedisClient sets the shared Redis client for Redis-backed lanes.
func (m *Manager) SetRedisClient(client redis.Cmdable) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.redisClient = client
}

// Register registers a new lane with the given configuration.
// Returns an error if a lane with the same name already exists.
func (m *Manager) Register(config *Config) (Lane, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return m.RegisterSpec(&LaneSpec{
		Type:   LaneTypeMemory,
		Memory: config,
	})
}

// RegisterSpec registers a new lane based on a LaneSpec.
func (m *Manager) RegisterSpec(spec *LaneSpec) (Lane, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	name := spec.Name()
	if name == "" {
		return nil, fmt.Errorf("lane name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.lanes[name]; exists {
		return nil, &DuplicateLaneError{LaneName: name}
	}

	var lane Lane
	switch spec.Type {
	case LaneTypeMemory:
		memLane, err := New(spec.Memory)
		if err != nil {
			return nil, err
		}
		lane = memLane
	case LaneTypeRedis:
		if m.redisClient == nil {
			return nil, fmt.Errorf("redis client is not configured")
		}
		redisLane, err := NewRedisLane(m.redisClient, spec.Redis)
		if err != nil {
			return nil, err
		}
		lane = redisLane

		if fallbackCfg := spec.fallbackConfig(); fallbackCfg != nil {
			fallbackLane, err := New(fallbackCfg)
			if err != nil {
				return nil, err
			}
			fallback, err := NewFallbackLane(redisLane, fallbackLane, spec.FallbackConfig)
			if err != nil {
				return nil, err
			}
			lane = fallback
		}
	default:
		return nil, fmt.Errorf("unsupported lane type: %s", spec.Type)
	}

	if setter, ok := lane.(interface{ SetManager(*Manager) }); ok {
		setter.SetManager(m)
	}

	m.lanes[name] = lane
	m.configs[name] = spec

	if runner, ok := lane.(interface{ Run() }); ok {
		runner.Run()
	}

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

// AggregateStats returns a single Stats value summed across all lanes.
func (m *Manager) AggregateStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agg := Stats{Name: "all"}
	for _, lane := range m.lanes {
		stats := lane.Stats()
		agg.Pending += stats.Pending
		agg.Running += stats.Running
		agg.Completed += stats.Completed
		agg.Failed += stats.Failed
		agg.Dropped += stats.Dropped
		agg.Capacity += stats.Capacity
		agg.MaxConcurrency += stats.MaxConcurrency
	}

	return agg
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
	m.configs = make(map[string]*LaneSpec)

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
