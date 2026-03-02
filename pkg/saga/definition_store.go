package saga

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

const sagaDefinitionKeyPrefix = "saga:def:"

// ErrSagaDefinitionNotFound indicates no persisted definition was found for the saga.
var ErrSagaDefinitionNotFound = errors.New("saga definition not found")

// SagaDefinitionStore persists serializable saga definition snapshots keyed by saga ID.
type SagaDefinitionStore interface {
	Save(ctx context.Context, sagaID string, snapshot *DefinitionSnapshot) error
	Load(ctx context.Context, sagaID string) (*DefinitionSnapshot, error)
	Delete(ctx context.Context, sagaID string) error
	List(ctx context.Context) (map[string]*DefinitionSnapshot, error)
}

// BadgerSagaDefinitionStore stores saga definition snapshots in Badger.
type BadgerSagaDefinitionStore struct {
	db *badger.DB
}

// NewBadgerSagaDefinitionStore creates a new badger-backed definition store.
func NewBadgerSagaDefinitionStore(db *badger.DB) (*BadgerSagaDefinitionStore, error) {
	if db == nil {
		return nil, fmt.Errorf("badger db cannot be nil")
	}
	return &BadgerSagaDefinitionStore{db: db}, nil
}

// Save persists one definition snapshot keyed by saga ID.
func (s *BadgerSagaDefinitionStore) Save(ctx context.Context, sagaID string, snapshot *DefinitionSnapshot) error {
	if strings.TrimSpace(sagaID) == "" {
		return fmt.Errorf("saga id cannot be empty")
	}
	if snapshot == nil {
		return fmt.Errorf("definition snapshot cannot be nil")
	}
	payload := snapshot.Clone()
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal saga definition: %w", err)
	}
	key := []byte(sagaDefinitionKey(sagaID))
	return s.db.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return txn.Set(key, data)
	})
}

// Load retrieves a persisted definition snapshot by saga ID.
func (s *BadgerSagaDefinitionStore) Load(ctx context.Context, sagaID string) (*DefinitionSnapshot, error) {
	if strings.TrimSpace(sagaID) == "" {
		return nil, fmt.Errorf("saga id cannot be empty")
	}
	var snapshot DefinitionSnapshot
	err := s.db.View(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		item, err := txn.Get([]byte(sagaDefinitionKey(sagaID)))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrSagaDefinitionNotFound
			}
			return err
		}
		return item.Value(func(v []byte) error {
			return json.Unmarshal(v, &snapshot)
		})
	})
	if err != nil {
		return nil, err
	}
	return snapshot.Clone(), nil
}

// Delete removes a stored definition snapshot by saga ID.
func (s *BadgerSagaDefinitionStore) Delete(ctx context.Context, sagaID string) error {
	if strings.TrimSpace(sagaID) == "" {
		return fmt.Errorf("saga id cannot be empty")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		key := []byte(sagaDefinitionKey(sagaID))
		err := txn.Delete(key)
		if err == badger.ErrKeyNotFound {
			return ErrSagaDefinitionNotFound
		}
		return err
	})
}

// List returns all persisted definition snapshots keyed by saga ID.
func (s *BadgerSagaDefinitionStore) List(ctx context.Context) (map[string]*DefinitionSnapshot, error) {
	results := make(map[string]*DefinitionSnapshot)
	prefix := []byte(sagaDefinitionKeyPrefix)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			item := it.Item()
			key := string(item.Key())
			sagaID := strings.TrimPrefix(key, sagaDefinitionKeyPrefix)
			if sagaID == "" {
				continue
			}
			var snapshot DefinitionSnapshot
			if err := item.Value(func(v []byte) error {
				return json.Unmarshal(v, &snapshot)
			}); err != nil {
				return err
			}
			results[sagaID] = snapshot.Clone()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func sagaDefinitionKey(sagaID string) string {
	return sagaDefinitionKeyPrefix + sagaID
}
