// Package badger provides a Badger-based implementation of the storage interface.
package badger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/goclaw/goclaw/pkg/storage"
)

// Config holds configuration for BadgerStorage.
type Config struct {
	Path              string
	SyncWrites        bool
	ValueLogFileSize  int64
	NumVersionsToKeep int
}

// BadgerStorage implements the Storage interface using Badger.
type BadgerStorage struct {
	db     *badger.DB
	config *Config
}

// NewBadgerStorage creates a new Badger storage instance.
func NewBadgerStorage(config *Config) (*BadgerStorage, error) {
	opts := badger.DefaultOptions(config.Path)
	opts.SyncWrites = config.SyncWrites
	opts.ValueLogFileSize = config.ValueLogFileSize
	opts.NumVersionsToKeep = config.NumVersionsToKeep

	db, err := badger.Open(opts)
	if err != nil {
		return nil, &storage.StorageUnavailableError{Cause: err}
	}

	return &BadgerStorage{
		db:     db,
		config: config,
	}, nil
}

// Key generation functions
func workflowKey(id string) []byte {
	return []byte(fmt.Sprintf("workflow:%s", id))
}

func taskKey(workflowID, taskID string) []byte {
	return []byte(fmt.Sprintf("workflow:%s:task:%s", workflowID, taskID))
}

func workflowIndexStatusKey(status, id string) []byte {
	return []byte(fmt.Sprintf("workflow:index:status:%s:%s", status, id))
}

func workflowIndexCreatedKey(timestamp time.Time, id string) []byte {
	return []byte(fmt.Sprintf("workflow:index:created:%d:%s", timestamp.Unix(), id))
}

// Serialization helpers
func serialize(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, &storage.SerializationError{
			Operation: "marshal",
			Cause:     err,
		}
	}
	return data, nil
}

func deserialize(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return &storage.SerializationError{
			Operation: "unmarshal",
			Cause:     err,
		}
	}
	return nil
}

// SaveWorkflow saves a workflow to Badger.
func (b *BadgerStorage) SaveWorkflow(ctx context.Context, wf *storage.WorkflowState) error {
	data, err := serialize(wf)
	if err != nil {
		return err
	}

	return b.db.Update(func(txn *badger.Txn) error {
		// Save workflow data
		if err := txn.Set(workflowKey(wf.ID), data); err != nil {
			return err
		}

		// Update status index
		if err := txn.Set(workflowIndexStatusKey(wf.Status, wf.ID), []byte{}); err != nil {
			return err
		}

		// Update created time index
		if err := txn.Set(workflowIndexCreatedKey(wf.CreatedAt, wf.ID), []byte{}); err != nil {
			return err
		}

		return nil
	})
}

// GetWorkflow retrieves a workflow by ID.
func (b *BadgerStorage) GetWorkflow(ctx context.Context, id string) (*storage.WorkflowState, error) {
	var wf storage.WorkflowState

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(workflowKey(id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return &storage.NotFoundError{
					EntityType: "workflow",
					ID:         id,
				}
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return deserialize(val, &wf)
		})
	})

	if err != nil {
		return nil, err
	}

	return &wf, nil
}

// ListWorkflows lists workflows with optional filtering and pagination.
func (b *BadgerStorage) ListWorkflows(ctx context.Context, filter *storage.WorkflowFilter) ([]*storage.WorkflowState, int, error) {
	var workflows []*storage.WorkflowState

	err := b.db.View(func(txn *badger.Txn) error {
		// If status filter is specified, use status index
		if filter != nil && len(filter.Status) > 0 {
			for _, status := range filter.Status {
				prefix := []byte(fmt.Sprintf("workflow:index:status:%s:", status))
				opts := badger.DefaultIteratorOptions
				opts.Prefix = prefix
				opts.PrefetchValues = false

				it := txn.NewIterator(opts)
				defer it.Close()

				for it.Rewind(); it.Valid(); it.Next() {
					item := it.Item()
					key := string(item.Key())
					// Extract workflow ID from index key: workflow:index:status:{status}:{id}
					parts := strings.Split(key, ":")
					if len(parts) >= 5 {
						workflowID := strings.Join(parts[4:], ":") // Handle IDs with colons
						wf, err := b.getWorkflowInTxn(txn, workflowID)
						if err != nil {
							continue // Skip if workflow not found
						}
						workflows = append(workflows, wf)
					}
				}
			}
		} else {
			// No filter, scan all workflows
			prefix := []byte("workflow:")
			opts := badger.DefaultIteratorOptions
			opts.Prefix = prefix

			it := txn.NewIterator(opts)
			defer it.Close()

			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				key := string(item.Key())

				// Skip index keys
				if strings.Contains(key, ":index:") || strings.Contains(key, ":task:") {
					continue
				}

				var wf storage.WorkflowState
				err := item.Value(func(val []byte) error {
					return deserialize(val, &wf)
				})
				if err != nil {
					continue
				}

				workflows = append(workflows, &wf)
			}
		}

		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	total := len(workflows)

	// Apply pagination
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		end := filter.Offset + filter.Limit

		if start > len(workflows) {
			start = len(workflows)
		}
		if end > len(workflows) {
			end = len(workflows)
		}

		workflows = workflows[start:end]
	}

	return workflows, total, nil
}

// getWorkflowInTxn retrieves a workflow within a transaction.
func (b *BadgerStorage) getWorkflowInTxn(txn *badger.Txn, id string) (*storage.WorkflowState, error) {
	var wf storage.WorkflowState

	item, err := txn.Get(workflowKey(id))
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, &storage.NotFoundError{
				EntityType: "workflow",
				ID:         id,
			}
		}
		return nil, err
	}

	err = item.Value(func(val []byte) error {
		return deserialize(val, &wf)
	})

	if err != nil {
		return nil, err
	}

	return &wf, nil
}

// DeleteWorkflow deletes a workflow and all its tasks.
func (b *BadgerStorage) DeleteWorkflow(ctx context.Context, id string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		// Check if workflow exists
		_, err := b.getWorkflowInTxn(txn, id)
		if err != nil {
			return err
		}

		// Delete workflow data
		if err := txn.Delete(workflowKey(id)); err != nil {
			return err
		}

		// Delete all tasks for this workflow
		prefix := []byte(fmt.Sprintf("workflow:%s:task:", id))
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			if err := txn.Delete(it.Item().Key()); err != nil {
				return err
			}
		}

		// Delete index entries (status and created)
		// Note: We'd need to know the status and created time to delete specific index entries
		// For simplicity, we'll leave orphaned index entries (they'll be ignored on read)

		return nil
	})
}

// SaveTask saves a task state.
func (b *BadgerStorage) SaveTask(ctx context.Context, workflowID string, task *storage.TaskState) error {
	// Verify workflow exists
	_, err := b.GetWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}

	data, err := serialize(task)
	if err != nil {
		return err
	}

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(taskKey(workflowID, task.ID), data)
	})
}

// GetTask retrieves a task by workflow ID and task ID.
func (b *BadgerStorage) GetTask(ctx context.Context, workflowID, taskID string) (*storage.TaskState, error) {
	var task storage.TaskState

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(taskKey(workflowID, taskID))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return &storage.NotFoundError{
					EntityType: "task",
					ID:         taskID,
				}
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return deserialize(val, &task)
		})
	})

	if err != nil {
		return nil, err
	}

	return &task, nil
}

// ListTasks lists all tasks for a workflow.
func (b *BadgerStorage) ListTasks(ctx context.Context, workflowID string) ([]*storage.TaskState, error) {
	// Verify workflow exists
	_, err := b.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	var tasks []*storage.TaskState

	err = b.db.View(func(txn *badger.Txn) error {
		prefix := []byte(fmt.Sprintf("workflow:%s:task:", workflowID))
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			var task storage.TaskState
			err := item.Value(func(val []byte) error {
				return deserialize(val, &task)
			})
			if err != nil {
				continue
			}

			tasks = append(tasks, &task)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// Close closes the Badger database.
func (b *BadgerStorage) Close() error {
	// Run garbage collection before closing
	if err := b.db.RunValueLogGC(0.5); err != nil && err != badger.ErrNoRewrite {
		// Log error but don't fail close
	}

	return b.db.Close()
}
