package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	// EnvPrefix is the prefix for environment variables.
	EnvPrefix = "GOCLAW_"
	// Delimiter is the key delimiter for nested config.
	Delimiter = "."
)

// Loader handles configuration loading from various sources.
type Loader struct {
	k      *koanf.Koanf
	parser koanf.Parser
}

// NewLoader creates a new configuration loader.
func NewLoader() *Loader {
	return &Loader{
		k: koanf.New(Delimiter),
	}
}

// Load loads configuration from all sources with the following priority:
// 1. Command line flags (highest)
// 2. Environment variables
// 3. Configuration files
// 4. Defaults (lowest)
func (l *Loader) Load(configPath string, overrides map[string]interface{}) (*Config, error) {
	// 1. Load defaults
	if err := l.loadDefaults(); err != nil {
		return nil, fmt.Errorf("failed to load defaults: %w", err)
	}

	// 2. Load from file if specified
	if configPath != "" {
		if err := l.loadFile(configPath); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	} else {
		// Try to find config in standard locations
		l.loadDefaultFiles()
	}

	// 3. Load from environment variables
	if err := l.loadEnv(); err != nil {
		return nil, fmt.Errorf("failed to load env vars: %w", err)
	}

	// 4. Apply command line overrides (merge, not replace)
	if len(overrides) > 0 {
		if err := l.k.Load(confmap.Provider(overrides, Delimiter), nil); err != nil {
			return nil, fmt.Errorf("failed to apply overrides: %w", err)
		}
	}
	
	// Workaround: Koanf replaces nested structs, so we need to reload defaults
	// for fields that weren't overridden. We do this by checking if critical
	// fields are zero and re-applying defaults.
	if err := l.fillDefaults(); err != nil {
		return nil, fmt.Errorf("failed to fill defaults: %w", err)
	}

	// Unmarshal to struct
	var cfg Config
	if err := l.k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "mapstructure",
	}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate
	if err := ValidateWithDetails(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// loadDefaults loads the default configuration.
func (l *Loader) loadDefaults() error {
	defaults := DefaultConfig()
	return l.k.Load(confmap.Provider(map[string]interface{}{
		"app":           defaults.App,
		"server":        defaults.Server,
		"log":           defaults.Log,
		"orchestration": defaults.Orchestration,
		"cluster":       defaults.Cluster,
		"storage":       defaults.Storage,
		"metrics":       defaults.Metrics,
		"tracing":       defaults.Tracing,
	}, Delimiter), nil)
}

// loadFile loads configuration from a file.
func (l *Loader) loadFile(path string) error {
	// Determine parser based on extension
	ext := strings.ToLower(filepath.Ext(path))
	var parser koanf.Parser

	switch ext {
	case ".yaml", ".yml":
		parser = yaml.Parser()
	case ".json":
		parser = json.Parser()
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", path)
	}

	return l.k.Load(file.Provider(path), parser)
}

// loadDefaultFiles tries to load config from standard locations.
func (l *Loader) loadDefaultFiles() {
	// Try these locations in order
	candidates := []string{
		"config.yaml",
		"config.yml",
		"config.json",
		"configs/config.yaml",
		"/etc/goclaw/config.yaml",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			_ = l.loadFile(path) // Ignore error, try next
			return
		}
	}
}

// loadEnv loads configuration from environment variables.
func (l *Loader) loadEnv() error {
	return l.k.Load(env.Provider(EnvPrefix, Delimiter, func(s string) string {
		// Transform environment variable names
		// GOCLAW_SERVER_PORT -> server.port
		// GOCLAW_LOG_LEVEL -> log.level
		return strings.ToLower(strings.TrimPrefix(s, EnvPrefix))
	}), nil)
}

// Get returns a configuration value by key.
func (l *Loader) Get(key string) interface{} {
	return l.k.Get(key)
}

// GetString returns a string configuration value.
func (l *Loader) GetString(key string) string {
	return l.k.String(key)
}

// GetInt returns an int configuration value.
func (l *Loader) GetInt(key string) int {
	return l.k.Int(key)
}

// GetBool returns a bool configuration value.
func (l *Loader) GetBool(key string) bool {
	return l.k.Bool(key)
}

// Set sets a configuration value.
func (l *Loader) Set(key string, value interface{}) error {
	return l.k.Set(key, value)
}

// fillDefaults fills in default values for any zero-value critical fields.
func (l *Loader) fillDefaults() error {
	defaults := DefaultConfig()
	
	// Helper to set default if key is not set or is zero
	setIfZero := func(key string, defaultVal interface{}) {
		if l.k.Get(key) == nil {
			l.k.Set(key, defaultVal)
		}
	}
	
	// App defaults
	setIfZero("app.name", defaults.App.Name)
	setIfZero("app.version", defaults.App.Version)
	setIfZero("app.environment", defaults.App.Environment)
	
	// Server defaults
	setIfZero("server.host", defaults.Server.Host)
	setIfZero("server.grpc.port", defaults.Server.GRPC.Port)
	setIfZero("server.grpc.max_concurrent_streams", defaults.Server.GRPC.MaxConcurrentStreams)
	setIfZero("server.http.read_timeout", defaults.Server.HTTP.ReadTimeout)
	setIfZero("server.http.write_timeout", defaults.Server.HTTP.WriteTimeout)
	setIfZero("server.http.idle_timeout", defaults.Server.HTTP.IdleTimeout)
	setIfZero("server.http.max_header_bytes", defaults.Server.HTTP.MaxHeaderBytes)
	
	// Log defaults
	setIfZero("log.format", defaults.Log.Format)
	setIfZero("log.output", defaults.Log.Output)
	
	// Orchestration defaults
	setIfZero("orchestration.max_agents", defaults.Orchestration.MaxAgents)
	setIfZero("orchestration.queue.type", defaults.Orchestration.Queue.Type)
	setIfZero("orchestration.queue.size", defaults.Orchestration.Queue.Size)
	setIfZero("orchestration.scheduler.type", defaults.Orchestration.Scheduler.Type)
	setIfZero("orchestration.scheduler.check_interval", defaults.Orchestration.Scheduler.CheckInterval)
	
	// Storage defaults
	setIfZero("storage.type", defaults.Storage.Type)
	
	// Metrics defaults
	setIfZero("metrics.enabled", defaults.Metrics.Enabled)
	setIfZero("metrics.path", defaults.Metrics.Path)
	setIfZero("metrics.port", defaults.Metrics.Port)
	
	return nil
}

// Print prints the loaded configuration for debugging.
func (l *Loader) Print() string {
	return l.k.Sprint()
}

// Load is a convenience function to load configuration.
func Load(configPath string, overrides map[string]interface{}) (*Config, error) {
	loader := NewLoader()
	return loader.Load(configPath, overrides)
}

// LoadOrDie loads configuration and panics on error.
func LoadOrDie(configPath string, overrides map[string]interface{}) *Config {
	cfg, err := Load(configPath, overrides)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}
