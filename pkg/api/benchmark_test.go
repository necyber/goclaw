package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/goclaw/goclaw/pkg/storage/memory"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
)

// setupBenchmarkServer creates a test server for benchmarking
func setupBenchmarkServer(b *testing.B) (*httptest.Server, func()) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "benchmark",
			Environment: "test",
		},
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 18082,
			HTTP: config.HTTPConfig{
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
			},
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents: 100,
		},
	}

	log := logger.New(&logger.Config{
		Level:  logger.ErrorLevel, // Reduce logging noise in benchmarks
		Format: "json",
		Output: "stdout",
	})

	// Create and start engine
	ctx := context.Background()
	eng, err := engine.New(cfg, log, memory.NewMemoryStorage())
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	if err := eng.Start(ctx); err != nil {
		b.Fatalf("Failed to start engine: %v", err)
	}

	// Create handlers
	testHandlers := &Handlers{
		Workflow: handlers.NewWorkflowHandler(eng, log),
		Health:   handlers.NewHealthHandler(eng),
	}

	// Create router
	router := NewRouter(cfg, log, testHandlers)

	// Create test server
	server := httptest.NewServer(router)

	cleanup := func() {
		server.Close()
		eng.Stop(ctx)
	}

	return server, cleanup
}

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	server, cleanup := setupBenchmarkServer(b)
	defer cleanup()

	client := server.Client()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL + "/health")
		if err != nil {
			b.Fatalf("Failed to call health check: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("Health check status = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	}
}

// BenchmarkReadyCheck benchmarks the readiness check endpoint
func BenchmarkReadyCheck(b *testing.B) {
	server, cleanup := setupBenchmarkServer(b)
	defer cleanup()

	client := server.Client()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL + "/ready")
		if err != nil {
			b.Fatalf("Failed to call ready check: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("Ready check status = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	}
}

// BenchmarkStatusCheck benchmarks the status endpoint
func BenchmarkStatusCheck(b *testing.B) {
	server, cleanup := setupBenchmarkServer(b)
	defer cleanup()

	client := server.Client()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL + "/status")
		if err != nil {
			b.Fatalf("Failed to call status check: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("Status check status = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	}
}

// BenchmarkSubmitWorkflow benchmarks workflow submission
func BenchmarkSubmitWorkflow(b *testing.B) {
	server, cleanup := setupBenchmarkServer(b)
	defer cleanup()

	client := server.Client()

	workflowReq := models.WorkflowRequest{
		Name:        "benchmark-workflow",
		Description: "Benchmark test workflow",
		Tasks: []models.TaskDefinition{
			{
				ID:   "task-1",
				Name: "Benchmark task",
				Type: "http",
			},
		},
	}

	body, _ := json.Marshal(workflowReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Post(server.URL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
		if err != nil {
			b.Fatalf("Failed to submit workflow: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			b.Fatalf("Submit workflow status = %v, want %v", resp.StatusCode, http.StatusCreated)
		}
	}
}

// BenchmarkGetWorkflow benchmarks workflow status retrieval
func BenchmarkGetWorkflow(b *testing.B) {
	server, cleanup := setupBenchmarkServer(b)
	defer cleanup()

	client := server.Client()

	// Submit a workflow first
	workflowReq := models.WorkflowRequest{
		Name: "benchmark-workflow",
		Tasks: []models.TaskDefinition{
			{
				ID:   "task-1",
				Name: "Benchmark task",
				Type: "http",
			},
		},
	}

	body, _ := json.Marshal(workflowReq)
	resp, err := client.Post(server.URL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
	if err != nil {
		b.Fatalf("Failed to submit workflow: %v", err)
	}

	var submitResp models.WorkflowResponse
	json.NewDecoder(resp.Body).Decode(&submitResp)
	resp.Body.Close()

	workflowID := submitResp.ID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL + "/api/v1/workflows/" + workflowID)
		if err != nil {
			b.Fatalf("Failed to get workflow: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("Get workflow status = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	}
}

// BenchmarkListWorkflows benchmarks workflow listing
func BenchmarkListWorkflows(b *testing.B) {
	server, cleanup := setupBenchmarkServer(b)
	defer cleanup()

	client := server.Client()

	// Submit some workflows first
	for i := 0; i < 10; i++ {
		workflowReq := models.WorkflowRequest{
			Name: fmt.Sprintf("benchmark-workflow-%d", i),
			Tasks: []models.TaskDefinition{
				{
					ID:   "task-1",
					Name: "Benchmark task",
					Type: "http",
				},
			},
		}

		body, _ := json.Marshal(workflowReq)
		resp, _ := client.Post(server.URL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
		resp.Body.Close()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL + "/api/v1/workflows?limit=10")
		if err != nil {
			b.Fatalf("Failed to list workflows: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("List workflows status = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	}
}

// BenchmarkCancelWorkflow benchmarks workflow cancellation
func BenchmarkCancelWorkflow(b *testing.B) {
	server, cleanup := setupBenchmarkServer(b)
	defer cleanup()

	client := server.Client()

	// Pre-create workflows for cancellation
	workflowIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		workflowReq := models.WorkflowRequest{
			Name: fmt.Sprintf("cancel-benchmark-%d", i),
			Tasks: []models.TaskDefinition{
				{
					ID:   "task-1",
					Name: "Benchmark task",
					Type: "http",
				},
			},
		}

		body, _ := json.Marshal(workflowReq)
		resp, _ := client.Post(server.URL+"/api/v1/workflows", "application/json", bytes.NewReader(body))

		var submitResp models.WorkflowResponse
		json.NewDecoder(resp.Body).Decode(&submitResp)
		resp.Body.Close()

		workflowIDs[i] = submitResp.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Post(server.URL+"/api/v1/workflows/"+workflowIDs[i]+"/cancel", "application/json", nil)
		if err != nil {
			b.Fatalf("Failed to cancel workflow: %v", err)
		}
		resp.Body.Close()
	}
}

// BenchmarkEndToEndWorkflow benchmarks the complete workflow lifecycle
func BenchmarkEndToEndWorkflow(b *testing.B) {
	server, cleanup := setupBenchmarkServer(b)
	defer cleanup()

	client := server.Client()

	workflowReq := models.WorkflowRequest{
		Name:        "e2e-benchmark",
		Description: "End-to-end benchmark workflow",
		Tasks: []models.TaskDefinition{
			{
				ID:   "task-1",
				Name: "First task",
				Type: "http",
			},
			{
				ID:        "task-2",
				Name:      "Second task",
				Type:      "script",
				DependsOn: []string{"task-1"},
			},
		},
	}

	body, _ := json.Marshal(workflowReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Submit workflow
		resp, err := client.Post(server.URL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
		if err != nil {
			b.Fatalf("Failed to submit workflow: %v", err)
		}

		var submitResp models.WorkflowResponse
		json.NewDecoder(resp.Body).Decode(&submitResp)
		resp.Body.Close()

		workflowID := submitResp.ID

		// Get workflow status
		resp, err = client.Get(server.URL + "/api/v1/workflows/" + workflowID)
		if err != nil {
			b.Fatalf("Failed to get workflow: %v", err)
		}
		resp.Body.Close()

		// Cancel workflow
		resp, err = client.Post(server.URL+"/api/v1/workflows/"+workflowID+"/cancel", "application/json", nil)
		if err != nil {
			b.Fatalf("Failed to cancel workflow: %v", err)
		}
		resp.Body.Close()
	}
}
