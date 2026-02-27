package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
)

const checkpointKeyPrefix = "checkpoint:"

// Checkpoint stores resumable Saga runtime state.
type Checkpoint struct {
	SagaID         string         `json:"saga_id"`
	State          SagaState      `json:"state"`
	CompletedSteps []string       `json:"completed_steps"`
	FailedStep     string         `json:"failed_step,omitempty"`
	StepResults    map[string]any `json:"step_results,omitempty"`
	LastUpdated    time.Time      `json:"last_updated"`
}

// CheckpointStore persists and retrieves checkpoint snapshots.
type CheckpointStore interface {
	Save(ctx context.Context, checkpoint *Checkpoint) error
	Load(ctx context.Context, sagaID string) (*Checkpoint, error)
	Delete(ctx context.Context, sagaID string) error
}

// BadgerCheckpointStore stores checkpoints in Badger.
type BadgerCheckpointStore struct {
	db *badger.DB
}

// NewBadgerCheckpointStore creates a checkpoint store.
func NewBadgerCheckpointStore(db *badger.DB) (*BadgerCheckpointStore, error) {
	if db == nil {
		return nil, fmt.Errorf("badger db cannot be nil")
	}
	return &BadgerCheckpointStore{db: db}, nil
}

// Save writes checkpoint at key "checkpoint:{sagaID}".
func (s *BadgerCheckpointStore) Save(ctx context.Context, checkpoint *Checkpoint) error {
	if checkpoint == nil {
		return fmt.Errorf("checkpoint cannot be nil")
	}
	if checkpoint.SagaID == "" {
		return fmt.Errorf("checkpoint saga_id cannot be empty")
	}
	if checkpoint.LastUpdated.IsZero() {
		checkpoint.LastUpdated = time.Now().UTC()
	}

	data, err := SerializeCheckpoint(checkpoint)
	if err != nil {
		return err
	}

	key := []byte(checkpointKey(checkpoint.SagaID))
	return s.db.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return txn.Set(key, data)
	})
}

// Load reads checkpoint from key "checkpoint:{sagaID}".
func (s *BadgerCheckpointStore) Load(ctx context.Context, sagaID string) (*Checkpoint, error) {
	key := []byte(checkpointKey(sagaID))
	var checkpoint Checkpoint

	err := s.db.View(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("checkpoint not found for saga %s", sagaID)
			}
			return err
		}
		return item.Value(func(v []byte) error {
			cp, decErr := DeserializeCheckpoint(v)
			if decErr != nil {
				return decErr
			}
			checkpoint = *cp
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return &checkpoint, nil
}

// Delete removes checkpoint data for one saga.
func (s *BadgerCheckpointStore) Delete(ctx context.Context, sagaID string) error {
	key := []byte(checkpointKey(sagaID))
	return s.db.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return txn.Delete(key)
	})
}

// SerializeCheckpoint serializes checkpoint to JSON.
func SerializeCheckpoint(checkpoint *Checkpoint) ([]byte, error) {
	data, err := json.Marshal(checkpoint)
	if err != nil {
		return nil, fmt.Errorf("serialize checkpoint: %w", err)
	}
	return data, nil
}

// DeserializeCheckpoint deserializes checkpoint JSON.
func DeserializeCheckpoint(data []byte) (*Checkpoint, error) {
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("deserialize checkpoint: %w", err)
	}
	return &checkpoint, nil
}

// Checkpointer provides utility helpers for auto checkpoint creation.
type Checkpointer struct {
	store CheckpointStore
}

// NewCheckpointer creates a helper for auto-checkpoint updates.
func NewCheckpointer(store CheckpointStore) (*Checkpointer, error) {
	if store == nil {
		return nil, fmt.Errorf("checkpoint store cannot be nil")
	}
	return &Checkpointer{store: store}, nil
}

// RecordStepCompletion updates runtime state and persists the latest checkpoint.
func (c *Checkpointer) RecordStepCompletion(ctx context.Context, instance *SagaInstance, stepID string, result any) error {
	if instance == nil {
		return fmt.Errorf("saga instance cannot be nil")
	}

	instance.MarkStepCompleted(stepID, result)
	cp := &Checkpoint{
		SagaID:         instance.ID,
		State:          instance.State,
		CompletedSteps: append([]string(nil), instance.CompletedSteps...),
		FailedStep:     instance.FailedStep,
		StepResults:    copyResultMap(instance.StepResults),
		LastUpdated:    time.Now().UTC(),
	}
	return c.store.Save(ctx, cp)
}

// Snapshot creates a checkpoint from an instance without persisting.
func Snapshot(instance *SagaInstance) *Checkpoint {
	if instance == nil {
		return nil
	}
	return &Checkpoint{
		SagaID:         instance.ID,
		State:          instance.State,
		CompletedSteps: append([]string(nil), instance.CompletedSteps...),
		FailedStep:     instance.FailedStep,
		StepResults:    copyResultMap(instance.StepResults),
		LastUpdated:    time.Now().UTC(),
	}
}

func checkpointKey(sagaID string) string {
	return checkpointKeyPrefix + sagaID
}

func copyResultMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	copied := make(map[string]any, len(source))
	for k, v := range source {
		copied[k] = v
	}
	return copied
}
