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
		UI: UIConfig{
			Enabled:                 true,
			BasePath:                "/ui",
			DevProxy:                "",
			MaxWebSocketConnections: 100,
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
			Type:       "",
			Exporter:   "otlpgrpc",
			Endpoint:   "localhost:4317",
			Headers:    map[string]string{},
			Timeout:    5 * time.Second,
			Sampler:    "parentbased_traceidratio",
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
		Redis: RedisLaneConfig{
			Enabled:      false,
			Address:      "localhost:6379",
			Password:     "",
			DB:           0,
			MaxRetries:   3,
			PoolSize:     10,
			MinIdleConns: 2,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		Signal: SignalConfig{
			Mode:          "local",
			BufferSize:    16,
			ChannelPrefix: "goclaw:signal:",
		},
		Saga: SagaConfig{
			Enabled:                    false,
			MaxConcurrent:              100,
			DefaultTimeout:             5 * time.Minute,
			DefaultStepTimeout:         30 * time.Second,
			WALSyncMode:                "sync",
			WALRetention:               7 * 24 * time.Hour,
			WALCleanupInterval:         1 * time.Hour,
			CompensationPolicy:         "auto",
			CompensationMaxRetries:     3,
			CompensationInitialBackoff: 100 * time.Millisecond,
			CompensationMaxBackoff:     5 * time.Second,
			CompensationBackoffFactor:  2.0,
		},
	}
}
