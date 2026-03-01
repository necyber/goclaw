package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test structs for validating custom validators
type FileExistsTestStruct struct {
	Path string `validate:"file_exists"`
}

type DirExistsTestStruct struct {
	Path string `validate:"dir_exists"`
}

type HostTestStruct struct {
	Host string `validate:"host"`
}

func TestValidateFileExists(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"empty path (optional)", "", true},
		{"existing file", tmpFile, true},
		{"non-existent file", "/nonexistent/file.txt", false},
		{"directory instead of file", tmpDir, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := FileExistsTestStruct{Path: tt.path}
			err := validate.Struct(s)
			if tt.expected && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("expected invalid for path %q, got valid", tt.path)
			}
		})
	}
}

func TestValidateDirExists(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"empty path (optional)", "", true},
		{"existing directory", tmpDir, true},
		{"non-existent directory", "/nonexistent/dir", false},
		{"file instead of directory", tmpFile, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := DirExistsTestStruct{Path: tt.path}
			err := validate.Struct(s)
			if tt.expected && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("expected invalid for path %q, got valid", tt.path)
			}
		})
	}
}

func TestValidateHost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{"empty host (optional)", "", true},
		{"localhost", "localhost", true},
		{"IP address", "127.0.0.1", true},
		{"IP with port", "127.0.0.1:8080", true},
		{"hostname", "example.com", true},
		{"hostname with subdomain", "api.example.com", true},
		{"hostname with multiple subdomains", "api.v1.example.com", true},
		{"IPv6 localhost", "::1", true},
		{"IPv6 address", "2001:db8::1", true},
		{"host with underscore", "my_server", true},
		{"invalid host with space", "invalid host", false},
		{"invalid host with tab", "invalid\thost", false},
		{"invalid host with newline", "invalid\nhost", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := HostTestStruct{Host: tt.host}
			err := validate.Struct(s)
			if tt.expected && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("expected invalid for host %q, got valid", tt.host)
			}
		})
	}
}

func TestIsValidHostChar(t *testing.T) {
	tests := []struct {
		char     rune
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'-', true},
		{'.', true},
		{':', true},
		{'_', true},
		{' ', false},
		{'!', false},
		{'@', false},
		{'#', false},
		{'$', false},
		{'%', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := isValidHostChar(tt.char)
			if result != tt.expected {
				t.Errorf("isValidHostChar(%q) = %v, want %v", tt.char, result, tt.expected)
			}
		})
	}
}

func TestValidateWithDetails_UIBasePath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.UI.BasePath = "invalid"

	err := ValidateWithDetails(cfg)
	if err == nil {
		t.Fatal("expected validation error for ui.base_path")
	}
}

func TestValidateWithDetails_HTTPTimeouts(t *testing.T) {
	tests := []struct {
		name  string
		apply func(cfg *Config)
	}{
		{
			name: "read timeout non-positive",
			apply: func(cfg *Config) {
				cfg.Server.HTTP.ReadTimeout = 0
			},
		},
		{
			name: "write timeout non-positive",
			apply: func(cfg *Config) {
				cfg.Server.HTTP.WriteTimeout = 0
			},
		},
		{
			name: "idle timeout non-positive",
			apply: func(cfg *Config) {
				cfg.Server.HTTP.IdleTimeout = 0
			},
		},
		{
			name: "shutdown timeout non-positive",
			apply: func(cfg *Config) {
				cfg.Server.HTTP.ShutdownTimeout = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.apply(cfg)

			err := ValidateWithDetails(cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateWithDetails_HTTPTimeouts_Valid(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.HTTP.ReadTimeout = 5 * time.Second
	cfg.Server.HTTP.WriteTimeout = 5 * time.Second
	cfg.Server.HTTP.IdleTimeout = 10 * time.Second
	cfg.Server.HTTP.ShutdownTimeout = 3 * time.Second

	if err := ValidateWithDetails(cfg); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}
