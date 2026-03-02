package saga

import (
	"context"
	"errors"
	"testing"
)

func TestBadgerSagaDefinitionStoreCRUD(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewBadgerSagaDefinitionStore(db)
	if err != nil {
		t.Fatalf("NewBadgerSagaDefinitionStore() error = %v", err)
	}

	snapshot := &DefinitionSnapshot{
		Name:          "order-saga",
		Policy:        "manual",
		TimeoutMS:     2000,
		StepTimeoutMS: 500,
		Metadata:      map[string]string{"source": "test"},
		Input:         map[string]any{"order_id": "o-1"},
		Steps: []DefinitionStepSnapshot{
			{ID: "a", EnableCompensation: true},
			{ID: "b", DependsOn: []string{"a"}, ShouldFail: true},
		},
	}

	if err := store.Save(context.Background(), "saga-1", snapshot); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load(context.Background(), "saga-1")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Name != snapshot.Name || loaded.Policy != snapshot.Policy {
		t.Fatalf("loaded snapshot mismatch: %#v", loaded)
	}
	if len(loaded.Steps) != 2 || loaded.Steps[1].ID != "b" {
		t.Fatalf("unexpected loaded steps: %#v", loaded.Steps)
	}

	all, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected one snapshot in list, got %d", len(all))
	}
	if _, ok := all["saga-1"]; !ok {
		t.Fatalf("expected saga-1 in list, got keys %#v", all)
	}

	if err := store.Delete(context.Background(), "saga-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err = store.Load(context.Background(), "saga-1")
	if !errors.Is(err, ErrSagaDefinitionNotFound) {
		t.Fatalf("expected ErrSagaDefinitionNotFound, got %v", err)
	}
}

func TestBadgerSagaDefinitionStoreValidationAndMissing(t *testing.T) {
	db := openTestBadger(t)
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewBadgerSagaDefinitionStore(db)
	if err != nil {
		t.Fatalf("NewBadgerSagaDefinitionStore() error = %v", err)
	}

	snapshot := &DefinitionSnapshot{
		Name:  "validation",
		Steps: []DefinitionStepSnapshot{{ID: "a"}},
	}

	if err := store.Save(context.Background(), "", snapshot); err == nil {
		t.Fatal("expected empty saga id save error")
	}
	if err := store.Save(context.Background(), "saga-validation", nil); err == nil {
		t.Fatal("expected nil snapshot save error")
	}
	if _, err := store.Load(context.Background(), ""); err == nil {
		t.Fatal("expected empty saga id load error")
	}
	if _, err := store.Load(context.Background(), "missing"); !errors.Is(err, ErrSagaDefinitionNotFound) {
		t.Fatalf("expected ErrSagaDefinitionNotFound for missing load, got %v", err)
	}
	if err := store.Delete(context.Background(), ""); err == nil {
		t.Fatal("expected empty saga id delete error")
	}
}

func TestBuildDefinitionFromSnapshot(t *testing.T) {
	snapshot := &DefinitionSnapshot{
		Name:          "snapshot-build",
		Policy:        "skip",
		TimeoutMS:     1000,
		StepTimeoutMS: 200,
		Input:         map[string]any{"user_id": "u-1"},
		Steps: []DefinitionStepSnapshot{
			{ID: "a", EnableCompensation: true},
			{ID: "b", DependsOn: []string{"a"}, SkipCompensation: true},
		},
	}

	def, input, err := BuildDefinitionFromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("BuildDefinitionFromSnapshot() error = %v", err)
	}
	if def.Name != "snapshot-build" {
		t.Fatalf("unexpected definition name: %s", def.Name)
	}
	if def.Policy != SkipCompensate {
		t.Fatalf("expected skip policy, got %v", def.Policy)
	}
	inputMap, ok := input.(map[string]any)
	if !ok {
		t.Fatalf("expected map input, got %T", input)
	}
	if inputMap["user_id"] != "u-1" {
		t.Fatalf("unexpected input payload: %#v", inputMap)
	}
	if len(def.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(def.Steps))
	}
}
