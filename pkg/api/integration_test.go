package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/api/response"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
	"github.com/goclaw/goclaw/pkg/storage/memory"
)

// setupIntegrationTest creates a test server and returns the base URL and cleanup function.
func setupIntegrationTest(t *testing.T) (string, *http.Client, func()) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.App.Name = "test"
	cfg.App.Environment = "test"
	cfg.Server.CORS.Enabled = false
	cfg.UI.Enabled = false
	cfg.Server.HTTP.ReadTimeout = 30 * time.Second
	cfg.Server.HTTP.WriteTimeout = 30 * time.Second
	cfg.Server.HTTP.IdleTimeout = 60 * time.Second
	cfg.Server.HTTP.ShutdownTimeout = 5 * time.Second
	cfg.Orchestration.MaxAgents = 10

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	ctx := context.Background()
	eng, err := engine.New(cfg, log, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	testHandlers := &Handlers{
		Workflow: handlers.NewWorkflowHandler(eng, log),
		Health:   handlers.NewHealthHandler(eng),
	}

	router := NewRouter(cfg, log, testHandlers)
	server := httptest.NewServer(router)
	client := server.Client()

	waitForReady(t, client, server.URL+"/health", 2*time.Second)

	cleanup := func() {
		server.Close()
		_ = eng.Stop(ctx)
	}

	return server.URL, client, cleanup
}

func waitForReady(t *testing.T, client *http.Client, endpoint string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(endpoint)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("service did not become ready: %s", endpoint)
}

// TestIntegration_WorkflowLifecycle tests the complete workflow lifecycle.
func TestIntegration_WorkflowLifecycle(t *testing.T) {
	baseURL, client, cleanup := setupIntegrationTest(t)
	defer cleanup()

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
				ID:           "task-2",
				Name:         "Second task",
				Type:         "script",
				Dependencies: []string{"task-1"},
				Timeout:      600,
			},
		},
		Metadata: map[string]string{
			"environment": "test",
			"test_id":     "integration-001",
		},
	}

	body, _ := json.Marshal(workflowReq)
	resp, err := client.Post(baseURL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
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
	if submitResp.ID == "" || submitResp.WorkflowID == "" {
		t.Fatal("expected id and workflow_id in submit response")
	}
	if submitResp.ID != submitResp.WorkflowID {
		t.Fatalf("id=%s workflow_id=%s mismatch", submitResp.ID, submitResp.WorkflowID)
	}
	workflowID := submitResp.ID

	resp, err = client.Get(baseURL + "/api/v1/workflows/" + workflowID)
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
	if statusResp.WorkflowID != workflowID || statusResp.ID != workflowID {
		t.Fatalf("status IDs mismatch id=%s workflow_id=%s want=%s", statusResp.ID, statusResp.WorkflowID, workflowID)
	}
	if statusResp.Name != workflowReq.Name {
		t.Errorf("Status response name = %v, want %v", statusResp.Name, workflowReq.Name)
	}

	resp, err = client.Get(baseURL + "/api/v1/workflows")
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
		if wf.WorkflowID == workflowID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Submitted workflow not found in list by workflow_id")
	}

	resp, err = client.Post(baseURL+"/api/v1/workflows/"+workflowID+"/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to cancel workflow: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Cancel workflow status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var cancelResp models.WorkflowResponse
	if err := json.NewDecoder(resp.Body).Decode(&cancelResp); err != nil {
		t.Fatalf("Failed to decode cancel response: %v", err)
	}
	if cancelResp.WorkflowID != workflowID || cancelResp.ID != workflowID {
		t.Fatalf("cancel IDs mismatch id=%s workflow_id=%s want=%s", cancelResp.ID, cancelResp.WorkflowID, workflowID)
	}
	if cancelResp.Status != "cancelled" {
		t.Fatalf("cancel status = %s, want cancelled", cancelResp.Status)
	}
}

// TestIntegration_HealthChecks tests all health check endpoints.
func TestIntegration_HealthChecks(t *testing.T) {
	baseURL, client, cleanup := setupIntegrationTest(t)
	defer cleanup()

	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/health status = %v, want %v", resp.StatusCode, http.StatusOK)
	}
	var healthResp models.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("failed to decode /health: %v", err)
	}
	if healthResp.Status != "healthy" || healthResp.Timestamp.IsZero() {
		t.Fatalf("unexpected /health payload: %+v", healthResp)
	}

	resp, err = client.Get(baseURL + "/ready")
	if err != nil {
		t.Fatalf("Failed to call /ready: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/ready status = %v, want %v", resp.StatusCode, http.StatusOK)
	}
	var readyResp models.ReadyResponse
	if err := json.NewDecoder(resp.Body).Decode(&readyResp); err != nil {
		t.Fatalf("failed to decode /ready: %v", err)
	}
	if _, ok := readyResp.Checks["engine"]; !ok {
		t.Fatal("expected /ready checks.engine")
	}
	if _, ok := readyResp.Checks["storage"]; !ok {
		t.Fatal("expected /ready checks.storage")
	}

	resp, err = client.Get(baseURL + "/status")
	if err != nil {
		t.Fatalf("Failed to call /status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/status status = %v, want %v", resp.StatusCode, http.StatusOK)
	}
	var statusResp models.StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		t.Fatalf("failed to decode /status: %v", err)
	}
	if statusResp.Timestamp.IsZero() || statusResp.Uptime == "" {
		t.Fatalf("unexpected /status payload: %+v", statusResp)
	}
	if _, ok := statusResp.Components["engine"]; !ok {
		t.Fatal("expected /status components.engine")
	}
	if _, ok := statusResp.Components["storage"]; !ok {
		t.Fatal("expected /status components.storage")
	}
}

// TestIntegration_ErrorHandling tests error scenarios.
func TestIntegration_ErrorHandling(t *testing.T) {
	baseURL, client, cleanup := setupIntegrationTest(t)
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
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "get nonexistent task result",
			method:         "GET",
			endpoint:       "/api/v1/workflows/wf-123/tasks/task-456/result",
			body:           nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid pagination",
			method:         "GET",
			endpoint:       "/api/v1/workflows?limit=-1",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
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

			resp, err := client.Do(req)
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

// TestIntegration_ErrorResponse_RequestID ensures middleware request-id is propagated in error payloads.
func TestIntegration_ErrorResponse_RequestID(t *testing.T) {
	baseURL, client, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/workflows?limit=-1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var errResp response.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Error.RequestID == "" || errResp.Error.RequestID == "unknown" {
		t.Fatalf("unexpected request_id: %q", errResp.Error.RequestID)
	}
	headerID := resp.Header.Get("X-Request-ID")
	if headerID == "" {
		t.Fatal("expected X-Request-ID header")
	}
	if errResp.Error.RequestID != headerID {
		t.Fatalf("error.request_id=%q header=%q mismatch", errResp.Error.RequestID, headerID)
	}
}

// TestIntegration_WorkflowPayloadCompatibility verifies both dependencies and depends_on payload forms.
func TestIntegration_WorkflowPayloadCompatibility(t *testing.T) {
	baseURL, client, cleanup := setupIntegrationTest(t)
	defer cleanup()

	payloads := []string{
		`{
			"name":"deps-canonical",
			"tasks":[
				{"id":"task-1","name":"Task 1","type":"function"},
				{"id":"task-2","name":"Task 2","type":"function","dependencies":["task-1"]}
			]
		}`,
		`{
			"name":"deps-alias",
			"tasks":[
				{"id":"task-1","name":"Task 1","type":"function"},
				{"id":"task-2","name":"Task 2","type":"function","depends_on":["task-1"]}
			]
		}`,
	}

	for i, payload := range payloads {
		resp, err := client.Post(baseURL+"/api/v1/workflows", "application/json", bytes.NewBufferString(payload))
		if err != nil {
			t.Fatalf("payload %d submit failed: %v", i, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("payload %d status=%d want=%d", i, resp.StatusCode, http.StatusCreated)
		}
	}
}

// TestIntegration_DocsEndpoints tests docs route reachability and compatibility alias.
func TestIntegration_DocsEndpoints(t *testing.T) {
	baseURL, client, cleanup := setupIntegrationTest(t)
	defer cleanup()

	endpoints := []string{
		"/docs",
		"/docs/",
		"/docs/openapi.yaml",
		"/swagger/index.html",
	}
	for _, ep := range endpoints {
		resp, err := client.Get(baseURL + ep)
		if err != nil {
			t.Fatalf("GET %s failed: %v", ep, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Fatalf("GET %s returned 404", ep)
		}
	}

	resp, err := client.Get(baseURL + "/docs/openapi.yaml")
	if err != nil {
		t.Fatalf("GET /docs/openapi.yaml failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /docs/openapi.yaml status=%d want=%d", resp.StatusCode, http.StatusOK)
	}

	var specBuf bytes.Buffer
	if _, err := specBuf.ReadFrom(resp.Body); err != nil {
		t.Fatalf("failed to read openapi spec: %v", err)
	}
	if !bytes.Contains(specBuf.Bytes(), []byte("openapi: 3.0.3")) {
		t.Fatalf("expected OpenAPI 3.0.3 document")
	}
}

// TestIntegration_ConcurrentWorkflowSubmission tests concurrent workflow submissions.
func TestIntegration_ConcurrentWorkflowSubmission(t *testing.T) {
	baseURL, client, cleanup := setupIntegrationTest(t)
	defer cleanup()

	numWorkers := 10
	var wg sync.WaitGroup
	errors := make(chan error, numWorkers)
	workflowIDs := make(chan string, numWorkers)

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
			resp, err := client.Post(baseURL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
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

			workflowIDs <- submitResp.WorkflowID
		}(i)
	}

	wg.Wait()
	close(errors)
	close(workflowIDs)

	for err := range errors {
		t.Error(err)
	}

	ids := make([]string, 0, numWorkers)
	for id := range workflowIDs {
		ids = append(ids, id)
	}

	if len(ids) != numWorkers {
		t.Errorf("Created %d workflows, want %d", len(ids), numWorkers)
	}

	resp, err := client.Get(baseURL + "/api/v1/workflows?limit=100")
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
}

// TestIntegration_Pagination tests workflow list pagination.
func TestIntegration_Pagination(t *testing.T) {
	baseURL, client, cleanup := setupIntegrationTest(t)
	defer cleanup()

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
		resp, err := client.Post(baseURL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to submit workflow %d: %v", i, err)
		}
		resp.Body.Close()
	}

	resp, err := client.Get(baseURL + "/api/v1/workflows?limit=5&offset=0")
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
}
