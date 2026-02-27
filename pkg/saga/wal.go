package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

const (
	walKeyPrefix      = "wal:"
	walSequencePrefix = "wal-seq:"
)

// WALEntryType identifies one Saga state-change event.
type WALEntryType string

const (
	WALEntryTypeStepStarted           WALEntryType = "step_started"
	WALEntryTypeStepCompleted         WALEntryType = "step_completed"
	WALEntryTypeStepFailed            WALEntryType = "step_failed"
	WALEntryTypeCompensationStarted   WALEntryType = "compensation_started"
	WALEntryTypeCompensationCompleted WALEntryType = "compensation_completed"
	WALEntryTypeCompensationFailed    WALEntryType = "compensation_failed"
)

// WALEntry is one durable write-ahead log record.
type WALEntry struct {
	Sequence  uint64       `json:"sequence"`
	SagaID    string       `json:"saga_id"`
	StepID    string       `json:"step_id,omitempty"`
	Type      WALEntryType `json:"type"`
	Data      []byte       `json:"data,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// WALWriteMode controls whether writes are synchronous or asynchronous.
type WALWriteMode string

const (
	// WALWriteModeSync flushes each append before return.
	WALWriteModeSync WALWriteMode = "sync"
	// WALWriteModeAsync enqueues writes and flushes in background.
	WALWriteModeAsync WALWriteMode = "async"
)

// WAL provides append-only event logging for Saga state changes.
type WAL interface {
	Append(ctx context.Context, entry WALEntry) (uint64, error)
	List(ctx context.Context, sagaID string) ([]WALEntry, error)
	DeleteBySagaID(ctx context.Context, sagaID string) error
	Close() error
}

// WALOptions configures a Badger-backed WAL.
type WALOptions struct {
	WriteMode      WALWriteMode
	AsyncQueueSize int
}

type walAppendRequest struct {
	ctx   context.Context
	entry WALEntry
}

// BadgerWAL implements WAL on top of Badger.
type BadgerWAL struct {
	db        *badger.DB
	ownsDB    bool
	writeMode WALWriteMode

	appendCh chan walAppendRequest
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// OpenBadgerWAL opens a dedicated Badger DB for WAL usage.
func OpenBadgerWAL(path string, options WALOptions) (*BadgerWAL, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger wal: %w", err)
	}
	wal, err := NewBadgerWAL(db, options)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	wal.ownsDB = true
	return wal, nil
}

// NewBadgerWAL creates a WAL over an existing Badger DB instance.
func NewBadgerWAL(db *badger.DB, options WALOptions) (*BadgerWAL, error) {
	if db == nil {
		return nil, fmt.Errorf("badger db cannot be nil")
	}
	if options.WriteMode == "" {
		options.WriteMode = WALWriteModeSync
	}
	if options.AsyncQueueSize <= 0 {
		options.AsyncQueueSize = 1024
	}
	if options.WriteMode != WALWriteModeSync && options.WriteMode != WALWriteModeAsync {
		return nil, fmt.Errorf("unsupported wal write mode: %s", options.WriteMode)
	}

	wal := &BadgerWAL{
		db:        db,
		writeMode: options.WriteMode,
		stopCh:    make(chan struct{}),
	}

	if options.WriteMode == WALWriteModeAsync {
		wal.appendCh = make(chan walAppendRequest, options.AsyncQueueSize)
		wal.wg.Add(1)
		go wal.runAsyncWriter()
	}

	return wal, nil
}

// Append appends one WAL entry and returns its sequence number.
func (w *BadgerWAL) Append(ctx context.Context, entry WALEntry) (uint64, error) {
	if entry.SagaID == "" {
		return 0, fmt.Errorf("wal entry saga_id cannot be empty")
	}
	if entry.Type == "" {
		return 0, fmt.Errorf("wal entry type cannot be empty")
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	sequence, err := w.nextSequence(entry.SagaID)
	if err != nil {
		return 0, err
	}
	entry.Sequence = sequence

	if w.writeMode == WALWriteModeAsync {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-w.stopCh:
			return 0, fmt.Errorf("wal is closed")
		case w.appendCh <- walAppendRequest{ctx: ctx, entry: entry}:
			return sequence, nil
		default:
			// Fall back to synchronous write when queue is full.
			if err := w.writeEntry(ctx, entry); err != nil {
				return 0, err
			}
			return sequence, nil
		}
	}

	if err := w.writeEntry(ctx, entry); err != nil {
		return 0, err
	}
	return sequence, nil
}

// List returns all WAL entries for a saga in sequence order.
func (w *BadgerWAL) List(ctx context.Context, sagaID string) ([]WALEntry, error) {
	prefix := []byte(walPrefixForSaga(sagaID))
	entries := make([]WALEntry, 0)

	err := w.db.View(func(txn *badger.Txn) error {
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
			var entry WALEntry
			if err := item.Value(func(v []byte) error {
				return json.Unmarshal(v, &entry)
			}); err != nil {
				return fmt.Errorf("decode wal entry: %w", err)
			}
			entries = append(entries, entry)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// DeleteBySagaID removes all WAL entries for a saga.
func (w *BadgerWAL) DeleteBySagaID(ctx context.Context, sagaID string) error {
	prefix := []byte(walPrefixForSaga(sagaID))
	seqKey := []byte(sequenceKeyForSaga(sagaID))
	keys := make([][]byte, 0)

	if err := w.db.View(func(txn *badger.Txn) error {
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
			key := it.Item().KeyCopy(nil)
			keys = append(keys, key)
		}
		return nil
	}); err != nil {
		return err
	}

	return w.db.Update(func(txn *badger.Txn) error {
		for _, key := range keys {
			if err := txn.Delete(key); err != nil {
				return err
			}
		}
		_ = txn.Delete(seqKey)
		return nil
	})
}

// Close stops background routines and closes db if owned.
func (w *BadgerWAL) Close() error {
	close(w.stopCh)
	if w.appendCh != nil {
		close(w.appendCh)
	}
	w.wg.Wait()
	if w.ownsDB {
		return w.db.Close()
	}
	return nil
}

func (w *BadgerWAL) runAsyncWriter() {
	defer w.wg.Done()
	for req := range w.appendCh {
		if err := w.writeEntry(req.ctx, req.entry); err != nil {
			// Best effort logging path is intentionally omitted to keep package independent.
			_ = err
		}
	}
}

func (w *BadgerWAL) writeEntry(ctx context.Context, entry WALEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal wal entry: %w", err)
	}
	key := []byte(walEntryKey(entry.SagaID, entry.Sequence))

	return w.db.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return txn.Set(key, data)
	})
}

func (w *BadgerWAL) nextSequence(sagaID string) (uint64, error) {
	key := []byte(sequenceKeyForSaga(sagaID))
	var next uint64
	err := w.db.Update(func(txn *badger.Txn) error {
		current := uint64(0)
		item, err := txn.Get(key)
		switch {
		case err == nil:
			if err := item.Value(func(v []byte) error {
				parsed, parseErr := strconv.ParseUint(string(v), 10, 64)
				if parseErr != nil {
					return parseErr
				}
				current = parsed
				return nil
			}); err != nil {
				return err
			}
		case err == badger.ErrKeyNotFound:
			current = 0
		default:
			return err
		}

		next = current + 1
		return txn.Set(key, []byte(strconv.FormatUint(next, 10)))
	})
	if err != nil {
		return 0, fmt.Errorf("next wal sequence: %w", err)
	}
	return next, nil
}

func walPrefixForSaga(sagaID string) string {
	return fmt.Sprintf("%s%s:", walKeyPrefix, sagaID)
}

func sequenceKeyForSaga(sagaID string) string {
	return fmt.Sprintf("%s%s", walSequencePrefix, sagaID)
}

func walEntryKey(sagaID string, sequence uint64) string {
	return fmt.Sprintf("%s%s:%020d", walKeyPrefix, sagaID, sequence)
}

func parseSequenceFromWALKey(key string) (uint64, error) {
	parts := strings.Split(key, ":")
	if len(parts) < 3 {
		return 0, fmt.Errorf("invalid wal key format: %s", key)
	}
	sequenceRaw := parts[len(parts)-1]
	return strconv.ParseUint(sequenceRaw, 10, 64)
}
