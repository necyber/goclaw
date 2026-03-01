package memory

import (
	"context"
	"math"
	"sync"
	"time"
)

// DecayManager implements the FSRS-6 memory decay algorithm.
// It runs a background goroutine to periodically update memory strengths.
type DecayManager struct {
	mu               sync.Mutex
	threshold        float64
	defaultStability float64
	interval         time.Duration
	cancel           context.CancelFunc
	done             chan struct{}

	// Metrics
	totalDecayed   int64
	totalForgotten int64
}

// NewDecayManager creates a new FSRS-6 decay manager.
func NewDecayManager(threshold, defaultStability float64, interval time.Duration) *DecayManager {
	return &DecayManager{
		threshold:        threshold,
		defaultStability: defaultStability,
		interval:         interval,
	}
}

// UpdateStrength applies the FSRS-6 decay formula: S' = S * e^(-t/τ)
// where t is hours since last review and τ is the stability parameter.
func (d *DecayManager) UpdateStrength(entry *MemoryEntry) {
	now := time.Now()
	if entry.Stability <= 0 {
		entry.Stability = d.defaultStability
	}

	if entry.LastReview.IsZero() {
		entry.LastReview = now
		return
	}

	elapsed := now.Sub(entry.LastReview).Hours()
	if elapsed <= 0 {
		entry.LastReview = now
		return
	}

	entry.Strength *= math.Exp(-elapsed / entry.Stability)
	if entry.Strength < 0 {
		entry.Strength = 0
	}
	entry.LastReview = now
}

// BoostStrength resets strength to 1.0 and increases stability.
func (d *DecayManager) BoostStrength(entry *MemoryEntry) {
	entry.Strength = 1.0
	entry.LastReview = time.Now()
	// Increase stability by 50% on each successful retrieval
	entry.Stability *= 1.5
}

// InitEntry sets initial decay parameters for a new entry.
func (d *DecayManager) InitEntry(entry *MemoryEntry) {
	entry.Strength = 1.0
	entry.Stability = d.defaultStability
	entry.LastReview = time.Now()
}

// StartDecayLoop starts the background decay goroutine.
// The provided callback is called with entries that need updating.
func (d *DecayManager) StartDecayLoop(parentCtx context.Context, processFunc func(ctx context.Context) error) {
	d.mu.Lock()
	if d.cancel != nil {
		d.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(parentCtx)
	done := make(chan struct{})
	d.cancel = cancel
	d.done = done
	d.mu.Unlock()

	go func() {
		defer func() {
			close(done)
			d.mu.Lock()
			if d.done == done {
				d.done = nil
			}
			d.mu.Unlock()
		}()
		ticker := time.NewTicker(d.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := processFunc(ctx); err != nil {
					// Log error but continue
					continue
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// DecayEntries applies decay to a batch of entries and returns those below threshold.
func (d *DecayManager) DecayEntries(entries []*MemoryEntry) (updated []*MemoryEntry, forgotten []string) {
	// Pre-allocate with estimated capacity
	updated = make([]*MemoryEntry, 0, len(entries))
	forgotten = make([]string, 0, len(entries)/10)

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, entry := range entries {
		d.UpdateStrength(entry)
		if entry.Strength < d.threshold {
			forgotten = append(forgotten, entry.ID)
			d.totalForgotten++
		} else {
			updated = append(updated, entry)
			d.totalDecayed++
		}
	}
	return updated, forgotten
}

// Stop gracefully stops the decay loop.
func (d *DecayManager) Stop() {
	d.mu.Lock()
	cancel := d.cancel
	done := d.done
	d.cancel = nil
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// Stats returns decay metrics.
func (d *DecayManager) Stats() (decayed, forgotten int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.totalDecayed, d.totalForgotten
}
