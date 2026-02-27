package saga

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestMemorySagaStoreCRUD(t *testing.T) {
	store := NewMemorySagaStore()
	instance := &SagaInstance{
		ID:             "mem-1",
		DefinitionName: "demo",
		State:          SagaStateRunning,
		StepResults:    map[string]any{"a": "ok"},
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := store.Save(context.Background(), instance); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Get(context.Background(), "mem-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.ID != "mem-1" || loaded.State != SagaStateRunning {
		t.Fatalf("unexpected loaded instance: %#v", loaded)
	}

	list, total, err := store.List(context.Background(), SagaListFilter{State: "running"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("unexpected list result: total=%d len=%d", total, len(list))
	}

	if err := store.Delete(context.Background(), "mem-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Get(context.Background(), "mem-1"); err == nil {
		t.Fatal("expected ErrSagaNotFound after delete")
	}
}

func TestBadgerSagaStoreCRUDAndQuery(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewBadgerSagaStore(db)
	if err != nil {
		t.Fatalf("NewBadgerSagaStore() error = %v", err)
	}

	instances := []*SagaInstance{
		{
			ID:             "s1",
			DefinitionName: "demo",
			State:          SagaStateRunning,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		},
		{
			ID:             "s2",
			DefinitionName: "demo",
			State:          SagaStateCompleted,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		},
		{
			ID:             "s3",
			DefinitionName: "demo",
			State:          SagaStateRunning,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		},
	}
	for _, instance := range instances {
		if err := store.Save(context.Background(), instance); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Verify key format "saga:{id}" is used.
	err = db.View(func(txn *badger.Txn) error {
		_, getErr := txn.Get([]byte("saga:s1"))
		return getErr
	})
	if err != nil {
		t.Fatalf("expected saga data key to exist: %v", err)
	}

	running, total, err := store.List(context.Background(), SagaListFilter{
		State:  "running",
		Limit:  1,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("List(running) error = %v", err)
	}
	if total != 2 || len(running) != 1 {
		t.Fatalf("unexpected running list result total=%d len=%d", total, len(running))
	}

	page2, _, err := store.List(context.Background(), SagaListFilter{
		State:  "running",
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("List(page2) error = %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("expected one record in second page, got %d", len(page2))
	}

	if err := store.Delete(context.Background(), "s2"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Get(context.Background(), "s2"); err == nil {
		t.Fatal("expected not found after delete")
	}
}
