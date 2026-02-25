// Package handlers provides HTTP request handlers.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/api/response"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
)

// WorkflowHandler handles workflow-related endpoints.
type WorkflowHandler struct {
	engine    *engine.Engine
	logger    logger.Logger
	validator *validator.Validate
}

// NewWorkflowHandler creates a new workflow handler.
func NewWorkflowHandler(eng *engine.Engine, log logger.Logger) *WorkflowHandler {
	return &WorkflowHandler{
		engine:    eng,
		logger:    log,
		validator: validator.New(),
	}
}

// SubmitWorkflow handles POST /api/v1/workflows
func (h *WorkflowHandler) SubmitWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req models.WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Invalid request body", getRequestID(ctx))
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.logger.Error("Validation failed", "error", err)
		response.Error(w, http.StatusBadRequest, response.ErrCodeValidationFailed, err.Error(), getRequestID(ctx))
		return
	}

	// Submit workflow to engine
	workflowID, err := h.engine.SubmitWorkflowRequest(ctx, &req)
	if err != nil {
		h.logger.Error("Failed to submit workflow", "error", err)
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, "Failed to submit workflow", getRequestID(ctx))
		return
	}

	// Return response
	resp := models.WorkflowResponse{
		ID:      workflowID,
		Name:    req.Name,
		Status:  "pending",
		Message: "Workflow submitted successfully",
	}

	response.JSON(w, http.StatusCreated, resp)
}

// GetWorkflow handles GET /api/v1/workflows/{id}
func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "id")

	if workflowID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Workflow ID is required", getRequestID(ctx))
		return
	}

	// Get workflow status from engine
	status, err := h.engine.GetWorkflowStatusResponse(ctx, workflowID)
	if err != nil {
		h.logger.Error("Failed to get workflow", "id", workflowID, "error", err)
		response.Error(w, http.StatusNotFound, response.ErrCodeNotFound, "Workflow not found", getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusOK, status)
}

// ListWorkflows handles GET /api/v1/workflows
func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filter := models.WorkflowFilter{
		Status: r.URL.Query().Get("status"),
		Limit:  10,
		Offset: 0,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	// Get workflows from engine
	workflows, total, err := h.engine.ListWorkflowsResponse(ctx, filter)
	if err != nil {
		h.logger.Error("Failed to list workflows", "error", err)
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, "Failed to list workflows", getRequestID(ctx))
		return
	}

	// Build response
	summaries := make([]models.WorkflowSummary, 0, len(workflows))
	for _, wf := range workflows {
		summaries = append(summaries, models.WorkflowSummary{
			ID:          wf.ID,
			Name:        wf.Name,
			Status:      wf.Status,
			CreatedAt:   wf.CreatedAt,
			CompletedAt: wf.CompletedAt,
			TaskCount:   len(wf.Tasks),
		})
	}

	resp := models.WorkflowListResponse{
		Workflows: summaries,
		Total:     total,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
	}

	response.JSON(w, http.StatusOK, resp)
}

// CancelWorkflow handles POST /api/v1/workflows/{id}/cancel
func (h *WorkflowHandler) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "id")

	if workflowID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Workflow ID is required", getRequestID(ctx))
		return
	}

	// Cancel workflow
	if err := h.engine.CancelWorkflowRequest(ctx, workflowID); err != nil {
		h.logger.Error("Failed to cancel workflow", "id", workflowID, "error", err)
		response.Error(w, http.StatusConflict, response.ErrCodeConflict, err.Error(), getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Workflow cancelled successfully",
	})
}

// GetTaskResult handles GET /api/v1/workflows/{id}/tasks/{tid}/result
func (h *WorkflowHandler) GetTaskResult(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "id")
	taskID := chi.URLParam(r, "tid")

	if workflowID == "" || taskID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Workflow ID and Task ID are required", getRequestID(ctx))
		return
	}

	// Get task result from engine
	result, err := h.engine.GetTaskResultResponse(ctx, workflowID, taskID)
	if err != nil {
		h.logger.Error("Failed to get task result", "workflow_id", workflowID, "task_id", taskID, "error", err)
		response.Error(w, http.StatusNotFound, response.ErrCodeNotFound, "Task result not found", getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// getRequestID extracts request ID from context
func getRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value("request_id").(string); ok {
		return reqID
	}
	return "unknown"
}




