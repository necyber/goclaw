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
				Enabled:           false,
				Port:              9090,
				MaxConnections:    1000,
				MaxRecvMsgSize:    4 * 1024 * 1024, // 4MB
				MaxSendMsgSize:    4 * 1024 * 1024, // 4MB
				EnableReflection:  false,
				EnableHealthCheck: true,
				Keepalive: GRPCKeepaliveConfig{
					MaxIdleSeconds:      300,
					MaxAgeSeconds:       3600,
					MaxAgeGraceSeconds:  60,
					TimeSeconds:         60,
					TimeoutSeconds:      20,
					MinTimeSeconds:      30,
					PermitWithoutStream: false,
				},
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
				Path:              "./data/badger",
				SyncWrites:        true,
				ValueLogFileSize:  1073741824, // 1GB
				NumVersionsToKeep: 1,
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
		Memory: MemoryConfig{
			Enabled:          false,
			VectorDimension:  768,
			VectorWeight:     0.7,
			BM25Weight:       0.3,
			L1CacheSize:      1000,
			ForgetThreshold:  0.1,
			DecayInterval:    1 * time.Hour,
			DefaultStability: 24.0,
			BM25: BM25Config{
				K1: 1.5,
				B:  0.75,
			},
			HNSW: HNSWConfig{
				M:              16,
				EfConstruction: 200,
				EfSearch:       100,
			},
			StoragePath: "./data/memory",
		},
	}
}
