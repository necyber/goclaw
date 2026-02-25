// Package config provides configuration management for Goclaw.
package config

import (
	"fmt"
	"time"
)

// Config is the global configuration for Goclaw.
type Config struct {
	// App is the application configuration.
	App AppConfig `mapstructure:"app" validate:"required"`

	// Server is the server configuration.
	Server ServerConfig `mapstructure:"server" validate:"required"`

	// Log is the logging configuration.
	Log LogConfig `mapstructure:"log" validate:"required"`

	// Orchestration is the workflow orchestration configuration.
	Orchestration OrchestrationConfig `mapstructure:"orchestration"`

	// Cluster is the distributed cluster configuration (Phase 2).
	Cluster ClusterConfig `mapstructure:"cluster"`

	// Storage is the persistence configuration.
	Storage StorageConfig `mapstructure:"storage"`

	// Metrics is the observability configuration.
	Metrics MetricsConfig `mapstructure:"metrics"`

	// Tracing is the distributed tracing configuration (Phase 3).
	Tracing TracingConfig `mapstructure:"tracing"`
}

// AppConfig holds application metadata and settings.
type AppConfig struct {
	// Name is the application name.
	Name string `mapstructure:"name" validate:"required"`

	// Version is the application version.
	Version string `mapstructure:"version"`

	// Environment is the runtime environment (development, staging, production).
	Environment string `mapstructure:"environment" validate:"oneof=development staging production"`

	// Debug enables debug mode with verbose logging.
	Debug bool `mapstructure:"debug"`
}

// ServerConfig holds the HTTP/gRPC server configuration.
type ServerConfig struct {
	// Host is the bind address.
	Host string `mapstructure:"host"`

	// Port is the HTTP API port.
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`

	// GRPC is the gRPC server configuration.
	GRPC GRPCConfig `mapstructure:"grpc"`

	// HTTP is the HTTP server configuration.
	HTTP HTTPConfig `mapstructure:"http"`

	// CORS is the CORS configuration.
	CORS CORSConfig `mapstructure:"cors"`
}

// GRPCConfig holds gRPC-specific settings.
type GRPCConfig struct {
	// Enabled enables the gRPC server.
	Enabled bool `mapstructure:"enabled"`

	// Port is the gRPC server port.
	Port int `mapstructure:"port" validate:"min=1,max=65535"`

	// MaxConnections is the maximum number of concurrent connections.
	MaxConnections int `mapstructure:"max_connections" validate:"min=0"`

	// MaxRecvMsgSize is the maximum message size the server can receive (bytes).
	MaxRecvMsgSize int `mapstructure:"max_recv_msg_size" validate:"min=0"`

	// MaxSendMsgSize is the maximum message size the server can send (bytes).
	MaxSendMsgSize int `mapstructure:"max_send_msg_size" validate:"min=0"`

	// EnableReflection enables gRPC server reflection for debugging.
	EnableReflection bool `mapstructure:"enable_reflection"`

	// EnableHealthCheck enables gRPC health check service.
	EnableHealthCheck bool `mapstructure:"enable_health_check"`

	// TLS is the TLS/mTLS configuration.
	TLS GRPCTLSConfig `mapstructure:"tls"`

	// Keepalive is the keepalive configuration.
	Keepalive GRPCKeepaliveConfig `mapstructure:"keepalive"`
}

// GRPCTLSConfig holds gRPC TLS/mTLS settings.
type GRPCTLSConfig struct {
	// Enabled indicates whether TLS is enabled.
	Enabled bool `mapstructure:"enabled"`

	// CertFile is the path to the server certificate file.
	CertFile string `mapstructure:"cert_file"`

	// KeyFile is the path to the server private key file.
	KeyFile string `mapstructure:"key_file"`

	// CAFile is the path to the CA certificate file for mTLS.
	CAFile string `mapstructure:"ca_file"`

	// ClientAuth indicates whether to require client certificates (mTLS).
	ClientAuth bool `mapstructure:"client_auth"`
}

// GRPCKeepaliveConfig holds gRPC keepalive settings.
type GRPCKeepaliveConfig struct {
	// MaxIdleSeconds is the maximum idle time before closing connection.
	MaxIdleSeconds int `mapstructure:"max_idle_seconds" validate:"min=0"`

	// MaxAgeSeconds is the maximum connection age.
	MaxAgeSeconds int `mapstructure:"max_age_seconds" validate:"min=0"`

	// MaxAgeGraceSeconds is the grace period for closing connections.
	MaxAgeGraceSeconds int `mapstructure:"max_age_grace_seconds" validate:"min=0"`

	// TimeSeconds is the keepalive ping interval.
	TimeSeconds int `mapstructure:"time_seconds" validate:"min=0"`

	// TimeoutSeconds is the keepalive ping timeout.
	TimeoutSeconds int `mapstructure:"timeout_seconds" validate:"min=0"`

	// MinTimeSeconds is the minimum time between client pings.
	MinTimeSeconds int `mapstructure:"min_time_seconds" validate:"min=0"`

	// PermitWithoutStream allows pings without active streams.
	PermitWithoutStream bool `mapstructure:"permit_without_stream"`
}

// HTTPConfig holds HTTP-specific settings.
type HTTPConfig struct {
	// Enabled enables the HTTP server.
	Enabled bool `mapstructure:"enabled"`

	// ReadTimeout is the maximum duration for reading the entire request.
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes.
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// IdleTimeout is the maximum amount of time to wait for the next request.
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`

	// ShutdownTimeout is the maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`

	// MaxHeaderBytes limits the size of request headers.
	MaxHeaderBytes int `mapstructure:"max_header_bytes"`
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	// Enabled enables CORS support.
	Enabled bool `mapstructure:"enabled"`

	// AllowedOrigins is the list of allowed origins.
	AllowedOrigins []string `mapstructure:"allowed_origins"`

	// AllowedMethods is the list of allowed HTTP methods.
	AllowedMethods []string `mapstructure:"allowed_methods"`

	// AllowedHeaders is the list of allowed headers.
	AllowedHeaders []string `mapstructure:"allowed_headers"`

	// ExposedHeaders is the list of headers exposed to the client.
	ExposedHeaders []string `mapstructure:"exposed_headers"`

	// AllowCredentials indicates whether credentials are allowed.
	AllowCredentials bool `mapstructure:"allow_credentials"`

	// MaxAge is the maximum age of CORS preflight cache in seconds.
	MaxAge int `mapstructure:"max_age"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	// Level is the log level (debug, info, warn, error).
	Level string `mapstructure:"level" validate:"oneof=debug info warn error"`

	// Format is the output format (json, text).
	Format string `mapstructure:"format" validate:"oneof=json text"`

	// Output is the output destination (stdout, stderr, or file path).
	Output string `mapstructure:"output"`
}

// OrchestrationConfig holds workflow engine settings.
type OrchestrationConfig struct {
	// MaxAgents is the maximum number of concurrent agents.
	MaxAgents int `mapstructure:"max_agents" validate:"min=1"`

	// Queue is the task queue configuration.
	Queue QueueConfig `mapstructure:"queue"`

	// Scheduler is the task scheduler configuration.
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
}

// QueueConfig holds task queue settings.
type QueueConfig struct {
	// Type is the queue implementation (memory, redis).
	Type string `mapstructure:"type" validate:"oneof=memory redis"`

	// Size is the maximum queue size.
	Size int `mapstructure:"size" validate:"min=1"`
}

// SchedulerConfig holds scheduler settings.
type SchedulerConfig struct {
	// Type is the scheduling algorithm (round_robin, priority, load_balanced).
	Type string `mapstructure:"type" validate:"oneof=round_robin priority load_balanced"`

	// CheckInterval is how often to check for new tasks.
	CheckInterval time.Duration `mapstructure:"check_interval"`
}

// ClusterConfig holds distributed mode settings (Phase 2).
type ClusterConfig struct {
	// Enabled enables distributed mode.
	Enabled bool `mapstructure:"enabled"`

	// NodeID is the unique identifier for this node.
	NodeID string `mapstructure:"node_id"`

	// Discovery is the service discovery configuration.
	Discovery DiscoveryConfig `mapstructure:"discovery"`

	// Gossip is the gossip protocol configuration.
	Gossip GossipConfig `mapstructure:"gossip"`
}

// DiscoveryConfig holds service discovery settings.
type DiscoveryConfig struct {
	// Type is the discovery provider (consul, etcd, kubernetes).
	Type string `mapstructure:"type" validate:"oneof=consul etcd kubernetes"`

	// Address is the discovery service endpoint.
	Address string `mapstructure:"address"`
}

// GossipConfig holds gossip protocol settings.
type GossipConfig struct {
	// BindPort is the port to bind for gossip.
	BindPort int `mapstructure:"bind_port" validate:"min=1,max=65535"`

	// AdvertiseAddr is the address to advertise to other nodes.
	AdvertiseAddr string `mapstructure:"advertise_addr"`
}

// StorageConfig holds persistence settings.
type StorageConfig struct {
	// Type is the storage backend (memory, badger, redis).
	Type string `mapstructure:"type" validate:"oneof=memory badger redis"`

	// Badger is the BadgerDB configuration.
	Badger BadgerConfig `mapstructure:"badger"`

	// Redis is the Redis configuration.
	Redis RedisConfig `mapstructure:"redis"`
}

// BadgerConfig holds BadgerDB-specific settings.
type BadgerConfig struct {
	// Path is the database directory path.
	Path string `mapstructure:"path"`

	// SyncWrites enables synchronous writes for durability.
	SyncWrites bool `mapstructure:"sync_writes"`

	// ValueLogFileSize is the maximum size of value log files in bytes.
	ValueLogFileSize int64 `mapstructure:"value_log_file_size"`

	// NumVersionsToKeep is the number of versions to keep per key.
	NumVersionsToKeep int `mapstructure:"num_versions_to_keep"`
}

// RedisConfig holds Redis-specific settings.
type RedisConfig struct {
	// Address is the Redis server address.
	Address string `mapstructure:"address"`

	// Password is the Redis password.
	Password string `mapstructure:"password"`

	// DB is the Redis database number.
	DB int `mapstructure:"db"`
}

// MetricsConfig holds observability settings.
type MetricsConfig struct {
	// Enabled enables metrics collection.
	Enabled bool `mapstructure:"enabled"`

	// Path is the metrics endpoint path.
	Path string `mapstructure:"path"`

	// Port is the metrics server port.
	Port int `mapstructure:"port" validate:"min=1,max=65535"`
}

// TracingConfig holds distributed tracing settings (Phase 3).
type TracingConfig struct {
	// Enabled enables distributed tracing.
	Enabled bool `mapstructure:"enabled"`

	// Type is the tracing backend (jaeger, zipkin).
	Type string `mapstructure:"type" validate:"oneof=jaeger zipkin"`

	// Endpoint is the collector endpoint.
	Endpoint string `mapstructure:"endpoint"`

	// SampleRate is the fraction of traces to sample (0.0-1.0).
	SampleRate float64 `mapstructure:"sample_rate" validate:"min=0,max=1"`
}

// Validate performs validation on the configuration.
func (c *Config) Validate() error {
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	return nil
}

// String returns a string representation of the configuration (without sensitive data).
func (c *Config) String() string {
	return fmt.Sprintf("Config{App: %s, Server: :%d, Env: %s}",
		c.App.Name, c.Server.Port, c.App.Environment)
}
