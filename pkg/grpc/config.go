package grpc

import (
	"fmt"
	"time"
)

// Config holds gRPC server configuration
type Config struct {
	// Address is the server listening address (e.g., ":9090")
	Address string

	// TLS configuration
	TLS *TLSConfig

	// MaxConnections is the maximum number of concurrent connections
	MaxConnections int

	// Keepalive settings
	Keepalive *KeepaliveConfig

	// MaxRecvMsgSize is the maximum message size the server can receive (bytes)
	MaxRecvMsgSize int

	// MaxSendMsgSize is the maximum message size the server can send (bytes)
	MaxSendMsgSize int

	// EnableReflection enables gRPC server reflection for debugging
	EnableReflection bool

	// EnableHealthCheck enables gRPC health check service
	EnableHealthCheck bool
}

// TLSConfig holds TLS/mTLS configuration
type TLSConfig struct {
	// Enabled indicates whether TLS is enabled
	Enabled bool

	// CertFile is the path to the server certificate file
	CertFile string

	// KeyFile is the path to the server private key file
	KeyFile string

	// CAFile is the path to the CA certificate file for mTLS
	CAFile string

	// ClientAuth indicates whether to require client certificates (mTLS)
	ClientAuth bool
}

// KeepaliveConfig holds keepalive configuration
type KeepaliveConfig struct {
	// MaxIdleSeconds is the maximum idle time before closing connection
	MaxIdleSeconds int

	// MaxAgeSeconds is the maximum connection age
	MaxAgeSeconds int

	// MaxAgeGraceSeconds is the grace period for closing connections
	MaxAgeGraceSeconds int

	// TimeSeconds is the keepalive ping interval
	TimeSeconds int

	// TimeoutSeconds is the keepalive ping timeout
	TimeoutSeconds int

	// MinTimeSeconds is the minimum time between client pings
	MinTimeSeconds int

	// PermitWithoutStream allows pings without active streams
	PermitWithoutStream bool
}

// DefaultConfig returns a default gRPC server configuration
func DefaultConfig() *Config {
	return &Config{
		Address:           ":9090",
		MaxConnections:    1000,
		MaxRecvMsgSize:    4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:    4 * 1024 * 1024, // 4MB
		EnableReflection:  false,
		EnableHealthCheck: true,
		Keepalive: &KeepaliveConfig{
			MaxIdleSeconds:      300,  // 5 minutes
			MaxAgeSeconds:       3600, // 1 hour
			MaxAgeGraceSeconds:  60,   // 1 minute
			TimeSeconds:         60,   // 1 minute
			TimeoutSeconds:      20,   // 20 seconds
			MinTimeSeconds:      30,   // 30 seconds
			PermitWithoutStream: false,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	if c.MaxConnections < 0 {
		return fmt.Errorf("max connections cannot be negative")
	}

	if c.MaxRecvMsgSize < 0 {
		return fmt.Errorf("max recv message size cannot be negative")
	}

	if c.MaxSendMsgSize < 0 {
		return fmt.Errorf("max send message size cannot be negative")
	}

	if c.TLS != nil && c.TLS.Enabled {
		if err := c.TLS.Validate(); err != nil {
			return fmt.Errorf("invalid TLS config: %w", err)
		}
	}

	if c.Keepalive != nil {
		if err := c.Keepalive.Validate(); err != nil {
			return fmt.Errorf("invalid keepalive config: %w", err)
		}
	}

	return nil
}

// Validate validates TLS configuration
func (t *TLSConfig) Validate() error {
	if !t.Enabled {
		return nil
	}

	if t.CertFile == "" {
		return fmt.Errorf("cert file is required when TLS is enabled")
	}

	if t.KeyFile == "" {
		return fmt.Errorf("key file is required when TLS is enabled")
	}

	if t.ClientAuth && t.CAFile == "" {
		return fmt.Errorf("CA file is required when client auth is enabled")
	}

	return nil
}

// Validate validates keepalive configuration
func (k *KeepaliveConfig) Validate() error {
	if k.MaxIdleSeconds < 0 {
		return fmt.Errorf("max idle seconds cannot be negative")
	}

	if k.MaxAgeSeconds < 0 {
		return fmt.Errorf("max age seconds cannot be negative")
	}

	if k.MaxAgeGraceSeconds < 0 {
		return fmt.Errorf("max age grace seconds cannot be negative")
	}

	if k.TimeSeconds < 0 {
		return fmt.Errorf("time seconds cannot be negative")
	}

	if k.TimeoutSeconds < 0 {
		return fmt.Errorf("timeout seconds cannot be negative")
	}

	if k.MinTimeSeconds < 0 {
		return fmt.Errorf("min time seconds cannot be negative")
	}

	if k.TimeoutSeconds > 0 && k.TimeSeconds > 0 {
		if time.Duration(k.TimeoutSeconds)*time.Second >= time.Duration(k.TimeSeconds)*time.Second {
			return fmt.Errorf("timeout must be less than ping interval")
		}
	}

	return nil
}
