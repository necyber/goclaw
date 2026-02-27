package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

const (
	sagaKeyPrefix        = "saga:"
	sagaIndexStatePrefix = "saga:index:state:"
)

// BadgerSagaStore stores saga instances in Badger.
type BadgerSagaStore struct {
	db *badger.DB
}

// NewBadgerSagaStore creates a Badger-backed saga store.
func NewBadgerSagaStore(db *badger.DB) (*BadgerSagaStore, error) {
	if db == nil {
		return nil, fmt.Errorf("badger db cannot be nil")
	}
	return &BadgerSagaStore{db: db}, nil
}

// Save persists one saga instance at key "saga:{sagaID}" and state index.
func (s *BadgerSagaStore) Save(ctx context.Context, instance *SagaInstance) error {
	if instance == nil {
		return fmt.Errorf("saga instance cannot be nil")
	}
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	key := []byte(sagaDataKey(instance.ID))
	newIndexKey := []byte(sagaStateIndexKey(instance.State.String(), instance.ID))

	return s.db.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var oldState string
		item, err := txn.Get(key)
		if err == nil {
			var previous SagaInstance
			if err := item.Value(func(v []byte) error { return json.Unmarshal(v, &previous) }); err == nil {
				oldState = previous.State.String()
			}
		}

		if err := txn.Set(key, data); err != nil {
			return err
		}
		if err := txn.Set(newIndexKey, []byte{}); err != nil {
			return err
		}
		if oldState != "" && oldState != instance.State.String() {
			_ = txn.Delete([]byte(sagaStateIndexKey(oldState, instance.ID)))
		}
		return nil
	})
}

// Get loads one saga instance by id.
func (s *BadgerSagaStore) Get(ctx context.Context, sagaID string) (*SagaInstance, error) {
	var instance SagaInstance
	err := s.db.View(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		item, err := txn.Get([]byte(sagaDataKey(sagaID)))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrSagaNotFound
			}
			return err
		}
		return item.Value(func(v []byte) error { return json.Unmarshal(v, &instance) })
	})
	if err != nil {
		return nil, err
	}
	return cloneInstance(&instance), nil
}

// List queries saga instances by state with pagination.
func (s *BadgerSagaStore) List(ctx context.Context, filter SagaListFilter) ([]*SagaInstance, int, error) {
	instances := make([]*SagaInstance, 0)

	err := s.db.View(func(txn *badger.Txn) error {
		if filter.State != "" {
			prefix := []byte(sagaStateIndexPrefix(filter.State))
			opts := badger.DefaultIteratorOptions
			opts.Prefix = prefix
			opts.PrefetchValues = false
			it := txn.NewIterator(opts)
			defer it.Close()

			for it.Rewind(); it.Valid(); it.Next() {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				key := string(it.Item().Key())
				sagaID := strings.TrimPrefix(key, sagaStateIndexPrefix(filter.State))
				instance, err := s.getInTxn(txn, sagaID)
				if err != nil {
					continue
				}
				instances = append(instances, instance)
			}
			return nil
		}

		prefix := []byte(sagaKeyPrefix)
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

			key := string(it.Item().Key())
			if strings.HasPrefix(key, sagaIndexStatePrefix) {
				continue
			}
			var instance SagaInstance
			if err := it.Item().Value(func(v []byte) error { return json.Unmarshal(v, &instance) }); err != nil {
				continue
			}
			instances = append(instances, &instance)
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	total := len(instances)
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	limit := filter.Limit
	if limit < 0 {
		limit = 0
	}
	end := total
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	paged := make([]*SagaInstance, 0, end-offset)
	for _, instance := range instances[offset:end] {
		paged = append(paged, cloneInstance(instance))
	}
	return paged, total, nil
}

// Delete removes one saga instance and state index.
func (s *BadgerSagaStore) Delete(ctx context.Context, sagaID string) error {
	key := []byte(sagaDataKey(sagaID))
	return s.db.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrSagaNotFound
			}
			return err
		}

		var instance SagaInstance
		if err := item.Value(func(v []byte) error { return json.Unmarshal(v, &instance) }); err != nil {
			return err
		}
		if err := txn.Delete(key); err != nil {
			return err
		}
		_ = txn.Delete([]byte(sagaStateIndexKey(instance.State.String(), sagaID)))
		return nil
	})
}

func (s *BadgerSagaStore) getInTxn(txn *badger.Txn, sagaID string) (*SagaInstance, error) {
	item, err := txn.Get([]byte(sagaDataKey(sagaID)))
	if err != nil {
		return nil, err
	}
	var instance SagaInstance
	if err := item.Value(func(v []byte) error { return json.Unmarshal(v, &instance) }); err != nil {
		return nil, err
	}
	return &instance, nil
}

func sagaDataKey(sagaID string) string {
	return sagaKeyPrefix + sagaID
}

func sagaStateIndexPrefix(state string) string {
	return sagaIndexStatePrefix + state + ":"
}

func sagaStateIndexKey(state, sagaID string) string {
	return sagaStateIndexPrefix(state) + sagaID
}
