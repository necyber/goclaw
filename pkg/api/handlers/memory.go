package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/goclaw/goclaw/pkg/api/response"
	"github.com/goclaw/goclaw/pkg/memory"
)

// MemoryHandler handles memory-related API endpoints.
type MemoryHandler struct {
	hub    *memory.MemoryHub
	logger memoryLogger
}

type memoryLogger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// NewMemoryHandler creates a new memory handler.
func NewMemoryHandler(hub *memory.MemoryHub, log memoryLogger) *MemoryHandler {
	return &MemoryHandler{
		hub:    hub,
		logger: log,
	}
}

// --- Request/Response types ---

type memorizeRequest struct {
	Content  string            `json:"content"`
	Vector   []float32         `json:"vector,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type memorizeResponse struct {
	ID string `json:"id"`
}

type queryRequest struct {
	Query    string            `json:"query,omitempty"`
	Vector   []float32         `json:"vector,omitempty"`
	Mode     string            `json:"mode,omitempty"`
	TopK     int               `json:"top_k,omitempty"`
	Filters  map[string]string `json:"filters,omitempty"`
}

type deleteRequest struct {
	IDs []string `json:"ids"`
}

type deleteResponse struct {
	Deleted int `json:"deleted"`
}

// StoreMemory handles POST /api/v1/memory/{sessionID}
func (h *MemoryHandler) StoreMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Session ID is required", getRequestID(ctx))
		return
	}

	var req memorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Invalid request body", getRequestID(ctx))
		return
	}

	if req.Content == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeValidationFailed, "Content is required", getRequestID(ctx))
		return
	}

	id, err := h.hub.Memorize(ctx, sessionID, req.Content, req.Vector, req.Metadata)
	if err != nil {
		h.logger.Error("Failed to store memory", "session_id", sessionID, "error", err)
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, "Failed to store memory", getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusCreated, memorizeResponse{ID: id})
}

// QueryMemory handles GET /api/v1/memory/{sessionID}
func (h *MemoryHandler) QueryMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Session ID is required", getRequestID(ctx))
		return
	}

	queryText := r.URL.Query().Get("query")
	mode := r.URL.Query().Get("mode")
	topK := 10
	if topKStr := r.URL.Query().Get("limit"); topKStr != "" {
		if v, err := strconv.Atoi(topKStr); err == nil && v > 0 {
			topK = v
		}
	}

	if queryText == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeValidationFailed, "Query parameter is required", getRequestID(ctx))
		return
	}

	query := memory.Query{
		Text: queryText,
		Mode: mode,
		TopK: topK,
	}

	// Parse metadata filters from query params
	filters := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(key) > 9 && key[:9] == "metadata." {
			filters[key[9:]] = values[0]
		}
	}
	if len(filters) > 0 {
		query.Filters = filters
	}

	results, err := h.hub.Retrieve(ctx, sessionID, query)
	if err != nil {
		h.logger.Error("Failed to query memory", "session_id", sessionID, "error", err)
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, "Failed to query memory", getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusOK, results)
}

// DeleteMemory handles DELETE /api/v1/memory/{sessionID}
func (h *MemoryHandler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Session ID is required", getRequestID(ctx))
		return
	}

	var req deleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Invalid request body", getRequestID(ctx))
		return
	}

	if len(req.IDs) == 0 {
		response.Error(w, http.StatusBadRequest, response.ErrCodeValidationFailed, "At least one entry ID is required", getRequestID(ctx))
		return
	}

	if err := h.hub.Forget(ctx, sessionID, req.IDs); err != nil {
		h.logger.Error("Failed to delete memory", "session_id", sessionID, "error", err)
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, "Failed to delete memory", getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusOK, deleteResponse{Deleted: len(req.IDs)})
}

// ListMemory handles GET /api/v1/memory/{sessionID}/list
func (h *MemoryHandler) ListMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Session ID is required", getRequestID(ctx))
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	entries, total, err := h.hub.List(ctx, sessionID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list memory", "session_id", sessionID, "error", err)
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, "Failed to list memory", getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetStats handles GET /api/v1/memory/{sessionID}/stats
func (h *MemoryHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Session ID is required", getRequestID(ctx))
		return
	}

	stats, err := h.hub.GetStats(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to get memory stats", "session_id", sessionID, "error", err)
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, "Failed to get memory stats", getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusOK, stats)
}

// DeleteSession handles DELETE /api/v1/memory/{sessionID}/all
func (h *MemoryHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Session ID is required", getRequestID(ctx))
		return
	}

	count, err := h.hub.DeleteSession(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to delete session", "session_id", sessionID, "error", err)
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, "Failed to delete session", getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusOK, deleteResponse{Deleted: count})
}

// DeleteWeakMemories handles DELETE /api/v1/memory/{sessionID}/weak
func (h *MemoryHandler) DeleteWeakMemories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		response.Error(w, http.StatusBadRequest, response.ErrCodeBadRequest, "Session ID is required", getRequestID(ctx))
		return
	}

	threshold := 0.1
	if v := r.URL.Query().Get("threshold"); v != "" {
		if t, err := strconv.ParseFloat(v, 64); err == nil {
			threshold = t
		}
	}

	count, err := h.hub.ForgetByThreshold(ctx, sessionID, threshold)
	if err != nil {
		h.logger.Error("Failed to delete weak memories", "session_id", sessionID, "error", err)
		response.Error(w, http.StatusInternalServerError, response.ErrCodeInternalServer, "Failed to delete weak memories", getRequestID(ctx))
		return
	}

	response.JSON(w, http.StatusOK, deleteResponse{Deleted: count})
}
