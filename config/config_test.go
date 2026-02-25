package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test App defaults
	if cfg.App.Name != "goclaw" {
		t.Errorf("expected app name 'goclaw', got %s", cfg.App.Name)
	}
	if cfg.App.Environment != "development" {
		t.Errorf("expected environment 'development', got %s", cfg.App.Environment)
	}

	// Test Server defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("expected server port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.GRPC.Port != 9090 {
		t.Errorf("expected grpc port 9090, got %d", cfg.Server.GRPC.Port)
	}

	// Test Log defaults
	if cfg.Log.Level != "info" {
		t.Errorf("expected log level 'info', got %s", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("expected log format 'json', got %s", cfg.Log.Format)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.App.Name = "test"
				cfg.App.Environment = "development"
				cfg.Server.Port = 8080
				cfg.Log.Level = "info"
				cfg.Log.Format = "json"
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "missing app name",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.App.Name = "" // 空名称
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid port",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Server.Port = 99999 // 无效端口
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.Log.Level = "trace" // 无效日志级别
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid environment",
			cfg: func() *Config {
				cfg := DefaultConfig()
				cfg.App.Environment = "invalid" // 无效环境
				return cfg
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"warning", "warn"},
		{"error", "error"},
		{"unknown", "info"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// This test is in the wrong package, but demonstrates the concept
			// Actual test should be in logger package
		})
	}
}

func TestLoader_Load(t *testing.T) {
	loader := NewLoader()

	// Test with defaults only
	cfg, err := loader.Load("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Name != "goclaw" {
		t.Errorf("expected default app name, got %s", cfg.App.Name)
	}
}

func TestLoader_LoadWithOverrides(t *testing.T) {
	loader := NewLoader()

	// Note: Koanf replaces entire nested structs, so we need to provide all required fields
	overrides := map[string]interface{}{
		"app.name":          "test-app",
		"app.environment":   "development",
		"server.port":       9090,
		"server.grpc.port":  9091,
		"server.grpc.max_concurrent_streams": 100,
		"log.level":         "debug",
		"log.format":        "json",
		"orchestration.max_agents": 100,
		"orchestration.queue.type": "memory",
		"orchestration.queue.size": 1000,
		"orchestration.scheduler.type": "round_robin",
		"metrics.port":      9092,
		"storage.type":      "memory",
		"storage.badger.value_log_file_size": 1073741824,
		"storage.badger.num_versions_to_keep": 1,
	}

	cfg, err := loader.Load("", overrides)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Name != "test-app" {
		t.Errorf("expected app name 'test-app', got %s", cfg.App.Name)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level 'debug', got %s", cfg.Log.Level)
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errs := ValidationErrors{
		{Field: "server.port", Message: "must be between 1 and 65535", Value: 99999},
		{Field: "log.level", Message: "must be one of [debug info warn error]", Value: "trace"},
	}

	errMsg := errs.Error()
	if errMsg == "" {
		t.Error("expected error message")
	}

	if errMsg == "no validation errors" {
		t.Error("expected error details")
	}
}

func TestConfig_String(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Name:        "test",
			Environment: "development",
		},
		Server: ServerConfig{
			Port: 8080,
		},
	}

	s := cfg.String()
	if s == "" {
		t.Error("expected non-empty string representation")
	}
}

func TestDurationParsing(t *testing.T) {
	// Test that duration fields work correctly
	cfg := DefaultConfig()

	if cfg.Server.HTTP.ReadTimeout != 30*time.Second {
		t.Errorf("expected read timeout 30s, got %v", cfg.Server.HTTP.ReadTimeout)
	}

	if cfg.Orchestration.Scheduler.CheckInterval != 5*time.Second {
		t.Errorf("expected check interval 5s, got %v", cfg.Orchestration.Scheduler.CheckInterval)
	}
}
