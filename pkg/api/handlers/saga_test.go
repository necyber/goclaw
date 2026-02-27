package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dgbadger "github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/logger"
	"github.com/goclaw/goclaw/pkg/saga"
)

func newSagaHandlerForTest(t *testing.T) (*SagaHandler, *saga.BadgerCheckpointStore, func()) {
	t.Helper()

	opts := dgbadger.DefaultOptions(t.TempDir())
	opts.Logger = nil
	db, err := dgbadger.Open(opts)
	if err != nil {
		t.Fatalf("open badger: %v", err)
	}

	wal, err := saga.NewBadgerWAL(db, saga.WALOptions{WriteMode: saga.WALWriteModeSync})
	if err != nil {
		t.Fatalf("new wal: %v", err)
	}
	checkpointStore, err := saga.NewBadgerCheckpointStore(db)
	if err != nil {
		t.Fatalf("new checkpoint store: %v", err)
	}
	checkpointer, err := saga.NewCheckpointer(checkpointStore)
	if err != nil {
		t.Fatalf("new checkpointer: %v", err)
	}
	orchestrator := saga.NewSagaOrchestrator(
		saga.WithWAL(wal),
		saga.WithCheckpointer(checkpointer),
		saga.WithSagaStore(saga.NewMemorySagaStore()),
	)
	recovery, err := saga.NewRecoveryManager(orchestrator, checkpointStore, nil)
	if err != nil {
		t.Fatalf("new recovery manager: %v", err)
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})
	handler := NewSagaHandler(orchestrator, checkpointStore, recovery, log)
	cleanup := func() {
		_ = wal.Close()
		_ = db.Close()
	}
	return handler, checkpointStore, cleanup
}

func TestSagaHandlerSubmitAndGet(t *testing.T) {
	handler, _, cleanup := newSagaHandlerForTest(t)
	defer cleanup()

	reqBody := models.SagaSubmitRequest{
		Name: "submit-get",
		Steps: []models.SagaStepRequest{
			{ID: "a"},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sagas", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.SubmitSaga(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("SubmitSaga() status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}

	var submitResp models.SagaSubmitResponse
	if err := json.NewDecoder(w.Body).Decode(&submitResp); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}

	// wait for async execution
	time.Sleep(50 * time.Millisecond)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/sagas/"+submitResp.SagaID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", submitResp.SagaID)
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx))
	getW := httptest.NewRecorder()
	handler.GetSaga(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GetSaga() status = %d, want %d, body=%s", getW.Code, http.StatusOK, getW.Body.String())
	}
}

func TestSagaHandlerSubmitValidationError(t *testing.T) {
	handler, _, cleanup := newSagaHandlerForTest(t)
	defer cleanup()

	reqBody := models.SagaSubmitRequest{
		Name: "invalid-cycle",
		Steps: []models.SagaStepRequest{
			{ID: "a", DependsOn: []string{"b"}},
			{ID: "b", DependsOn: []string{"a"}},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sagas", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.SubmitSaga(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("SubmitSaga() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSagaHandlerListSagas(t *testing.T) {
	handler, _, cleanup := newSagaHandlerForTest(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		reqBody := models.SagaSubmitRequest{
			Name: "list-saga",
			Steps: []models.SagaStepRequest{
				{ID: "a"},
			},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sagas", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.SubmitSaga(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("SubmitSaga() status = %d", w.Code)
		}
	}

	time.Sleep(60 * time.Millisecond)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sagas?limit=2&offset=0", nil)
	w := httptest.NewRecorder()
	handler.ListSagas(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListSagas() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp models.SagaListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if resp.Total < 3 {
		t.Fatalf("expected at least 3 sagas, got %d", resp.Total)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected page size 2, got %d", len(resp.Items))
	}
}

func TestSagaHandlerCompensateAndRecoverValidation(t *testing.T) {
	handler, checkpointStore, cleanup := newSagaHandlerForTest(t)
	defer cleanup()

	// submit saga with manual policy that enters pending-compensation
	reqBody := models.SagaSubmitRequest{
		Name:   "manual-compensate",
		Policy: "manual",
		Steps: []models.SagaStepRequest{
			{ID: "a", EnableCompensation: true},
			{ID: "b", DependsOn: []string{"a"}, ShouldFail: true},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sagas", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.SubmitSaga(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("SubmitSaga() status = %d, want %d", w.Code, http.StatusCreated)
	}
	var submitResp models.SagaSubmitResponse
	_ = json.NewDecoder(w.Body).Decode(&submitResp)
	time.Sleep(80 * time.Millisecond)

	// manual compensate should be accepted
	compReq := httptest.NewRequest(http.MethodPost, "/api/v1/sagas/"+submitResp.SagaID+"/compensate", bytes.NewReader([]byte(`{"reason":"manual"}`)))
	compCtx := chi.NewRouteContext()
	compCtx.URLParams.Add("id", submitResp.SagaID)
	compReq = compReq.WithContext(context.WithValue(compReq.Context(), chi.RouteCtxKey, compCtx))
	compW := httptest.NewRecorder()
	handler.CompensateSaga(compW, compReq)
	if compW.Code != http.StatusAccepted {
		t.Fatalf("CompensateSaga() status = %d, want %d, body=%s", compW.Code, http.StatusAccepted, compW.Body.String())
	}

	// recover terminal saga should return conflict
	_ = checkpointStore.Save(context.Background(), &saga.Checkpoint{
		DefinitionName: "manual-compensate",
		SagaID:         submitResp.SagaID,
		State:          saga.SagaStateCompleted,
		LastUpdated:    time.Now().UTC(),
	})
	recReq := httptest.NewRequest(http.MethodPost, "/api/v1/sagas/"+submitResp.SagaID+"/recover", nil)
	recCtx := chi.NewRouteContext()
	recCtx.URLParams.Add("id", submitResp.SagaID)
	recReq = recReq.WithContext(context.WithValue(recReq.Context(), chi.RouteCtxKey, recCtx))
	recW := httptest.NewRecorder()
	handler.RecoverSaga(recW, recReq)
	if recW.Code != http.StatusConflict {
		t.Fatalf("RecoverSaga() status = %d, want %d", recW.Code, http.StatusConflict)
	}
}
