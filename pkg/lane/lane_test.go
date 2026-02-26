package lane

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Name:           "test",
				Capacity:       100,
				MaxConcurrency: 8,
				Backpressure:   Block,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			config: &Config{
				Name:           "",
				Capacity:       100,
				MaxConcurrency: 8,
			},
			wantErr: true,
		},
		{
			name: "zero capacity",
			config: &Config{
				Name:           "test",
				Capacity:       0,
				MaxConcurrency: 8,
			},
			wantErr: true,
		},
		{
			name: "zero concurrency",
			config: &Config{
				Name:           "test",
				Capacity:       100,
				MaxConcurrency: 0,
			},
			wantErr: true,
		},
		{
			name: "redirect without target",
			config: &Config{
				Name:           "test",
				Capacity:       100,
				MaxConcurrency: 8,
				Backpressure:   Redirect,
				RedirectLane:   "",
			},
			wantErr: true,
		},
		{
			name: "negative rate limit",
			config: &Config{
				Name:           "test",
				Capacity:       100,
				MaxConcurrency: 8,
				RateLimit:      -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChannelLane_Submit(t *testing.T) {
	config := &Config{
		Name:           "test",
		Capacity:       10,
		MaxConcurrency: 2,
		Backpressure:   Block,
	}

	lane, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create lane: %v", err)
	}
	defer lane.Close(context.Background())
	lane.Run()

	var counter atomic.Int32

	// Create a task
	task := NewTaskFunc("task-1", "test", 1, func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})

	// Submit task
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = lane.Submit(ctx, task)
	if err != nil {
		t.Errorf("Submit() error = %v", err)
	}

	// Wait for task to complete
	time.Sleep(100 * time.Millisecond)

	if counter.Load() != 1 {
		t.Errorf("Expected counter to be 1, got %d", counter.Load())
	}
}

func TestChannelLane_TrySubmit(t *testing.T) {
	config := &Config{
		Name:           "test",
		Capacity:       1,
		MaxConcurrency: 1,
		Backpressure:   Block,
	}

	lane, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create lane: %v", err)
	}
	defer lane.Close(context.Background())
	lane.Run()

	// Create a slow task to fill the worker
	slowTask := NewTaskFunc("slow", "test", 1, func(ctx context.Context) error {
		time.Sleep(500 * time.Millisecond)
		return nil
	})

	// Create a normal task
	normalTask := NewTaskFunc("normal", "test", 1, func(ctx context.Context) error {
		return nil
	})

	// Submit slow task (should succeed)
	if !lane.TrySubmit(slowTask) {
		t.Error("First TrySubmit should succeed")
	}

	// Try to submit another task while worker is busy and queue is full
	// This might succeed or fail depending on timing
	_ = lane.TrySubmit(normalTask)
}

func TestChannelLane_BackpressureDrop(t *testing.T) {
	config := &Config{
		Name:           "test",
		Capacity:       1,
		MaxConcurrency: 1,
		Backpressure:   Drop,
	}

	lane, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create lane: %v", err)
	}
	defer lane.Close(context.Background())
	lane.Run()

	// Create a slow task to fill the worker and queue
	slowTask := NewTaskFunc("slow", "test", 1, func(ctx context.Context) error {
		time.Sleep(500 * time.Millisecond)
		return nil
	})

	// Fill the queue
	for i := 0; i < 10; i++ {
		task := NewTaskFunc(fmt.Sprintf("task-%d", i), "test", 1, func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		lane.Submit(context.Background(), task)
	}

	// Try to submit to full queue with Drop strategy
	err = lane.Submit(context.Background(), slowTask)
	if err == nil {
		t.Error("Expected error for dropped task")
	}

	if !IsTaskDroppedError(err) {
		t.Errorf("Expected TaskDroppedError, got %T", err)
	}
}

func TestChannelLane_Stats(t *testing.T) {
	config := &Config{
		Name:           "test",
		Capacity:       10,
		MaxConcurrency: 2,
		Backpressure:   Block,
	}

	lane, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create lane: %v", err)
	}
	defer lane.Close(context.Background())
	lane.Run()

	// Initial stats
	stats := lane.Stats()
	if stats.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", stats.Name)
	}
	if stats.Capacity != 10 {
		t.Errorf("Expected capacity 10, got %d", stats.Capacity)
	}
	if stats.MaxConcurrency != 2 {
		t.Errorf("Expected max concurrency 2, got %d", stats.MaxConcurrency)
	}

	// Submit a task and check stats
	task := NewTaskFunc("task-1", "test", 1, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	lane.Submit(context.Background(), task)

	// Wait a bit for task to be processed
	time.Sleep(100 * time.Millisecond)

	stats = lane.Stats()
	if stats.Completed != 1 {
		t.Errorf("Expected 1 completed task, got %d", stats.Completed)
	}
}

func TestChannelLane_Close(t *testing.T) {
	config := &Config{
		Name:           "test",
		Capacity:       10,
		MaxConcurrency: 2,
		Backpressure:   Block,
	}

	lane, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create lane: %v", err)
	}

	lane.Run()

	// Submit some tasks
	for i := 0; i < 5; i++ {
		task := NewTaskFunc(fmt.Sprintf("task-%d", i), "test", 1, func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		lane.Submit(context.Background(), task)
	}

	// Close the lane
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = lane.Close(ctx)
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify lane is closed
	if !lane.IsClosed() {
		t.Error("Expected lane to be closed")
	}

	// Try to submit to closed lane
	task := NewTaskFunc("task", "test", 1, func(ctx context.Context) error {
		return nil
	})
	err = lane.Submit(context.Background(), task)
	if !IsLaneClosedError(err) {
		t.Errorf("Expected LaneClosedError, got %T", err)
	}
}

func TestManager(t *testing.T) {
	manager := NewManager()
	defer manager.Close(context.Background())

	// Register a lane
	config := &Config{
		Name:           "cpu",
		Capacity:       100,
		MaxConcurrency: 8,
		Backpressure:   Block,
	}

	lane, err := manager.Register(config)
	if err != nil {
		t.Fatalf("Failed to register lane: %v", err)
	}

	if lane.Name() != "cpu" {
		t.Errorf("Expected lane name 'cpu', got '%s'", lane.Name())
	}

	// Try to register duplicate
	_, err = manager.Register(config)
	if !IsDuplicateLane(err) {
		t.Errorf("Expected DuplicateLaneError, got %T", err)
	}

	// Get lane
	gotLane, err := manager.GetLane("cpu")
	if err != nil {
		t.Errorf("GetLane() error = %v", err)
	}
	if gotLane.Name() != "cpu" {
		t.Errorf("Expected lane name 'cpu', got '%s'", gotLane.Name())
	}

	// Get non-existent lane
	_, err = manager.GetLane("memory")
	if !IsLaneNotFoundError(err) {
		t.Errorf("Expected LaneNotFoundError, got %T", err)
	}

	// Check HasLane
	if !manager.HasLane("cpu") {
		t.Error("Expected HasLane('cpu') to be true")
	}
	if manager.HasLane("memory") {
		t.Error("Expected HasLane('memory') to be false")
	}

	// Get lane names
	names := manager.LaneNames()
	if len(names) != 1 || names[0] != "cpu" {
		t.Errorf("Expected names ['cpu'], got %v", names)
	}
}

func TestManager_Submit(t *testing.T) {
	manager := NewManager()
	defer manager.Close(context.Background())

	// Register a lane
	config := &Config{
		Name:           "cpu",
		Capacity:       100,
		MaxConcurrency: 8,
		Backpressure:   Block,
	}

	_, err := manager.Register(config)
	if err != nil {
		t.Fatalf("Failed to register lane: %v", err)
	}

	var counter atomic.Int32

	// Create and submit a task
	task := NewTaskFunc("task-1", "cpu", 1, func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})

	err = manager.Submit(context.Background(), task)
	if err != nil {
		t.Errorf("Submit() error = %v", err)
	}

	// Wait for task to complete
	time.Sleep(100 * time.Millisecond)

	if counter.Load() != 1 {
		t.Errorf("Expected counter to be 1, got %d", counter.Load())
	}
}

func TestManager_GetStats(t *testing.T) {
	manager := NewManager()
	defer manager.Close(context.Background())

	// Register multiple lanes
	configs := []*Config{
		{
			Name:           "cpu",
			Capacity:       100,
			MaxConcurrency: 8,
			Backpressure:   Block,
		},
		{
			Name:           "io",
			Capacity:       50,
			MaxConcurrency: 4,
			Backpressure:   Block,
		},
	}

	for _, config := range configs {
		_, err := manager.Register(config)
		if err != nil {
			t.Fatalf("Failed to register lane: %v", err)
		}
	}

	// Get stats
	stats := manager.GetStats()
	if len(stats) != 2 {
		t.Errorf("Expected 2 stats, got %d", len(stats))
	}

	if _, ok := stats["cpu"]; !ok {
		t.Error("Expected stats for 'cpu' lane")
	}
	if _, ok := stats["io"]; !ok {
		t.Error("Expected stats for 'io' lane")
	}
}

func TestTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10, 5) // 10 tokens/sec, capacity 5

	// Should be able to consume immediately (full bucket)
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Errorf("Expected Allow() to return true on iteration %d", i)
		}
	}

	// Bucket should be empty now
	if tb.Allow() {
		t.Error("Expected Allow() to return false when bucket is empty")
	}

	// Wait for tokens to replenish
	time.Sleep(200 * time.Millisecond)

	// Should have ~2 tokens now
	if !tb.Allow() {
		t.Error("Expected Allow() to return true after waiting")
	}
}

func TestPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue()

	// Push tasks with different priorities
	tasks := []*TaskFunc{
		NewTaskFunc("low", "test", 1, nil),
		NewTaskFunc("high", "test", 10, nil),
		NewTaskFunc("medium", "test", 5, nil),
	}

	for _, task := range tasks {
		pq.Push(task)
	}

	if pq.Len() != 3 {
		t.Errorf("Expected length 3, got %d", pq.Len())
	}

	// Pop should return highest priority first
	task := pq.Pop()
	if task.ID() != "high" {
		t.Errorf("Expected 'high', got '%s'", task.ID())
	}

	task = pq.Pop()
	if task.ID() != "medium" {
		t.Errorf("Expected 'medium', got '%s'", task.ID())
	}

	task = pq.Pop()
	if task.ID() != "low" {
		t.Errorf("Expected 'low', got '%s'", task.ID())
	}

	if !pq.IsEmpty() {
		t.Error("Expected queue to be empty")
	}
}

func TestWorkerPool(t *testing.T) {
	var counter atomic.Int32

	wp := NewWorkerPool(2, func(task Task) {
		counter.Add(1)
		time.Sleep(10 * time.Millisecond)
	})

	wp.Start()

	// Submit tasks
	for i := 0; i < 10; i++ {
		task := NewTaskFunc(fmt.Sprintf("task-%d", i), "test", 1, nil)
		wp.Submit(task)
	}

	// Wait for tasks to complete
	time.Sleep(200 * time.Millisecond)

	wp.Stop()

	if counter.Load() != 10 {
		t.Errorf("Expected 10 tasks processed, got %d", counter.Load())
	}
}

func TestBackpressureStrategy_String(t *testing.T) {
	tests := []struct {
		strategy BackpressureStrategy
		want     string
	}{
		{Block, "block"},
		{Drop, "drop"},
		{Redirect, "redirect"},
		{BackpressureStrategy(999), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.strategy.String(); got != tt.want {
			t.Errorf("BackpressureStrategy.String() = %v, want %v", got, tt.want)
		}
	}
}

func TestStats_Utilization(t *testing.T) {
	tests := []struct {
		name  string
		stats Stats
		want  float64
	}{
		{
			name: "empty",
			stats: Stats{
				Capacity:       100,
				MaxConcurrency: 10,
				Pending:        0,
				Running:        0,
			},
			want: 0,
		},
		{
			name: "half full",
			stats: Stats{
				Capacity:       100,
				MaxConcurrency: 10,
				Pending:        50,
				Running:        5,
			},
			want: 0.5,
		},
		{
			name: "full",
			stats: Stats{
				Capacity:       100,
				MaxConcurrency: 10,
				Pending:        100,
				Running:        10,
			},
			want: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stats.Utilization()
			if got != tt.want {
				t.Errorf("Stats.Utilization() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions for error type checking
func IsDuplicateLane(err error) bool {
	_, ok := err.(*DuplicateLaneError)
	return ok
}
