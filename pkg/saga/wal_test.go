package saga

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func openTestBadger(t testing.TB) *badger.DB {
	t.Helper()
	opts := badger.DefaultOptions(t.TempDir())
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatalf("open badger: %v", err)
	}
	return db
}

func TestBadgerWALAppendAndListSync(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	wal, err := NewBadgerWAL(db, WALOptions{WriteMode: WALWriteModeSync})
	if err != nil {
		t.Fatalf("NewBadgerWAL() error = %v", err)
	}

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := wal.Append(ctx, WALEntry{
			SagaID: "saga-sync",
			StepID: fmt.Sprintf("step-%d", i),
			Type:   WALEntryTypeStepStarted,
		})
		if err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	entries, err := wal.List(ctx, "saga-sync")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	for i, entry := range entries {
		wantSeq := uint64(i + 1)
		if entry.Sequence != wantSeq {
			t.Fatalf("entry[%d] sequence = %d, want %d", i, entry.Sequence, wantSeq)
		}
		key := walEntryKey("saga-sync", entry.Sequence)
		parsed, err := parseSequenceFromWALKey(key)
		if err != nil {
			t.Fatalf("parseSequenceFromWALKey() error = %v", err)
		}
		if parsed != entry.Sequence {
			t.Fatalf("parsed sequence = %d, want %d", parsed, entry.Sequence)
		}
	}
}

func TestBadgerWALAppendAndListAsync(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	wal, err := NewBadgerWAL(db, WALOptions{
		WriteMode:      WALWriteModeAsync,
		AsyncQueueSize: 16,
	})
	if err != nil {
		t.Fatalf("NewBadgerWAL() error = %v", err)
	}
	t.Cleanup(func() { _ = wal.Close() })

	ctx := context.Background()
	for i := 0; i < 10; i++ {
		if _, err := wal.Append(ctx, WALEntry{
			SagaID: "saga-async",
			StepID: fmt.Sprintf("step-%d", i),
			Type:   WALEntryTypeStepCompleted,
		}); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		entries, err := wal.List(ctx, "saga-async")
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(entries) == 10 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting async entries, got %d", len(entries))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestBadgerWALSequenceIsMonotonicPerSaga(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	wal, err := NewBadgerWAL(db, WALOptions{WriteMode: WALWriteModeSync})
	if err != nil {
		t.Fatalf("NewBadgerWAL() error = %v", err)
	}

	ctx := context.Background()
	seqA1, _ := wal.Append(ctx, WALEntry{SagaID: "a", Type: WALEntryTypeStepStarted})
	seqA2, _ := wal.Append(ctx, WALEntry{SagaID: "a", Type: WALEntryTypeStepCompleted})
	seqB1, _ := wal.Append(ctx, WALEntry{SagaID: "b", Type: WALEntryTypeStepStarted})

	if seqA1 != 1 || seqA2 != 2 {
		t.Fatalf("unexpected saga a sequence values: %d, %d", seqA1, seqA2)
	}
	if seqB1 != 1 {
		t.Fatalf("unexpected saga b sequence value: %d", seqB1)
	}
}

func TestBadgerWALDeleteBySagaID(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	wal, err := NewBadgerWAL(db, WALOptions{WriteMode: WALWriteModeSync})
	if err != nil {
		t.Fatalf("NewBadgerWAL() error = %v", err)
	}

	ctx := context.Background()
	_, _ = wal.Append(ctx, WALEntry{SagaID: "s1", Type: WALEntryTypeStepStarted})
	_, _ = wal.Append(ctx, WALEntry{SagaID: "s1", Type: WALEntryTypeStepCompleted})
	_, _ = wal.Append(ctx, WALEntry{SagaID: "s2", Type: WALEntryTypeStepStarted})

	if err := wal.DeleteBySagaID(ctx, "s1"); err != nil {
		t.Fatalf("DeleteBySagaID() error = %v", err)
	}

	s1, err := wal.List(ctx, "s1")
	if err != nil {
		t.Fatalf("List(s1) error = %v", err)
	}
	if len(s1) != 0 {
		t.Fatalf("expected s1 entries deleted, got %d", len(s1))
	}

	s2, err := wal.List(ctx, "s2")
	if err != nil {
		t.Fatalf("List(s2) error = %v", err)
	}
	if len(s2) != 1 {
		t.Fatalf("expected s2 entries to remain, got %d", len(s2))
	}
}

func BenchmarkBadgerWALAppendSync(b *testing.B) {
	db := openTestBadger(b)
	defer db.Close()

	wal, err := NewBadgerWAL(db, WALOptions{WriteMode: WALWriteModeSync})
	if err != nil {
		b.Fatalf("NewBadgerWAL() error = %v", err)
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := wal.Append(ctx, WALEntry{
			SagaID: "bench-saga",
			StepID: "bench-step",
			Type:   WALEntryTypeStepCompleted,
			Data:   []byte("ok"),
		}); err != nil {
			b.Fatalf("Append() error = %v", err)
		}
	}
}
