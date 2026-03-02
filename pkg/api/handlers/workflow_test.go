package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/goclaw/goclaw/pkg/storage/memory"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/api/response"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
)

func createTestEngine(t *testing.T) (*engine.Engine, func()) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "development",
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents: 10,
		},
	}
	return createTestEngineWithConfig(t, cfg)
}

func createTestEngineWithConfig(t *testing.T, cfg *config.Config) (*engine.Engine, func()) {
	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	eng, err := engine.New(cfg, log, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	cleanup := func() {
		eng.Stop(ctx)
	}

	return eng, cleanup
}

func waitForWorkflowStatus(t *testing.T, eng *engine.Engine, workflowID, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := eng.GetWorkflowStatusResponse(context.Background(), workflowID)
		if err == nil && status.Status == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("workflow %s did not reach %s within %v", workflowID, want, timeout)
}

func TestWorkflowHandler_SubmitWorkflow_Success(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	// Create valid request
	reqBody := models.WorkflowRequest{
		Name:        "test-workflow",
		Description: "Test workflow description",
		Tasks: []models.TaskDefinition{
			{
				ID:   "task-1",
				Name: "First task",
				Type: "http",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitWorkflow(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("SubmitWorkflow() status = %v, want %v, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify response structure
	var resp models.WorkflowResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID == "" {
		t.Error("Expected workflow ID in response")
	}
	if resp.WorkflowID == "" {
		t.Error("Expected workflow_id in response")
	}
	if resp.WorkflowID != resp.ID {
		t.Errorf("Response workflow_id = %v, want %v", resp.WorkflowID, resp.ID)
	}
	if resp.Name != reqBody.Name {
		t.Errorf("Response name = %v, want %v", resp.Name, reqBody.Name)
	}
}

func TestWorkflowHandler_SubmitWorkflow_AsyncFlag(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	reqBody := models.WorkflowRequest{
		Name:  "async-workflow",
		Async: true,
		Tasks: []models.TaskDefinition{
			{
				ID:   "task-1",
				Name: "First task",
				Type: "function",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitWorkflow(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("SubmitWorkflow() status = %v, want %v, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp models.WorkflowResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "pending" {
		t.Fatalf("response status = %s, want pending", resp.Status)
	}

	status, err := eng.GetWorkflowStatusResponse(context.Background(), resp.ID)
	if err != nil {
		t.Fatalf("GetWorkflowStatusResponse() error = %v", err)
	}
	if status.Status != "pending" {
		t.Fatalf("persisted status = %s, want pending", status.Status)
	}
}

func TestWorkflowHandler_SubmitWorkflow_InvalidJSON(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitWorkflow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SubmitWorkflow() with invalid JSON status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestWorkflowHandler_SubmitWorkflow_ValidationError(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	// Create request missing required fields
	reqBody := models.WorkflowRequest{
		// Missing Name (required)
		Description: "Test workflow",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitWorkflow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SubmitWorkflow() with validation error status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestWorkflowHandler_GetWorkflow_Success(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	// First submit a workflow
	reqBody := models.WorkflowRequest{
		Name:        "test-workflow",
		Description: "Test workflow description",
		Tasks: []models.TaskDefinition{
			{
				ID:   "task-1",
				Name: "First task",
				Type: "http",
			},
		},
	}

	ctx := context.Background()
	workflowID, err := eng.SubmitWorkflowRequest(ctx, &reqBody)
	if err != nil {
		t.Fatalf("Failed to submit workflow: %v", err)
	}

	// Now get the workflow
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workflowID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetWorkflow(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetWorkflow() status = %v, want %v, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify response
	var resp models.WorkflowStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID != workflowID {
		t.Errorf("Response ID = %v, want %v", resp.ID, workflowID)
	}
	if resp.WorkflowID != workflowID {
		t.Errorf("Response workflow_id = %v, want %v", resp.WorkflowID, workflowID)
	}
}

func TestWorkflowHandler_GetWorkflow_NotFound(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetWorkflow(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetWorkflow() with nonexistent ID status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestWorkflowHandler_GetWorkflow_MissingID(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/", nil)
	rctx := chi.NewRouteContext()
	// Don't add ID parameter
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetWorkflow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GetWorkflow() with missing ID status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestWorkflowHandler_ListWorkflows_Empty(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	w := httptest.NewRecorder()

	handler.ListWorkflows(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListWorkflows() status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp models.WorkflowListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Total != 0 {
		t.Errorf("ListWorkflows() total = %v, want 0", resp.Total)
	}
	if resp.Limit != 50 {
		t.Errorf("ListWorkflows() default limit = %v, want 50", resp.Limit)
	}
	if len(resp.Workflows) != 0 {
		t.Errorf("ListWorkflows() workflows count = %v, want 0", len(resp.Workflows))
	}
}

func TestWorkflowHandler_ListWorkflows_WithWorkflows(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	// Submit a few workflows
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		reqBody := models.WorkflowRequest{
			Name:        "test-workflow",
			Description: "Test workflow description",
			Tasks: []models.TaskDefinition{
				{
					ID:   "task-1",
					Name: "First task",
					Type: "http",
				},
			},
		}
		_, err := eng.SubmitWorkflowRequest(ctx, &reqBody)
		if err != nil {
			t.Fatalf("Failed to submit workflow: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	w := httptest.NewRecorder()

	handler.ListWorkflows(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListWorkflows() status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp models.WorkflowListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("ListWorkflows() total = %v, want 3", resp.Total)
	}
	if len(resp.Workflows) != 3 {
		t.Errorf("ListWorkflows() workflows count = %v, want 3", len(resp.Workflows))
	}
	for _, wf := range resp.Workflows {
		if wf.WorkflowID == "" {
			t.Fatal("expected workflow_id in workflow summary")
		}
		if wf.WorkflowID != wf.ID {
			t.Fatalf("workflow summary workflow_id=%s, id=%s", wf.WorkflowID, wf.ID)
		}
	}
}

func TestWorkflowHandler_ListWorkflows_WithPagination(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	// Submit 5 workflows
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		reqBody := models.WorkflowRequest{
			Name:        "test-workflow",
			Description: "Test workflow description",
			Tasks: []models.TaskDefinition{
				{
					ID:   "task-1",
					Name: "First task",
					Type: "http",
				},
			},
		}
		_, err := eng.SubmitWorkflowRequest(ctx, &reqBody)
		if err != nil {
			t.Fatalf("Failed to submit workflow: %v", err)
		}
	}

	// Request with limit=2
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?limit=2&offset=0", nil)
	w := httptest.NewRecorder()

	handler.ListWorkflows(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListWorkflows() status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp models.WorkflowListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Total != 5 {
		t.Errorf("ListWorkflows() total = %v, want 5", resp.Total)
	}
	if len(resp.Workflows) != 2 {
		t.Errorf("ListWorkflows() workflows count = %v, want 2", len(resp.Workflows))
	}
	if resp.Limit != 2 {
		t.Errorf("ListWorkflows() limit = %v, want 2", resp.Limit)
	}
}

func TestWorkflowHandler_ListWorkflows_InvalidPagination(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	tests := []string{
		"/api/v1/workflows?limit=abc",
		"/api/v1/workflows?limit=-1",
		"/api/v1/workflows?offset=abc",
		"/api/v1/workflows?offset=-1",
	}

	for _, endpoint := range tests {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		w := httptest.NewRecorder()

		handler.ListWorkflows(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("ListWorkflows(%s) status = %v, want %v", endpoint, w.Code, http.StatusBadRequest)
		}
	}
}

func TestWorkflowHandler_ListWorkflows_LimitCap(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?limit=1000", nil)
	w := httptest.NewRecorder()

	handler.ListWorkflows(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListWorkflows() status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp models.WorkflowListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Limit != 100 {
		t.Fatalf("ListWorkflows() effective limit = %v, want 100", resp.Limit)
	}
}

func TestWorkflowHandler_CancelWorkflow_Success(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	// First submit a workflow
	reqBody := models.WorkflowRequest{
		Name:        "test-workflow",
		Description: "Test workflow description",
		Tasks: []models.TaskDefinition{
			{
				ID:   "task-1",
				Name: "First task",
				Type: "http",
			},
		},
	}

	ctx := context.Background()
	workflowID, err := eng.SubmitWorkflowRequest(ctx, &reqBody)
	if err != nil {
		t.Fatalf("Failed to submit workflow: %v", err)
	}

	// Now cancel the workflow
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID+"/cancel", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workflowID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.CancelWorkflow(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CancelWorkflow() status = %v, want %v, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp models.WorkflowResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.WorkflowID != workflowID || resp.ID != workflowID {
		t.Fatalf("cancel response IDs mismatch: id=%s workflow_id=%s want=%s", resp.ID, resp.WorkflowID, workflowID)
	}
	if resp.Status != "cancelled" {
		t.Fatalf("cancel response status = %s, want cancelled", resp.Status)
	}
}

func TestWorkflowHandler_CancelWorkflow_TimeoutContractAlignsWithTaskResult(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "development",
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents:           10,
			CancellationTimeout: 50 * time.Millisecond,
		},
	}
	eng, cleanup := createTestEngineWithConfig(t, cfg)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	reqBody := &models.WorkflowRequest{
		Name: "cancel-timeout-contract",
		Tasks: []models.TaskDefinition{
			{
				ID:   "task-1",
				Name: "First task",
				Type: "function",
			},
		},
	}

	statusResp, err := eng.SubmitWorkflowRuntime(context.Background(), reqBody, engine.SubmitWorkflowOptions{
		Mode: engine.SubmissionModeAsync,
		TaskFns: map[string]func(context.Context) error{
			"task-1": func(context.Context) error {
				// Ignore cancellation to force graceful timeout expiration.
				time.Sleep(200 * time.Millisecond)
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("SubmitWorkflowRuntime() error = %v", err)
	}
	workflowID := statusResp.ID
	waitForWorkflowStatus(t, eng, workflowID, "running", 2*time.Second)

	cancelReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID+"/cancel", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workflowID)
	cancelReq = cancelReq.WithContext(context.WithValue(cancelReq.Context(), chi.RouteCtxKey, rctx))
	cancelResp := httptest.NewRecorder()

	handler.CancelWorkflow(cancelResp, cancelReq)
	if cancelResp.Code != http.StatusOK {
		t.Fatalf("CancelWorkflow() status = %v, want %v, body: %s", cancelResp.Code, http.StatusOK, cancelResp.Body.String())
	}

	var wfResp models.WorkflowResponse
	if err := json.NewDecoder(cancelResp.Body).Decode(&wfResp); err != nil {
		t.Fatalf("failed to decode cancel response: %v", err)
	}
	if wfResp.Status != "failed" {
		t.Fatalf("cancel response status = %s, want failed", wfResp.Status)
	}

	taskReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID+"/tasks/task-1/result", nil)
	taskCtx := chi.NewRouteContext()
	taskCtx.URLParams.Add("id", workflowID)
	taskCtx.URLParams.Add("tid", "task-1")
	taskReq = taskReq.WithContext(context.WithValue(taskReq.Context(), chi.RouteCtxKey, taskCtx))
	taskResp := httptest.NewRecorder()

	handler.GetTaskResult(taskResp, taskReq)
	if taskResp.Code != http.StatusOK {
		t.Fatalf("GetTaskResult() status = %v, want %v, body: %s", taskResp.Code, http.StatusOK, taskResp.Body.String())
	}

	var resultResp models.TaskResultResponse
	if err := json.NewDecoder(taskResp.Body).Decode(&resultResp); err != nil {
		t.Fatalf("failed to decode task result response: %v", err)
	}
	if resultResp.Status != "failed" {
		t.Fatalf("task result status = %s, want failed", resultResp.Status)
	}
	if !strings.Contains(strings.ToLower(resultResp.Error), "deadline") {
		t.Fatalf("task result error = %q, expected deadline-derived error", resultResp.Error)
	}
}

func TestWorkflowHandler_CancelWorkflow_NotFound(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/nonexistent/cancel", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.CancelWorkflow(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("CancelWorkflow() with nonexistent ID status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestWorkflowHandler_CancelWorkflow_MissingID(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows//cancel", nil)
	rctx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.CancelWorkflow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CancelWorkflow() with missing ID status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestWorkflowHandler_GetTaskResult_NotFound(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/wf-123/tasks/task-456/result", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "wf-123")
	rctx.URLParams.Add("tid", "task-456")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetTaskResult(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetTaskResult() with nonexistent workflow status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestWorkflowHandler_GetTaskResult_MissingIDs(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	// Test missing workflow ID
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows//tasks/task-456/result", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tid", "task-456")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetTaskResult(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GetTaskResult() with missing workflow ID status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	// Test missing task ID
	req = httptest.NewRequest(http.MethodGet, "/api/v1/workflows/wf-123/tasks//result", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "wf-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()

	handler.GetTaskResult(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GetTaskResult() with missing task ID status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestWorkflowHandler_GetTaskResult_NonTerminalPending(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	reqBody := models.WorkflowRequest{
		Name: "non-terminal-task-result",
		Tasks: []models.TaskDefinition{
			{
				ID:   "task-1",
				Name: "First task",
				Type: "function",
			},
		},
	}

	workflowID, err := eng.SubmitWorkflowRequest(context.Background(), &reqBody)
	if err != nil {
		t.Fatalf("SubmitWorkflowRequest() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID+"/tasks/task-1/result", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workflowID)
	rctx.URLParams.Add("tid", "task-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetTaskResult(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("GetTaskResult() status = %v, want %v, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}

	var resp response.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error.Code != response.ErrCodeConflict {
		t.Fatalf("error code = %s, want %s", resp.Error.Code, response.ErrCodeConflict)
	}
	if strings.TrimSpace(resp.Error.RequestID) == "" {
		t.Fatal("expected non-empty request_id")
	}
}

func TestWorkflowHandler_SubmitWorkflow_AcceptsDependenciesField(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	body := `{
		"name":"deps-workflow",
		"tasks":[
			{"id":"task-1","name":"Task 1","type":"function"},
			{"id":"task-2","name":"Task 2","type":"function","dependencies":["task-1"]}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitWorkflow(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestWorkflowHandler_SubmitWorkflow_AcceptsDependsOnAlias(t *testing.T) {
	eng, cleanup := createTestEngine(t)
	defer cleanup()

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewWorkflowHandler(eng, log)

	body := `{
		"name":"depends-on-workflow",
		"tasks":[
			{"id":"task-1","name":"Task 1","type":"function"},
			{"id":"task-2","name":"Task 2","type":"function","depends_on":["task-1"]}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitWorkflow(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
}
