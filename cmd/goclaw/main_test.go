package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/engine"
	grpcpkg "github.com/goclaw/goclaw/pkg/grpc"
	grpchandlers "github.com/goclaw/goclaw/pkg/grpc/handlers"
	grpcstreaming "github.com/goclaw/goclaw/pkg/grpc/streaming"
	"github.com/goclaw/goclaw/pkg/logger"
	signalpkg "github.com/goclaw/goclaw/pkg/signal"
	"github.com/goclaw/goclaw/pkg/storage"
)

// mockStorage is a minimal mock implementation for testing
type mockStorage struct{}

func (m *mockStorage) SaveWorkflow(ctx context.Context, wf *storage.WorkflowState) error {
	return nil
}

func (m *mockStorage) GetWorkflow(ctx context.Context, id string) (*storage.WorkflowState, error) {
	return nil, &storage.NotFoundError{EntityType: "workflow", ID: id}
}

func (m *mockStorage) ListWorkflows(ctx context.Context, filter *storage.WorkflowFilter) ([]*storage.WorkflowState, int, error) {
	return nil, 0, nil
}

func (m *mockStorage) DeleteWorkflow(ctx context.Context, id string) error {
	return nil
}

func (m *mockStorage) SaveTask(ctx context.Context, workflowID string, task *storage.TaskState) error {
	return nil
}

func (m *mockStorage) GetTask(ctx context.Context, workflowID, taskID string) (*storage.TaskState, error) {
	return nil, &storage.NotFoundError{EntityType: "task", ID: taskID}
}

func (m *mockStorage) ListTasks(ctx context.Context, workflowID string) ([]*storage.TaskState, error) {
	return nil, nil
}

func (m *mockStorage) Close() error {
	return nil
}

func TestServerStartup(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "test",
		},
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 18080, // Use different port for testing
			HTTP: config.HTTPConfig{
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
			},
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"*"},
			},
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents: 10,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	// Initialize logger
	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	// Create and start engine
	ctx := context.Background()

	// Create in-memory storage for testing
	store := &mockStorage{}

	eng, err := engine.New(cfg, log, store)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer eng.Stop(ctx)

	// Initialize HTTP server with handlers
	workflowHandler := handlers.NewWorkflowHandler(eng, log)
	healthHandler := handlers.NewHealthHandler(eng)
	wsHandler := handlers.NewWebSocketHandler(log, handlers.WebSocketConfig{
		AllowedOrigins: cfg.Server.CORS.AllowedOrigins,
		MaxConnections: 10,
		PingInterval:   30 * time.Second,
		PongTimeout:    10 * time.Second,
	})

	apiHandlers := &api.Handlers{
		Workflow:  workflowHandler,
		Health:    healthHandler,
		WebSocket: wsHandler,
	}

	httpServer := api.NewHTTPServer(cfg, log, apiHandlers)

	// Start HTTP server in a separate goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Check if server started without errors
	select {
	case err := <-serverErrChan:
		t.Fatalf("Server failed to start: %v", err)
	default:
		// Server started successfully
	}

	// Test health endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", cfg.Server.Port))
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health endpoint returned status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test ready endpoint
	resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/ready", cfg.Server.Port))
	if err != nil {
		t.Fatalf("Failed to call ready endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ready endpoint returned status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test status endpoint
	resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/status", cfg.Server.Port))
	if err != nil {
		t.Fatalf("Failed to call status endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status endpoint returned status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Validate websocket route registration in startup path.
	wsResp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/ws/events", cfg.Server.Port))
	if err != nil {
		t.Fatalf("Failed to call websocket endpoint: %v", err)
	}
	defer wsResp.Body.Close()
	if wsResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Websocket endpoint status = %d, want %d", wsResp.StatusCode, http.StatusBadRequest)
	}

	// Graceful shutdown sequence aligns with main: close websocket manager first, then HTTP server.
	wsHandler.Close()

	// Graceful shutdown.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}
}

func TestServerStartup_WithSagaEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.App.Name = "test-saga"
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 18082
	cfg.Storage.Type = "memory"
	cfg.Storage.Badger.Path = t.TempDir()
	cfg.Saga.Enabled = true
	cfg.Saga.WALCleanupInterval = 50 * time.Millisecond
	cfg.Saga.WALRetention = 24 * time.Hour

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	ctx := context.Background()
	store := &mockStorage{}
	eng, err := engine.New(cfg, log, store)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer eng.Stop(ctx)

	sagaOrchestrator := eng.GetSagaOrchestrator()
	if sagaOrchestrator == nil {
		t.Fatal("expected saga orchestrator to be initialized")
	}

	sagaHandler := handlers.NewSagaHandler(
		sagaOrchestrator,
		eng.GetSagaCheckpointStore(),
		eng.GetSagaRecoveryManager(),
		log,
	)
	workflowHandler := handlers.NewWorkflowHandler(eng, log)
	healthHandler := handlers.NewHealthHandler(eng)
	wsHandler := handlers.NewWebSocketHandler(log, handlers.WebSocketConfig{
		AllowedOrigins: cfg.Server.CORS.AllowedOrigins,
		MaxConnections: 10,
		PingInterval:   30 * time.Second,
		PongTimeout:    10 * time.Second,
	})
	defer wsHandler.Close()

	apiHandlers := &api.Handlers{
		Workflow:  workflowHandler,
		Health:    healthHandler,
		Saga:      sagaHandler,
		WebSocket: wsHandler,
	}
	httpServer := api.NewHTTPServer(cfg, log, apiHandlers)

	serverErrChan := make(chan error, 1)
	go func() {
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()
	time.Sleep(100 * time.Millisecond)

	reqBody := models.SagaSubmitRequest{
		Name: "startup-saga",
		Steps: []models.SagaStepRequest{
			{ID: "a"},
		},
	}
	payload, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/api/v1/sagas", cfg.Server.Port),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("Failed to submit saga: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Submit saga status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	grpcServer, err := grpcpkg.New(cfg.Server.GRPC.ToGRPCConfig())
	if err != nil {
		t.Fatalf("failed to create grpc server: %v", err)
	}
	bus := signalpkg.NewLocalBus(16)
	defer bus.Close()
	sagaSvc := grpchandlers.NewSagaServiceServer(sagaOrchestrator, eng.GetSagaCheckpointStore())
	if err := registerGRPCServices(grpcServer, eng, bus, grpcstreaming.NewSubscriberRegistry(), sagaSvc); err != nil {
		t.Fatalf("registerGRPCServices() error = %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Failed to shutdown server: %v", err)
	}
}

func TestBuildOverrides(t *testing.T) {
	// Save original values
	origAppName := *appName
	origServerPort := *serverPort
	origLogLevel := *logLevel
	origDebugMode := *debugMode

	// Restore original values after test
	defer func() {
		*appName = origAppName
		*serverPort = origServerPort
		*logLevel = origLogLevel
		*debugMode = origDebugMode
	}()

	// Test with no overrides
	*appName = ""
	*serverPort = 0
	*logLevel = ""
	*debugMode = false

	overrides := buildOverrides()
	if len(overrides) != 0 {
		t.Errorf("Expected empty overrides, got %d items", len(overrides))
	}

	// Test with all overrides
	*appName = "test-app"
	*serverPort = 9090
	*logLevel = "debug"
	*debugMode = true

	overrides = buildOverrides()
	if len(overrides) != 4 {
		t.Errorf("Expected 4 overrides, got %d", len(overrides))
	}

	if overrides["app.name"] != "test-app" {
		t.Errorf("Expected app.name=test-app, got %v", overrides["app.name"])
	}
	if overrides["server.port"] != 9090 {
		t.Errorf("Expected server.port=9090, got %v", overrides["server.port"])
	}
	if overrides["log.level"] != "debug" {
		t.Errorf("Expected log.level=debug, got %v", overrides["log.level"])
	}
	if overrides["app.debug"] != true {
		t.Errorf("Expected app.debug=true, got %v", overrides["app.debug"])
	}
}

func TestPrintVersion(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printVersion()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check if output contains expected strings
	expectedStrings := []string{"Goclaw", "Version:", "Build Time:", "Git Commit:", "Go Version:"}
	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't. Output: %s", expected, output)
		}
	}
}

func TestPrintHelp(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printHelp()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check if output contains expected strings
	expectedStrings := []string{"Goclaw", "Usage:", "Options:", "Examples:"}
	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't. Output: %s", expected, output)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestInitializeSignalBus_LocalMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Signal.Mode = "local"

	bus, mode := initializeSignalBus(cfg, nil, nil)
	if mode != "local" {
		t.Fatalf("expected mode local, got %s", mode)
	}
	if !bus.Healthy() {
		t.Fatal("expected local bus to be healthy")
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("failed to close local bus: %v", err)
	}
}

func TestInitializeSignalBus_RedisFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Signal.Mode = "redis"

	bus, mode := initializeSignalBus(cfg, nil, nil)
	if mode != "local(fallback)" {
		t.Fatalf("expected fallback mode, got %s", mode)
	}
	if _, ok := bus.(*signalpkg.LocalBus); !ok {
		t.Fatalf("expected LocalBus fallback, got %T", bus)
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("failed to close fallback bus: %v", err)
	}
}

func TestInitializeRedisClient_NilConfig(t *testing.T) {
	_, err := initializeRedisClient(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestInitializeRedisClient_InvalidAddress(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Redis.Address = "127.0.0.1:0"
	cfg.Redis.DialTimeout = 10 * time.Millisecond
	cfg.Redis.ReadTimeout = 10 * time.Millisecond
	cfg.Redis.WriteTimeout = 10 * time.Millisecond

	client, err := initializeRedisClient(context.Background(), cfg)
	if err == nil {
		if client != nil {
			_ = client.Close()
		}
		t.Fatal("expected redis initialization error")
	}
}

func TestEngineStartupShutdown_WithDistributedFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Orchestration.Queue.Type = "redis"
	cfg.Signal.Mode = "redis"
	cfg.Redis.Enabled = true
	cfg.Redis.Address = "127.0.0.1:0"
	cfg.Redis.DialTimeout = 10 * time.Millisecond
	cfg.Redis.ReadTimeout = 10 * time.Millisecond
	cfg.Redis.WriteTimeout = 10 * time.Millisecond

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	redisClient, err := initializeRedisClient(context.Background(), cfg)
	if err == nil {
		if redisClient != nil {
			_ = redisClient.Close()
		}
		t.Fatal("expected redis initialization to fail for invalid address")
	}

	bus, mode := initializeSignalBus(cfg, nil, log)
	if mode != "local(fallback)" {
		t.Fatalf("expected local fallback mode, got %s", mode)
	}
	defer bus.Close()

	eng, err := engine.New(cfg, log, &mockStorage{}, engine.WithSignalBus(bus))
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("failed to start engine with fallback: %v", err)
	}

	if err := eng.Stop(ctx); err != nil {
		t.Fatalf("failed to stop engine with fallback: %v", err)
	}
}

func TestRegisterGRPCServices_MissingWiring(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.GRPC.Enabled = true
	grpcServer, err := grpcpkg.New(cfg.Server.GRPC.ToGRPCConfig())
	if err != nil {
		t.Fatalf("failed to create grpc server: %v", err)
	}

	eng, err := engine.New(cfg, logger.New(&logger.Config{Level: logger.InfoLevel, Format: "json", Output: "stdout"}), &mockStorage{})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	err = registerGRPCServices(grpcServer, eng, signalpkg.NewLocalBus(16), nil, nil)
	if err == nil {
		t.Fatal("expected missing streaming registry error")
	}
}

func TestRegisterGRPCServices_Success(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.GRPC.Enabled = true
	grpcServer, err := grpcpkg.New(cfg.Server.GRPC.ToGRPCConfig())
	if err != nil {
		t.Fatalf("failed to create grpc server: %v", err)
	}

	eng, err := engine.New(cfg, logger.New(&logger.Config{Level: logger.InfoLevel, Format: "json", Output: "stdout"}), &mockStorage{})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	bus := signalpkg.NewLocalBus(16)
	defer bus.Close()

	if err := registerGRPCServices(grpcServer, eng, bus, grpcstreaming.NewSubscriberRegistry(), nil); err != nil {
		t.Fatalf("registerGRPCServices() error = %v", err)
	}
}

func TestInitGRPCTracing_DisabledByConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.GRPC.Enabled = true
	cfg.Server.GRPC.EnableTracing = false
	cfg.Tracing.Enabled = true

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	shutdown, err := initGRPCTracing(context.Background(), cfg, log)
	if err != nil {
		t.Fatalf("initGRPCTracing() error = %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestInitGRPCTracing_InvalidConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.GRPC.Enabled = true
	cfg.Server.GRPC.EnableTracing = true
	cfg.Tracing.Enabled = true
	cfg.Tracing.Endpoint = ""

	_, err := initGRPCTracing(context.Background(), cfg, nil)
	if err == nil {
		t.Fatal("expected initGRPCTracing to fail for invalid tracing config")
	}
}

func TestSummarizeTracingEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{name: "raw host", endpoint: "localhost:4317", want: "localhost:4317"},
		{name: "with scheme and path", endpoint: "http://collector:4317/v1/traces", want: "collector:4317"},
		{name: "with spaces", endpoint: "  https://collector.internal:4318/otel  ", want: "collector.internal:4318"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summarizeTracingEndpoint(tt.endpoint); got != tt.want {
				t.Fatalf("summarizeTracingEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}
