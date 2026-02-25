package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	dgbadger "github.com/dgraph-io/badger/v4"
	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/memory"
)

type nopLogger struct{}

func (n *nopLogger) Debug(msg string, args ...any) {}
func (n *nopLogger) Info(msg string, args ...any)  {}
func (n *nopLogger) Warn(msg string, args ...any)  {}
func (n *nopLogger) Error(msg string, args ...any) {}

func setupMemoryHandler(t *testing.T) (*MemoryHandler, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "goclaw-memhandler-*")
	if err != nil {
		t.Fatal(err)
	}
	opts := dgbadger.DefaultOptions(dir)
	opts.Logger = nil
	db, err := dgbadger.Open(opts)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}

	cfg := &config.MemoryConfig{
		Enabled: true, VectorDimension: 3, VectorWeight: 0.7, BM25Weight: 0.3,
		L1CacheSize: 100, ForgetThreshold: 0.1, DecayInterval: 1<<63 - 1, DefaultStability: 24.0,
		BM25: config.BM25Config{K1: 1.5, B: 0.75},
	}
	l1 := memory.NewL1Cache(cfg.L1CacheSize)
	l2 := memory.NewL2Badger(db)
	ts := memory.NewTieredStorage(l1, l2)
	hub := memory.NewMemoryHub(cfg, ts, nil)
	hub.Start(context.Background())

	handler := NewMemoryHandler(hub, &nopLogger{})
	cleanup := func() {
		hub.Stop(context.Background())
		db.Close()
		os.RemoveAll(dir)
	}
	return handler, cleanup
}

func withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- 10.11 API 端点单元测试 ---

func TestMemoryHandler_StoreMemory(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	body := `{"content":"test memory content","metadata":{"type":"fact"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()

	h.StoreMemory(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("StoreMemory() status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp memorizeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID in response")
	}
}

func TestMemoryHandler_StoreMemory_EmptyContent(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	body := `{"content":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()

	h.StoreMemory(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("StoreMemory() with empty content status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMemoryHandler_StoreMemory_NoSessionID(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	body := `{"content":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "")
	w := httptest.NewRecorder()

	h.StoreMemory(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("StoreMemory() without session ID status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMemoryHandler_StoreMemory_InvalidJSON(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()

	h.StoreMemory(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("StoreMemory() with invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMemoryHandler_QueryMemory(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	// Store an entry first
	body := `{"content":"Go is a compiled language"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.StoreMemory(w, req)

	// Query
	req = httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-1?query=compiled+language&limit=5", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.QueryMemory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("QueryMemory() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestMemoryHandler_QueryMemory_NoQuery(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-1", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.QueryMemory(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("QueryMemory() without query status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMemoryHandler_DeleteMemory(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	// Store an entry
	body := `{"content":"to be deleted"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.StoreMemory(w, req)

	var storeResp memorizeResponse
	json.NewDecoder(w.Body).Decode(&storeResp)

	// Delete it
	delBody, _ := json.Marshal(deleteRequest{IDs: []string{storeResp.ID}})
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/memory/session-1", bytes.NewBuffer(delBody))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.DeleteMemory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DeleteMemory() status = %d, want %d", w.Code, http.StatusOK)
	}

	var delResp deleteResponse
	json.NewDecoder(w.Body).Decode(&delResp)
	if delResp.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", delResp.Deleted)
	}
}

func TestMemoryHandler_DeleteMemory_EmptyIDs(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	body := `{"ids":[]}`
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/memory/session-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.DeleteMemory(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("DeleteMemory() with empty IDs status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMemoryHandler_ListMemory(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	// Store entries
	for i := 0; i < 3; i++ {
		body := `{"content":"entry content"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req = withChiURLParam(req, "sessionID", "session-1")
		w := httptest.NewRecorder()
		h.StoreMemory(w, req)
	}

	// List
	req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-1/list?limit=10&offset=0", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.ListMemory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListMemory() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total, ok := resp["total"].(float64)
	if !ok || int(total) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
}

func TestMemoryHandler_GetStats(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	// Store an entry
	body := `{"content":"stats test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.StoreMemory(w, req)

	// Get stats
	req = httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-1/stats", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.GetStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetStats() status = %d, want %d", w.Code, http.StatusOK)
	}

	var stats memory.MemoryStats
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.TotalEntries != 1 {
		t.Errorf("expected 1 entry, got %d", stats.TotalEntries)
	}
}

func TestMemoryHandler_DeleteSession(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	// Store entries
	for i := 0; i < 3; i++ {
		body := `{"content":"session entry"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req = withChiURLParam(req, "sessionID", "session-1")
		w := httptest.NewRecorder()
		h.StoreMemory(w, req)
	}

	// Delete session
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/memory/session-1/all", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.DeleteSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DeleteSession() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp deleteResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Deleted != 3 {
		t.Errorf("expected 3 deleted, got %d", resp.Deleted)
	}
}

func TestMemoryHandler_DeleteWeakMemories(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	// Store an entry
	body := `{"content":"weak memory test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.StoreMemory(w, req)

	// Delete weak memories with very high threshold (should delete everything)
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/memory/session-1/weak?threshold=2.0", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.DeleteWeakMemories(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DeleteWeakMemories() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp deleteResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Deleted != 1 {
		t.Errorf("expected 1 deleted (threshold=2.0 should delete all), got %d", resp.Deleted)
	}
}

// --- 10.12 API 端点集成测试 ---

func TestMemoryHandler_Integration_StoreQueryDelete(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	// Store multiple entries
	entries := []string{
		`{"content":"Go is a compiled language"}`,
		`{"content":"Python is an interpreted language"}`,
		`{"content":"Rust has zero-cost abstractions"}`,
	}
	var storedIDs []string
	for _, body := range entries {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req = withChiURLParam(req, "sessionID", "session-1")
		w := httptest.NewRecorder()
		h.StoreMemory(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("store failed: %s", w.Body.String())
		}
		var resp memorizeResponse
		json.NewDecoder(w.Body).Decode(&resp)
		storedIDs = append(storedIDs, resp.ID)
	}

	// Verify count via stats
	req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-1/stats", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.GetStats(w, req)
	var stats memory.MemoryStats
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.TotalEntries != 3 {
		t.Errorf("expected 3 entries, got %d", stats.TotalEntries)
	}

	// Query
	req = httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-1?query=compiled+language&limit=10", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.QueryMemory(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("query failed: %s", w.Body.String())
	}

	// Delete one entry
	delBody, _ := json.Marshal(deleteRequest{IDs: []string{storedIDs[0]}})
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/memory/session-1", bytes.NewBuffer(delBody))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.DeleteMemory(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete failed: %s", w.Body.String())
	}

	// Verify count decreased
	req = httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-1/stats", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.GetStats(w, req)
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.TotalEntries != 2 {
		t.Errorf("expected 2 entries after delete, got %d", stats.TotalEntries)
	}

	// Delete session
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/memory/session-1/all", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.DeleteSession(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete session failed: %s", w.Body.String())
	}

	// Verify empty
	req = httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-1/stats", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.GetStats(w, req)
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.TotalEntries != 0 {
		t.Errorf("expected 0 entries after session delete, got %d", stats.TotalEntries)
	}
}

func TestMemoryHandler_Integration_SessionIsolation(t *testing.T) {
	h, cleanup := setupMemoryHandler(t)
	defer cleanup()

	// Store in session-1
	body := `{"content":"session 1 data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-1")
	w := httptest.NewRecorder()
	h.StoreMemory(w, req)

	// Store in session-2
	body = `{"content":"session 2 data"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/memory/session-2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "sessionID", "session-2")
	w = httptest.NewRecorder()
	h.StoreMemory(w, req)

	// Verify session-1 has 1 entry
	req = httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-1/stats", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.GetStats(w, req)
	var stats memory.MemoryStats
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.TotalEntries != 1 {
		t.Errorf("session-1: expected 1 entry, got %d", stats.TotalEntries)
	}

	// Delete session-1 should not affect session-2
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/memory/session-1/all", nil)
	req = withChiURLParam(req, "sessionID", "session-1")
	w = httptest.NewRecorder()
	h.DeleteSession(w, req)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/memory/session-2/stats", nil)
	req = withChiURLParam(req, "sessionID", "session-2")
	w = httptest.NewRecorder()
	h.GetStats(w, req)
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.TotalEntries != 1 {
		t.Errorf("session-2: expected 1 entry after session-1 delete, got %d", stats.TotalEntries)
	}
}
