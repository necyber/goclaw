package memory

import (
	"context"
	"math"
	"sync/atomic"
	"testing"
	"time"
)

func TestDecayManager_UpdateStrength(t *testing.T) {
	dm := NewDecayManager(0.1, 24.0, time.Hour)

	entry := &MemoryEntry{
		Strength:   1.0,
		Stability:  24.0,
		LastReview: time.Now().Add(-24 * time.Hour),
	}

	dm.UpdateStrength(entry)

	// After 24h with stability 24h: S' = 1.0 * e^(-24/24) = 1/e ≈ 0.368
	expected := 1.0 / math.E
	if math.Abs(entry.Strength-expected) > 0.01 {
		t.Errorf("expected strength ~%f, got %f", expected, entry.Strength)
	}
}

func TestDecayManager_BoostStrength(t *testing.T) {
	dm := NewDecayManager(0.1, 24.0, time.Hour)

	entry := &MemoryEntry{
		Strength:   0.5,
		Stability:  24.0,
		LastReview: time.Now().Add(-12 * time.Hour),
	}

	dm.BoostStrength(entry)

	if entry.Strength != 1.0 {
		t.Errorf("expected strength 1.0, got %f", entry.Strength)
	}
	if entry.Stability != 36.0 {
		t.Errorf("expected stability 36.0, got %f", entry.Stability)
	}
	if time.Since(entry.LastReview) > time.Second {
		t.Error("expected LastReview to be updated to now")
	}
}

func TestDecayManager_InitEntry(t *testing.T) {
	dm := NewDecayManager(0.1, 24.0, time.Hour)

	entry := &MemoryEntry{}
	dm.InitEntry(entry)

	if entry.Strength != 1.0 {
		t.Errorf("expected strength 1.0, got %f", entry.Strength)
	}
	if entry.Stability != 24.0 {
		t.Errorf("expected stability 24.0, got %f", entry.Stability)
	}
}

func TestDecayManager_DecayEntries(t *testing.T) {
	dm := NewDecayManager(0.1, 24.0, time.Hour)

	entries := []*MemoryEntry{
		{ID: "strong", Strength: 1.0, Stability: 24.0, LastReview: time.Now()},
		{ID: "weak", Strength: 0.05, Stability: 24.0, LastReview: time.Now().Add(-48 * time.Hour)},
	}

	updated, forgotten := dm.DecayEntries(entries)

	if len(forgotten) == 0 {
		t.Error("expected at least one forgotten entry")
	}
	if len(updated) == 0 {
		t.Error("expected at least one updated entry")
	}

	// The strong entry should survive
	foundStrong := false
	for _, e := range updated {
		if e.ID == "strong" {
			foundStrong = true
		}
	}
	if !foundStrong {
		t.Error("expected 'strong' entry to survive decay")
	}
}

func TestDecayManager_HighStabilitySlowDecay(t *testing.T) {
	dm := NewDecayManager(0.1, 24.0, time.Hour)

	entry := &MemoryEntry{
		Strength:   1.0,
		Stability:  1000.0, // Very high stability
		LastReview: time.Now().Add(-24 * time.Hour),
	}

	dm.UpdateStrength(entry)

	// With stability 1000h and 24h elapsed: S' = e^(-24/1000) ≈ 0.976
	if entry.Strength < 0.95 {
		t.Errorf("expected slow decay with high stability, got %f", entry.Strength)
	}
}

func TestDecayManager_StartStopRestart(t *testing.T) {
	dm := NewDecayManager(0.1, 24.0, 5*time.Millisecond)
	var calls atomic.Int64
	process := func(ctx context.Context) error {
		calls.Add(1)
		return nil
	}

	ctx := context.Background()
	dm.StartDecayLoop(ctx, process)
	time.Sleep(15 * time.Millisecond)
	dm.Stop()
	dm.Stop() // idempotent stop

	dm.StartDecayLoop(ctx, process)
	time.Sleep(15 * time.Millisecond)
	dm.Stop()

	if calls.Load() == 0 {
		t.Fatalf("expected decay process to run at least once")
	}
}

func TestDecayManager_UpdateStrength_UsesNonOverlappingTimeWindows(t *testing.T) {
	dm := NewDecayManager(0.1, 24.0, time.Hour)

	entry := &MemoryEntry{
		Strength:   1.0,
		Stability:  24.0,
		LastReview: time.Now().Add(-24 * time.Hour),
	}

	dm.UpdateStrength(entry)
	firstStrength := entry.Strength
	firstReview := entry.LastReview

	dm.UpdateStrength(entry)
	secondStrength := entry.Strength

	if secondStrength < firstStrength*0.99 {
		t.Fatalf("expected second decay update to avoid reapplying full prior interval: first=%f second=%f", firstStrength, secondStrength)
	}
	if entry.LastReview.Before(firstReview) {
		t.Fatalf("expected last review timestamp to stay monotonic")
	}
}
