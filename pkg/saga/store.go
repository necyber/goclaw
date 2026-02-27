package saga

import (
	"context"
	"fmt"
	"sync"
)

// SagaListFilter controls saga list query behavior.
type SagaListFilter struct {
	State  string
	Limit  int
	Offset int
}

// SagaStore provides persistence for Saga instances.
type SagaStore interface {
	Save(ctx context.Context, instance *SagaInstance) error
	Get(ctx context.Context, sagaID string) (*SagaInstance, error)
	List(ctx context.Context, filter SagaListFilter) ([]*SagaInstance, int, error)
	Delete(ctx context.Context, sagaID string) error
}

// MemorySagaStore is an in-memory SagaStore implementation.
type MemorySagaStore struct {
	mu        sync.RWMutex
	instances map[string]*SagaInstance
}

// NewMemorySagaStore creates an in-memory saga store.
func NewMemorySagaStore() *MemorySagaStore {
	return &MemorySagaStore{
		instances: make(map[string]*SagaInstance),
	}
}

// Save saves a saga instance.
func (s *MemorySagaStore) Save(_ context.Context, instance *SagaInstance) error {
	if instance == nil {
		return fmt.Errorf("saga instance cannot be nil")
	}
	s.mu.Lock()
	s.instances[instance.ID] = cloneInstance(instance)
	s.mu.Unlock()
	return nil
}

// Get gets one saga instance by id.
func (s *MemorySagaStore) Get(_ context.Context, sagaID string) (*SagaInstance, error) {
	s.mu.RLock()
	instance, ok := s.instances[sagaID]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrSagaNotFound
	}
	return cloneInstance(instance), nil
}

// List lists saga instances with optional state filter and pagination.
func (s *MemorySagaStore) List(_ context.Context, filter SagaListFilter) ([]*SagaInstance, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]*SagaInstance, 0, len(s.instances))
	for _, instance := range s.instances {
		if filter.State != "" && instance.State.String() != filter.State {
			continue
		}
		all = append(all, cloneInstance(instance))
	}
	total := len(all)

	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.Limit < 0 {
		filter.Limit = 0
	}
	if filter.Offset > total {
		filter.Offset = total
	}
	end := total
	if filter.Limit > 0 && filter.Offset+filter.Limit < end {
		end = filter.Offset + filter.Limit
	}

	return all[filter.Offset:end], total, nil
}

// Delete removes one saga instance.
func (s *MemorySagaStore) Delete(_ context.Context, sagaID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.instances[sagaID]; !ok {
		return ErrSagaNotFound
	}
	delete(s.instances, sagaID)
	return nil
}
