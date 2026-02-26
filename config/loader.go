package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	k *koanf.Koanf
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
		"memory":        defaults.Memory,
		"redis":         defaults.Redis,
		"signal":        defaults.Signal,
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
// It uses reflection to automatically traverse the default configuration
// and set any missing values in the loaded configuration.
func (l *Loader) fillDefaults() error {
	defaults := DefaultConfig()
	defaultsMap := structToMap(defaults, "")

	for key, value := range defaultsMap {
		if l.k.Get(key) == nil {
			if err := l.k.Set(key, value); err != nil {
				return fmt.Errorf("failed to set default for %s: %w", key, err)
			}
		}
	}

	return nil
}

// structToMap recursively converts a struct to a flat map with dot-separated keys.
// This enables automatic default value extraction without manual field listing.
func structToMap(v interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	val := reflect.ValueOf(v)

	// Dereference pointer if needed
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Only process structs
	if val.Kind() != reflect.Struct {
		return result
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get the mapstructure tag or use the field name
		key := field.Tag.Get("mapstructure")
		if key == "" || key == "-" {
			continue
		}

		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		// Handle different field types
		switch fieldVal.Kind() {
		case reflect.Ptr:
			if !fieldVal.IsNil() {
				// Dereference and process
				nested := structToMap(fieldVal.Elem().Interface(), fullKey)
				for k, v := range nested {
					result[k] = v
				}
			}
		case reflect.Struct:
			// Check if it's a time.Duration (has Duration method)
			if _, ok := fieldVal.Interface().(interface{ Duration() }); ok {
				result[fullKey] = fieldVal.Interface()
			} else {
				// Recursively process nested struct
				nested := structToMap(fieldVal.Interface(), fullKey)
				for k, v := range nested {
					result[k] = v
				}
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			result[fullKey] = fieldVal.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			result[fullKey] = fieldVal.Uint()
		case reflect.Float32, reflect.Float64:
			result[fullKey] = fieldVal.Float()
		case reflect.Bool:
			result[fullKey] = fieldVal.Bool()
		case reflect.String:
			result[fullKey] = fieldVal.String()
		case reflect.Slice:
			// Handle slices (like []string)
			if fieldVal.Kind() == reflect.Slice {
				sliceLen := fieldVal.Len()
				slice := make([]interface{}, sliceLen)
				for j := 0; j < sliceLen; j++ {
					slice[j] = fieldVal.Index(j).Interface()
				}
				result[fullKey] = slice
			}
		default:
			// For other types, just use the interface value
			result[fullKey] = fieldVal.Interface()
		}
	}

	return result
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
