// Package main demonstrates how to use the Goclaw engine to execute a workflow.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/engine"
)

func main() {
	// Build a minimal config.
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "engine-example",
			Environment: "development",
		},
		Server: config.ServerConfig{
			Port: 8080,
			GRPC: config.GRPCConfig{Port: 9090, MaxConcurrentStreams: 100},
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents: 4,
			Queue:     config.QueueConfig{Type: "memory", Size: 100},
			Scheduler: config.SchedulerConfig{Type: "round_robin"},
		},
		Storage: config.StorageConfig{Type: "memory"},
	}

	// Create and start the engine.
	eng, err := engine.New(cfg, nil)
	if err != nil {
		log.Fatalf("failed to create engine: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := eng.Start(ctx); err != nil {
		log.Fatalf("failed to start engine: %v", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		eng.Stop(stopCtx)
	}()

	fmt.Println("Engine started. Submitting workflow...")

	// Define a 3-layer DAG:
	//
	//   fetch ──┐
	//           ├──▶ process ──▶ report
	//   enrich ─┘
	wf := &engine.Workflow{
		ID: "example-workflow",
		Tasks: []*dag.Task{
			{ID: "fetch", Name: "Fetch Data", Agent: "fetcher"},
			{ID: "enrich", Name: "Enrich Data", Agent: "enricher"},
			{ID: "process", Name: "Process Data", Agent: "processor", Deps: []string{"fetch", "enrich"}},
			{ID: "report", Name: "Generate Report", Agent: "reporter", Deps: []string{"process"}},
		},
		TaskFns: map[string]func(context.Context) error{
			"fetch": func(ctx context.Context) error {
				fmt.Println("  [fetch]   fetching data...")
				time.Sleep(100 * time.Millisecond)
				fmt.Println("  [fetch]   done")
				return nil
			},
			"enrich": func(ctx context.Context) error {
				fmt.Println("  [enrich]  enriching data...")
				time.Sleep(80 * time.Millisecond)
				fmt.Println("  [enrich]  done")
				return nil
			},
			"process": func(ctx context.Context) error {
				fmt.Println("  [process] processing data...")
				time.Sleep(150 * time.Millisecond)
				fmt.Println("  [process] done")
				return nil
			},
			"report": func(ctx context.Context) error {
				fmt.Println("  [report]  generating report...")
				time.Sleep(50 * time.Millisecond)
				fmt.Println("  [report]  done")
				return nil
			},
		},
	}

	result, err := eng.Submit(ctx, wf)
	if err != nil {
		log.Fatalf("workflow failed: %v", err)
	}

	fmt.Printf("\nWorkflow %q completed with status: %v\n", result.WorkflowID, result.Status)
	fmt.Println("\nTask results:")
	for id, r := range result.TaskResults {
		duration := r.EndedAt.Sub(r.StartedAt).Round(time.Millisecond)
		fmt.Printf("  %-10s  state=%-10s  duration=%v\n", id, r.State, duration)
	}
}
