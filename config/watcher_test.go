package config

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	loader := NewLoader()

	t.Run("valid config path", func(t *testing.T) {
		// Create a temp config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("app:\n  name: test\n"), 0644); err != nil {
			t.Fatalf("failed to create temp config: %v", err)
		}

		watcher, err := NewWatcher(configPath, loader)
		if err != nil {
			t.Fatalf("NewWatcher failed: %v", err)
		}
		defer watcher.Stop()

		if watcher == nil {
			t.Fatal("expected non-nil watcher")
		}
		if watcher.ConfigPath() != configPath {
			t.Errorf("expected config path %s, got %s", configPath, watcher.ConfigPath())
		}
	})

	t.Run("empty config path", func(t *testing.T) {
		_, err := NewWatcher("", loader)
		if err == nil {
			t.Fatal("expected error for empty config path")
		}
	})

	t.Run("with debounce option", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("app:\n  name: test\n"), 0644); err != nil {
			t.Fatalf("failed to create temp config: %v", err)
		}

		watcher, err := NewWatcher(configPath, loader, WithDebounce(100*time.Millisecond))
		if err != nil {
			t.Fatalf("NewWatcher failed: %v", err)
		}
		defer watcher.Stop()

		if watcher.debounce != 100*time.Millisecond {
			t.Errorf("expected debounce 100ms, got %v", watcher.debounce)
		}
	})
}

func TestWatcher_Watch(t *testing.T) {
	loader := NewLoader()

	t.Run("detects file changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Create initial config
		initialContent := `app:
  name: test-app
server:
  port: 8080
log:
  level: info
  format: json
`
		if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
			t.Fatalf("failed to create temp config: %v", err)
		}

		watcher, err := NewWatcher(configPath, loader)
		if err != nil {
			t.Fatalf("NewWatcher failed: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var callbackCalled bool
		var callbackMu sync.Mutex
		var receivedConfig *Config

		watcher.OnChange(func(cfg *Config) {
			callbackMu.Lock()
			defer callbackMu.Unlock()
			callbackCalled = true
			receivedConfig = cfg
		})

		// Start watching in a goroutine
		watchErr := make(chan error, 1)
		go func() {
			watchErr <- watcher.Watch(ctx)
		}()

		// Wait a bit for watcher to start
		time.Sleep(100 * time.Millisecond)

		// Modify the config file
		updatedContent := `app:
  name: updated-app
server:
  port: 8080
log:
  level: debug
  format: json
`
		if err := os.WriteFile(configPath, []byte(updatedContent), 0644); err != nil {
			t.Fatalf("failed to update temp config: %v", err)
		}

		// Wait for callback to be called
		time.Sleep(600 * time.Millisecond)

		callbackMu.Lock()
		if !callbackCalled {
			t.Error("expected callback to be called after config change")
		}
		if receivedConfig != nil && receivedConfig.Log.Level != "debug" {
			t.Errorf("expected log level 'debug', got '%s'", receivedConfig.Log.Level)
		}
		callbackMu.Unlock()

		watcher.Stop()
	})

	t.Run("stops on context cancel", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("app:\n  name: test\n"), 0644); err != nil {
			t.Fatalf("failed to create temp config: %v", err)
		}

		watcher, err := NewWatcher(configPath, loader)
		if err != nil {
			t.Fatalf("NewWatcher failed: %v", err)
		}
		defer watcher.Stop()

		ctx, cancel := context.WithCancel(context.Background())

		watchErr := make(chan error, 1)
		go func() {
			watchErr <- watcher.Watch(ctx)
		}()

		// Cancel context
		time.Sleep(100 * time.Millisecond)
		cancel()

		select {
		case err := <-watchErr:
			if err != context.Canceled {
				t.Errorf("expected context.Canceled, got %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("watcher did not stop on context cancel")
		}
	})

	t.Run("prevents double watch", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("app:\n  name: test\n"), 0644); err != nil {
			t.Fatalf("failed to create temp config: %v", err)
		}

		watcher, err := NewWatcher(configPath, loader)
		if err != nil {
			t.Fatalf("NewWatcher failed: %v", err)
		}
		defer watcher.Stop()

		ctx := context.Background()

		// Start first watch
		go func() {
			watcher.Watch(ctx)
		}()

		// Wait for watcher to start
		time.Sleep(100 * time.Millisecond)

		// Try to start second watch - should fail
		err = watcher.Watch(ctx)
		if err == nil {
			t.Error("expected error when starting double watch")
		}
	})
}

func TestWatcher_OnChange(t *testing.T) {
	loader := NewLoader()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("app:\n  name: test\n"), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	watcher, err := NewWatcher(configPath, loader)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer watcher.Stop()

	// Register multiple callbacks
	var callCount int
	var mu sync.Mutex

	watcher.OnChange(func(cfg *Config) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})
	watcher.OnChange(func(cfg *Config) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	// Trigger reload manually
	watcher.reloadConfig(context.Background())

	// Wait for goroutines
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if callCount != 2 {
		t.Errorf("expected 2 callback calls, got %d", callCount)
	}
	mu.Unlock()
}

func TestWatcher_Stop(t *testing.T) {
	loader := NewLoader()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("app:\n  name: test\n"), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	watcher, err := NewWatcher(configPath, loader)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Start watching
	ctx := context.Background()
	go func() {
		watcher.Watch(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	if !watcher.IsRunning() {
		t.Error("expected watcher to be running")
	}

	// Stop the watcher
	if err := watcher.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Wait for watcher to stop
	time.Sleep(100 * time.Millisecond)

	if watcher.IsRunning() {
		t.Error("expected watcher to not be running after Stop")
	}
}

func TestWatcher_NonExistentFile(t *testing.T) {
	loader := NewLoader()

	watcher, err := NewWatcher("/nonexistent/config.yaml", loader)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer watcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Watch should fail because file doesn't exist
	err = watcher.Watch(ctx)
	if err == nil {
		t.Error("expected error when watching non-existent file")
	}
}

func TestHotReloadableConfig(t *testing.T) {
	t.Run("ExtractHotReloadable", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Log.Level = "debug"
		cfg.Log.Format = "text"
		cfg.Metrics.Enabled = false
		cfg.Metrics.Path = "/custom-metrics"
		cfg.Metrics.Port = 9999

		hot := ExtractHotReloadable(cfg)

		if hot.LogLevel != "debug" {
			t.Errorf("expected log level 'debug', got '%s'", hot.LogLevel)
		}
		if hot.LogFormat != "text" {
			t.Errorf("expected log format 'text', got '%s'", hot.LogFormat)
		}
		if hot.MetricsEnabled != false {
			t.Errorf("expected metrics enabled false, got %v", hot.MetricsEnabled)
		}
		if hot.MetricsPath != "/custom-metrics" {
			t.Errorf("expected metrics path '/custom-metrics', got '%s'", hot.MetricsPath)
		}
		if hot.MetricsPort != 9999 {
			t.Errorf("expected metrics port 9999, got %d", hot.MetricsPort)
		}
	})

	t.Run("Changed detects differences", func(t *testing.T) {
		h1 := HotReloadableConfig{
			LogLevel:       "info",
			LogFormat:      "json",
			MetricsEnabled: true,
			MetricsPath:    "/metrics",
			MetricsPort:    9091,
		}

		t.Run("no changes", func(t *testing.T) {
			h2 := h1
			if h1.Changed(h2) {
				t.Error("expected no change detected")
			}
		})

		t.Run("log level changed", func(t *testing.T) {
			h2 := h1
			h2.LogLevel = "debug"
			if !h1.Changed(h2) {
				t.Error("expected change detected for log level")
			}
		})

		t.Run("log format changed", func(t *testing.T) {
			h2 := h1
			h2.LogFormat = "text"
			if !h1.Changed(h2) {
				t.Error("expected change detected for log format")
			}
		})

		t.Run("metrics enabled changed", func(t *testing.T) {
			h2 := h1
			h2.MetricsEnabled = false
			if !h1.Changed(h2) {
				t.Error("expected change detected for metrics enabled")
			}
		})

		t.Run("metrics port changed", func(t *testing.T) {
			h2 := h1
			h2.MetricsPort = 8080
			if !h1.Changed(h2) {
				t.Error("expected change detected for metrics port")
			}
		})
	})
}
