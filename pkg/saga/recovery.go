package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// RecoveryLogger is the logging subset used by recovery and cleanup services.
type RecoveryLogger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

type nopRecoveryLogger struct{}

func (n *nopRecoveryLogger) Info(string, ...any) {}
func (n *nopRecoveryLogger) Warn(string, ...any) {}

// RecoveryManager coordinates startup recovery from checkpoints.
type RecoveryManager struct {
	orchestrator *SagaOrchestrator
	checkpoints  CheckpointStore
	logger       RecoveryLogger
}

// NewRecoveryManager creates a recovery manager.
func NewRecoveryManager(
	orchestrator *SagaOrchestrator,
	checkpoints CheckpointStore,
	logger RecoveryLogger,
) (*RecoveryManager, error) {
	if orchestrator == nil {
		return nil, fmt.Errorf("orchestrator cannot be nil")
	}
	if checkpoints == nil {
		return nil, fmt.Errorf("checkpoint store cannot be nil")
	}
	if logger == nil {
		logger = &nopRecoveryLogger{}
	}
	return &RecoveryManager{
		orchestrator: orchestrator,
		checkpoints:  checkpoints,
		logger:       logger,
	}, nil
}

// Recover scans incomplete saga checkpoints and resumes execution.
func (m *RecoveryManager) Recover(
	ctx context.Context,
	definitions map[string]*SagaDefinition,
	inputProvider func(sagaID string) any,
) (int, error) {
	checkpoints, err := m.checkpoints.List(ctx)
	if err != nil {
		return 0, err
	}

	m.logger.Info("saga recovery scan started", "checkpoints", len(checkpoints))

	recovered := 0
	var firstErr error
	for _, checkpoint := range checkpoints {
		if checkpoint == nil || checkpoint.State.IsTerminal() {
			continue
		}

		definition, ok := definitions[checkpoint.DefinitionName]
		if !ok {
			m.logger.Warn("skipping recovery, definition not found",
				"saga_id", checkpoint.SagaID,
				"definition", checkpoint.DefinitionName,
			)
			continue
		}

		var input any
		if inputProvider != nil {
			input = inputProvider(checkpoint.SagaID)
		}

		instance, err := m.orchestrator.ResumeFromCheckpoint(ctx, definition, checkpoint, input)
		if err != nil {
			m.logger.Warn("saga recovery failed", "saga_id", checkpoint.SagaID, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if instance != nil {
			_ = m.checkpoints.Save(ctx, Snapshot(instance))
		}

		recovered++
		m.logger.Info("saga recovered from checkpoint",
			"saga_id", checkpoint.SagaID,
			"state", checkpoint.State.String(),
		)
	}

	m.logger.Info("saga recovery scan completed", "recovered", recovered)
	return recovered, firstErr
}

// CleanupManager handles WAL retention and checkpoint cleanup.
type CleanupManager struct {
	wal         *BadgerWAL
	checkpoints CheckpointStore
	isTerminal  func(sagaID string) bool
	logger      RecoveryLogger

	mu      sync.Mutex
	running bool
}

// NewCleanupManager creates a cleanup manager.
func NewCleanupManager(
	wal *BadgerWAL,
	checkpoints CheckpointStore,
	isTerminal func(sagaID string) bool,
	logger RecoveryLogger,
) *CleanupManager {
	if logger == nil {
		logger = &nopRecoveryLogger{}
	}
	return &CleanupManager{
		wal:         wal,
		checkpoints: checkpoints,
		isTerminal:  isTerminal,
		logger:      logger,
	}
}

// Start runs periodic cleanup until the context is cancelled.
func (m *CleanupManager) Start(ctx context.Context, interval, retention time.Duration) error {
	if m.wal == nil {
		return nil
	}
	if interval <= 0 {
		return fmt.Errorf("cleanup interval must be > 0")
	}
	if retention <= 0 {
		return fmt.Errorf("retention must be > 0")
	}

	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("cleanup manager already running")
	}
	m.running = true
	m.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				m.mu.Lock()
				m.running = false
				m.mu.Unlock()
				return
			case <-ticker.C:
				deleted, err := m.RunOnce(ctx, retention)
				if err != nil {
					m.logger.Warn("wal cleanup failed", "error", err)
					continue
				}
				if deleted > 0 {
					m.logger.Info("wal cleanup completed", "deleted_entries", deleted)
				}
			}
		}
	}()

	return nil
}

// RunOnce performs one cleanup pass.
func (m *CleanupManager) RunOnce(ctx context.Context, retention time.Duration) (int, error) {
	if m.wal == nil {
		return 0, nil
	}
	if retention <= 0 {
		return 0, fmt.Errorf("retention must be > 0")
	}

	cutoff := time.Now().UTC().Add(-retention)
	expiredBySaga := make(map[string][][]byte)

	err := m.wal.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(walKeyPrefix)
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
			sagaID := parseSagaIDFromWALKey(key)
			if sagaID == "" {
				continue
			}
			if !m.isSagaTerminal(sagaID) {
				continue
			}

			var entry WALEntry
			if err := item.Value(func(v []byte) error {
				return json.Unmarshal(v, &entry)
			}); err != nil {
				return err
			}
			if entry.Timestamp.IsZero() || entry.Timestamp.After(cutoff) {
				continue
			}

			expiredBySaga[sagaID] = append(expiredBySaga[sagaID], item.KeyCopy(nil))
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	if len(expiredBySaga) == 0 {
		return 0, nil
	}

	totalDeleted := 0
	if err := m.wal.db.Update(func(txn *badger.Txn) error {
		for _, keys := range expiredBySaga {
			for _, key := range keys {
				if err := txn.Delete(key); err != nil {
					return err
				}
				totalDeleted++
			}
		}
		return nil
	}); err != nil {
		return 0, err
	}

	if m.checkpoints != nil {
		for sagaID := range expiredBySaga {
			if m.isSagaTerminal(sagaID) {
				_ = m.checkpoints.Delete(ctx, sagaID)
			}
		}
	}

	return totalDeleted, nil
}

func (m *CleanupManager) isSagaTerminal(sagaID string) bool {
	if m.isTerminal == nil {
		return true
	}
	return m.isTerminal(sagaID)
}

func parseSagaIDFromWALKey(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) < 3 || parts[0] != strings.TrimSuffix(walKeyPrefix, ":") {
		return ""
	}
	return parts[1]
}
