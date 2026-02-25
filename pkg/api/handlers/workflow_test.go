package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/models"
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
	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	eng, err := engine.New(cfg, log)
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
	if resp.Name != reqBody.Name {
		t.Errorf("Response name = %v, want %v", resp.Name, reqBody.Name)
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

	if w.Code != http.StatusConflict {
		t.Errorf("CancelWorkflow() with nonexistent ID status = %v, want %v", w.Code, http.StatusConflict)
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
