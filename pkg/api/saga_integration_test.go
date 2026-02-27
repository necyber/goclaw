package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dgbadger "github.com/dgraph-io/badger/v4"
	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/logger"
	"github.com/goclaw/goclaw/pkg/saga"
)

func TestSagaEndpointsIntegration(t *testing.T) {
	opts := dgbadger.DefaultOptions(t.TempDir())
	opts.Logger = nil
	db, err := dgbadger.Open(opts)
	if err != nil {
		t.Fatalf("open badger: %v", err)
	}
	defer db.Close()

	wal, err := saga.NewBadgerWAL(db, saga.WALOptions{WriteMode: saga.WALWriteModeSync})
	if err != nil {
		t.Fatalf("new wal: %v", err)
	}
	defer wal.Close()

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
	recoveryManager, err := saga.NewRecoveryManager(orchestrator, checkpointStore, nil)
	if err != nil {
		t.Fatalf("new recovery manager: %v", err)
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	cfg := config.DefaultConfig()
	httpHandlers := &Handlers{
		Saga: handlers.NewSagaHandler(orchestrator, checkpointStore, recoveryManager, log),
	}
	router := NewRouter(cfg, log, httpHandlers)

	submitReq := models.SagaSubmitRequest{
		Name: "integration-saga",
		Steps: []models.SagaStepRequest{
			{ID: "a"},
		},
	}
	body, _ := json.Marshal(submitReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sagas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("submit status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}

	var submitResp models.SagaSubmitResponse
	if err := json.NewDecoder(w.Body).Decode(&submitResp); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/sagas/"+submitResp.SagaID, nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getW.Code, http.StatusOK)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/sagas?state=completed", nil)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listW.Code, http.StatusOK)
	}

	// ensure route-level recover endpoint returns not found if no checkpoint
	recReq := httptest.NewRequest(http.MethodPost, "/api/v1/sagas/missing/recover", nil)
	recW := httptest.NewRecorder()
	router.ServeHTTP(recW, recReq)
	if recW.Code != http.StatusNotFound {
		t.Fatalf("recover missing status = %d, want %d", recW.Code, http.StatusNotFound)
	}

	// touch recovery manager to ensure wiring remains valid
	_, _ = recoveryManager.Recover(context.Background(), map[string]*saga.SagaDefinition{}, nil)
}
