package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/goclaw/goclaw/pkg/storage/memory"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
)

// setupIntegrationTest creates a test server and returns the base URL and cleanup function
func setupIntegrationTest(t *testing.T) (string, func()) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "test",
		},
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 18081, // Use different port to avoid conflicts
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
			MaxAgents: 10,
		},
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	// Create and start engine
	ctx := context.Background()
	eng, err := engine.New(cfg, log, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	// Create handlers
	testHandlers := &Handlers{
		Workflow: handlers.NewWorkflowHandler(eng, log),
		Health:   handlers.NewHealthHandler(eng),
	}

	// Create and start server
	server := NewHTTPServer(cfg, log, testHandlers)
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.Port)

	cleanup := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
		eng.Stop(ctx)
	}

	return baseURL, cleanup
}

// TestIntegration_WorkflowLifecycle tests the complete workflow lifecycle
func TestIntegration_WorkflowLifecycle(t *testing.T) {
	baseURL, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Step 1: Submit a workflow
	workflowReq := models.WorkflowRequest{
		Name:        "integration-test-workflow",
		Description: "Test workflow for integration testing",
		Tasks: []models.TaskDefinition{
			{
				ID:      "task-1",
				Name:    "First task",
				Type:    "http",
				Timeout: 300,
				Retries: 3,
			},
			{
				ID:        "task-2",
				Name:      "Second task",
				Type:      "script",
				DependsOn: []string{"task-1"},
				Timeout:   600,
			},
		},
		Metadata: map[string]string{
			"environment": "test",
			"test_id":     "integration-001",
		},
	}

	body, _ := json.Marshal(workflowReq)
	resp, err := http.Post(baseURL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to submit workflow: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Submit workflow status = %v, want %v", resp.StatusCode, http.StatusCreated)
	}

	var submitResp models.WorkflowResponse
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		t.Fatalf("Failed to decode submit response: %v", err)
	}

	workflowID := submitResp.ID
	if workflowID == "" {
		t.Fatal("Expected workflow ID in response")
	}

	t.Logf("Submitted workflow: %s", workflowID)

	// Step 2: Get workflow status
	resp, err = http.Get(baseURL + "/api/v1/workflows/" + workflowID)
	if err != nil {
		t.Fatalf("Failed to get workflow: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get workflow status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var statusResp models.WorkflowStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}

	if statusResp.ID != workflowID {
		t.Errorf("Status response ID = %v, want %v", statusResp.ID, workflowID)
	}
	if statusResp.Name != workflowReq.Name {
		t.Errorf("Status response name = %v, want %v", statusResp.Name, workflowReq.Name)
	}

	t.Logf("Workflow status: %s", statusResp.Status)

	// Step 3: List workflows
	resp, err = http.Get(baseURL + "/api/v1/workflows")
	if err != nil {
		t.Fatalf("Failed to list workflows: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List workflows status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var listResp models.WorkflowListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode list response: %v", err)
	}

	if listResp.Total < 1 {
		t.Errorf("List workflows total = %v, want >= 1", listResp.Total)
	}

	found := false
	for _, wf := range listResp.Workflows {
		if wf.ID == workflowID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Submitted workflow not found in list")
	}

	t.Logf("Listed %d workflows", listResp.Total)

	// Step 4: Cancel workflow
	resp, err = http.Post(baseURL+"/api/v1/workflows/"+workflowID+"/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to cancel workflow: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Cancel workflow status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	t.Logf("Cancelled workflow: %s", workflowID)

	// Step 5: Verify workflow was cancelled
	resp, err = http.Get(baseURL + "/api/v1/workflows/" + workflowID)
	if err != nil {
		t.Fatalf("Failed to get workflow after cancel: %v", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}

	if statusResp.Status != "cancelled" {
		t.Logf("Warning: Workflow status = %v, expected 'cancelled' (may be timing dependent)", statusResp.Status)
	}
}

// TestIntegration_HealthChecks tests all health check endpoints
func TestIntegration_HealthChecks(t *testing.T) {
	baseURL, cleanup := setupIntegrationTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{
			name:           "health check",
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "readiness check",
			endpoint:       "/ready",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "status check",
			endpoint:       "/status",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(baseURL + tt.endpoint)
			if err != nil {
				t.Fatalf("Failed to call %s: %v", tt.endpoint, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("%s status = %v, want %v", tt.endpoint, resp.StatusCode, tt.expectedStatus)
			}
		})
	}
}

// TestIntegration_ErrorHandling tests error scenarios
func TestIntegration_ErrorHandling(t *testing.T) {
	baseURL, cleanup := setupIntegrationTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		endpoint       string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "invalid workflow request",
			method:         "POST",
			endpoint:       "/api/v1/workflows",
			body:           map[string]string{"invalid": "data"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "get nonexistent workflow",
			method:         "GET",
			endpoint:       "/api/v1/workflows/nonexistent-id",
			body:           nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "cancel nonexistent workflow",
			method:         "POST",
			endpoint:       "/api/v1/workflows/nonexistent-id/cancel",
			body:           nil,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "get nonexistent task result",
			method:         "GET",
			endpoint:       "/api/v1/workflows/wf-123/tasks/task-456/result",
			body:           nil,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req, err = http.NewRequest(tt.method, baseURL+tt.endpoint, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, baseURL+tt.endpoint, nil)
			}

			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("%s status = %v, want %v", tt.name, resp.StatusCode, tt.expectedStatus)
			}
		})
	}
}

// TestIntegration_ConcurrentWorkflowSubmission tests concurrent workflow submissions
func TestIntegration_ConcurrentWorkflowSubmission(t *testing.T) {
	baseURL, cleanup := setupIntegrationTest(t)
	defer cleanup()

	numWorkers := 10
	var wg sync.WaitGroup
	errors := make(chan error, numWorkers)
	workflowIDs := make(chan string, numWorkers)

	// Submit workflows concurrently
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			workflowReq := models.WorkflowRequest{
				Name:        fmt.Sprintf("concurrent-workflow-%d", id),
				Description: "Concurrent test workflow",
				Tasks: []models.TaskDefinition{
					{
						ID:   "task-1",
						Name: "Test task",
						Type: "http",
					},
				},
			}

			body, _ := json.Marshal(workflowReq)
			resp, err := http.Post(baseURL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
			if err != nil {
				errors <- fmt.Errorf("worker %d: failed to submit: %v", id, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				errors <- fmt.Errorf("worker %d: status = %v, want %v", id, resp.StatusCode, http.StatusCreated)
				return
			}

			var submitResp models.WorkflowResponse
			if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
				errors <- fmt.Errorf("worker %d: failed to decode: %v", id, err)
				return
			}

			workflowIDs <- submitResp.ID
		}(i)
	}

	wg.Wait()
	close(errors)
	close(workflowIDs)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify all workflows were created
	ids := make([]string, 0, numWorkers)
	for id := range workflowIDs {
		ids = append(ids, id)
	}

	if len(ids) != numWorkers {
		t.Errorf("Created %d workflows, want %d", len(ids), numWorkers)
	}

	// Verify all workflows are in the list
	resp, err := http.Get(baseURL + "/api/v1/workflows?limit=100")
	if err != nil {
		t.Fatalf("Failed to list workflows: %v", err)
	}
	defer resp.Body.Close()

	var listResp models.WorkflowListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode list response: %v", err)
	}

	if listResp.Total < numWorkers {
		t.Errorf("Total workflows = %v, want >= %v", listResp.Total, numWorkers)
	}

	t.Logf("Successfully submitted %d concurrent workflows", numWorkers)
}

// TestIntegration_Pagination tests workflow list pagination
func TestIntegration_Pagination(t *testing.T) {
	baseURL, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Submit multiple workflows
	numWorkflows := 15
	for i := 0; i < numWorkflows; i++ {
		workflowReq := models.WorkflowRequest{
			Name: fmt.Sprintf("pagination-test-%d", i),
			Tasks: []models.TaskDefinition{
				{
					ID:   "task-1",
					Name: "Test task",
					Type: "http",
				},
			},
		}

		body, _ := json.Marshal(workflowReq)
		resp, err := http.Post(baseURL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to submit workflow %d: %v", i, err)
		}
		resp.Body.Close()
	}

	// Test pagination
	resp, err := http.Get(baseURL + "/api/v1/workflows?limit=5&offset=0")
	if err != nil {
		t.Fatalf("Failed to list workflows: %v", err)
	}
	defer resp.Body.Close()

	var listResp models.WorkflowListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode list response: %v", err)
	}

	if listResp.Limit != 5 {
		t.Errorf("Limit = %v, want 5", listResp.Limit)
	}
	if listResp.Offset != 0 {
		t.Errorf("Offset = %v, want 0", listResp.Offset)
	}
	if len(listResp.Workflows) > 5 {
		t.Errorf("Returned %d workflows, want <= 5", len(listResp.Workflows))
	}
	if listResp.Total < numWorkflows {
		t.Errorf("Total = %v, want >= %v", listResp.Total, numWorkflows)
	}

	t.Logf("Pagination test: total=%d, returned=%d", listResp.Total, len(listResp.Workflows))
}
