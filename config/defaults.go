package config

import "time"

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:        "goclaw",
			Version:     "dev",
			Environment: "development",
			Debug:       false,
		},
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			GRPC: GRPCConfig{
				Port:                 9090,
				MaxConcurrentStreams: 1000,
			},
			HTTP: HTTPConfig{
				ReadTimeout:    30 * time.Second,
				WriteTimeout:   30 * time.Second,
				IdleTimeout:    120 * time.Second,
				MaxHeaderBytes: 1 << 20, // 1MB
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Orchestration: OrchestrationConfig{
			MaxAgents: 1000,
			Queue: QueueConfig{
				Type: "memory",
				Size: 10000,
			},
			Scheduler: SchedulerConfig{
				Type:          "round_robin",
				CheckInterval: 5 * time.Second,
			},
		},
		Cluster: ClusterConfig{
			Enabled: false,
			NodeID:  "node-1",
			Discovery: DiscoveryConfig{
				Type:    "consul",
				Address: "localhost:8500",
			},
			Gossip: GossipConfig{
				BindPort:      7946,
				AdvertiseAddr: "",
			},
		},
		Storage: StorageConfig{
			Type: "memory",
			Badger: BadgerConfig{
				Path:       "./data",
				SyncWrites: false,
			},
			Redis: RedisConfig{
				Address:  "localhost:6379",
				Password: "",
				DB:       0,
			},
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
			Port:    9091,
		},
		Tracing: TracingConfig{
			Enabled:    false,
			Type:       "jaeger",
			Endpoint:   "http://localhost:14268/api/traces",
			SampleRate: 0.1,
		},
	}
}
