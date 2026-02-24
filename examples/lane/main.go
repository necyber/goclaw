// Example: Lane Queue System
//
// This example demonstrates how to use the Lane Queue System for task scheduling.
package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/goclaw/goclaw/pkg/lane"
)

func main() {
	fmt.Println("=== Lane Queue System Example ===")

	// Example 1: Basic Lane Usage
	basicExample()

	// Example 2: Multiple Lanes with Manager
	managerExample()

	// Example 3: Backpressure Strategies
	backpressureExample()

	// Example 4: Rate Limiting
	rateLimitExample()
}

func basicExample() {
	fmt.Println("--- Example 1: Basic Lane Usage ---")

	// Create a lane with capacity 10 and max 2 concurrent workers
	config := &lane.Config{
		Name:           "cpu",
		Capacity:       10,
		MaxConcurrency: 2,
		Backpressure:   lane.Block,
	}

	l, err := lane.New(config)
	if err != nil {
		log.Fatalf("Failed to create lane: %v", err)
	}
	defer l.Close(context.Background())

	// Start the lane's main loop
	l.Run()

	// Submit some tasks
	var counter atomic.Int32
	for i := 0; i < 5; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		task := lane.NewTaskFunc(taskID, "cpu", 1, func(ctx context.Context) error {
			fmt.Printf("  Executing %s\n", taskID)
			time.Sleep(100 * time.Millisecond)
			counter.Add(1)
			return nil
		})

		if err := l.Submit(context.Background(), task); err != nil {
			log.Printf("Failed to submit task: %v", err)
		}
	}

	// Wait for tasks to complete
	time.Sleep(500 * time.Millisecond)

	// Print stats
	stats := l.Stats()
	fmt.Printf("  Completed: %d, Failed: %d\n", stats.Completed, stats.Failed)
	fmt.Println()
}

func managerExample() {
	fmt.Println("--- Example 2: Multiple Lanes with Manager ---")

	// Create a manager
	manager := lane.NewManager()
	defer manager.Close(context.Background())

	// Register CPU lane
	cpuConfig := &lane.Config{
		Name:           "cpu",
		Capacity:       10,
		MaxConcurrency: 4,
		Backpressure:   lane.Block,
	}
	if _, err := manager.Register(cpuConfig); err != nil {
		log.Fatalf("Failed to register CPU lane: %v", err)
	}

	// Register IO lane
	ioConfig := &lane.Config{
		Name:           "io",
		Capacity:       20,
		MaxConcurrency: 8,
		Backpressure:   lane.Block,
	}
	if _, err := manager.Register(ioConfig); err != nil {
		log.Fatalf("Failed to register IO lane: %v", err)
	}

	// Submit tasks to different lanes
	tasks := []struct {
		id   string
		lane string
	}{
		{"read-file", "io"},
		{"parse-data", "cpu"},
		{"write-file", "io"},
		{"compute-1", "cpu"},
		{"compute-2", "cpu"},
	}

	for _, tt := range tasks {
		task := lane.NewTaskFunc(tt.id, tt.lane, 1, func(ctx context.Context) error {
			fmt.Printf("  Executing %s in %s lane\n", tt.id, tt.lane)
			time.Sleep(50 * time.Millisecond)
			return nil
		})

		if err := manager.Submit(context.Background(), task); err != nil {
			log.Printf("Failed to submit task: %v", err)
		}
	}

	// Wait for tasks to complete
	time.Sleep(300 * time.Millisecond)

	// Print stats for all lanes
	stats := manager.GetStats()
	for name, s := range stats {
		fmt.Printf("  %s lane: Completed=%d, Pending=%d\n", name, s.Completed, s.Pending)
	}
	fmt.Println()
}

func backpressureExample() {
	fmt.Println("--- Example 3: Backpressure Strategies ---")

	// Create a lane with Drop strategy
	dropConfig := &lane.Config{
		Name:           "limited",
		Capacity:       2,
		MaxConcurrency: 1,
		Backpressure:   lane.Drop,
	}

	l, err := lane.New(dropConfig)
	if err != nil {
		log.Fatalf("Failed to create lane: %v", err)
	}
	defer l.Close(context.Background())
	l.Run()

	// Fill the queue with slow tasks
	for i := 0; i < 3; i++ {
		task := lane.NewTaskFunc(fmt.Sprintf("slow-%d", i), "limited", 1, func(ctx context.Context) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		})
		l.Submit(context.Background(), task)
	}

	// Try to submit one more task (should be dropped)
	task := lane.NewTaskFunc("extra", "limited", 1, func(ctx context.Context) error {
		return nil
	})

	err = l.Submit(context.Background(), task)
	if lane.IsTaskDroppedError(err) {
		fmt.Println("  Task was dropped due to backpressure (expected)")
	} else if err != nil {
		fmt.Printf("  Unexpected error: %v\n", err)
	}

	stats := l.Stats()
	fmt.Printf("  Dropped tasks: %d\n", stats.Dropped)
	fmt.Println()
}

func rateLimitExample() {
	fmt.Println("--- Example 4: Rate Limiting ---")

	// Create a lane with rate limiting (2 tasks per second)
	config := &lane.Config{
		Name:           "rated",
		Capacity:       10,
		MaxConcurrency: 2,
		Backpressure:   lane.Block,
		RateLimit:      2, // 2 tasks per second
	}

	l, err := lane.New(config)
	if err != nil {
		log.Fatalf("Failed to create lane: %v", err)
	}
	defer l.Close(context.Background())
	l.Run()

	// Submit tasks quickly
	start := time.Now()
	for i := 0; i < 5; i++ {
		task := lane.NewTaskFunc(fmt.Sprintf("rate-%d", i), "rated", 1, func(ctx context.Context) error {
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := l.Submit(ctx, task)
		cancel()

		if err != nil {
			fmt.Printf("  Task %d failed: %v\n", i, err)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("  Submitted 5 tasks with rate limit in %v\n", elapsed)
	fmt.Println()
}
