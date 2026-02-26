package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors configuration file changes and triggers callbacks.
type Watcher struct {
	mu         sync.RWMutex
	watcher    *fsnotify.Watcher
	loader     *Loader
	configPath string
	callbacks  []func(*Config)
	debounce   time.Duration
	stopCh     chan struct{}
	running    bool
}

// WatcherOption is a functional option for Watcher configuration.
type WatcherOption func(*Watcher)

// WithDebounce sets the debounce duration for file change events.
func WithDebounce(d time.Duration) WatcherOption {
	return func(w *Watcher) {
		w.debounce = d
	}
}

// NewWatcher creates a new configuration file watcher.
func NewWatcher(configPath string, loader *Loader, opts ...WatcherOption) (*Watcher, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path is required for watching")
	}

	fswatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		watcher:    fswatcher,
		loader:     loader,
		configPath: configPath,
		debounce:   500 * time.Millisecond, // Default debounce
		stopCh:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w, nil
}

// Watch starts monitoring the configuration file for changes.
// It blocks until the context is cancelled or Stop is called.
func (w *Watcher) Watch(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watcher is already running")
	}
	w.running = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
	}()

	// Add the config file to the watcher
	if err := w.watcher.Add(w.configPath); err != nil {
		return fmt.Errorf("failed to watch config file %s: %w", w.configPath, err)
	}

	// Debounce timer
	var debounceTimer *time.Timer
	var lastEvent time.Time

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-w.stopCh:
			return nil

		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}

			// Only handle write and create events
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create {
				now := time.Now()

				// Debounce: reset timer on each event
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				// Only process if enough time has passed since last event
				if now.Sub(lastEvent) < w.debounce {
					lastEvent = now
					debounceTimer = time.AfterFunc(w.debounce, func() {
						w.reloadConfig(ctx)
					})
					continue
				}

				lastEvent = now
				w.reloadConfig(ctx)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			// Log error but continue watching
			fmt.Printf("config watcher error: %v\n", err)
		}
	}
}

// reloadConfig reloads the configuration and notifies callbacks.
func (w *Watcher) reloadConfig(ctx context.Context) {
	cfg, err := w.loader.Load(w.configPath, nil)
	if err != nil {
		fmt.Printf("failed to reload config: %v\n", err)
		return
	}

	// Notify all registered callbacks
	w.mu.RLock()
	callbacks := make([]func(*Config), len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.RUnlock()

	for _, cb := range callbacks {
		// Run callbacks in goroutines to avoid blocking
		go func(callback func(*Config)) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("config callback panic: %v\n", r)
				}
			}()
			callback(cfg)
		}(cb)
	}
}

// OnChange registers a callback to be called when the configuration changes.
// Callbacks are called concurrently in separate goroutines.
func (w *Watcher) OnChange(callback func(*Config)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// Stop stops the watcher and releases resources.
func (w *Watcher) Stop() error {
	close(w.stopCh)
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

// IsRunning returns whether the watcher is currently running.
func (w *Watcher) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// ConfigPath returns the path being watched.
func (w *Watcher) ConfigPath() string {
	return w.configPath
}

// HotReloadableConfig contains configuration values that can be hot-reloaded.
type HotReloadableConfig struct {
	LogLevel       string
	LogFormat      string
	MetricsEnabled bool
	MetricsPath    string
	MetricsPort    int
}

// ExtractHotReloadable extracts hot-reloadable values from Config.
func ExtractHotReloadable(cfg *Config) HotReloadableConfig {
	return HotReloadableConfig{
		LogLevel:       cfg.Log.Level,
		LogFormat:      cfg.Log.Format,
		MetricsEnabled: cfg.Metrics.Enabled,
		MetricsPath:    cfg.Metrics.Path,
		MetricsPort:    cfg.Metrics.Port,
	}
}

// Changed checks if hot-reloadable configuration has changed.
func (h HotReloadableConfig) Changed(other HotReloadableConfig) bool {
	return h.LogLevel != other.LogLevel ||
		h.LogFormat != other.LogFormat ||
		h.MetricsEnabled != other.MetricsEnabled ||
		h.MetricsPath != other.MetricsPath ||
		h.MetricsPort != other.MetricsPort
}
