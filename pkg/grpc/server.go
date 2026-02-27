package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/goclaw/goclaw/pkg/grpc/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Server represents a gRPC server instance
type Server struct {
	config       *Config
	grpcSrv      *grpc.Server
	listener     net.Listener
	healthServer *HealthServer
	pending      []serviceRegistration
	mu           sync.RWMutex
	running      bool
}

type serviceRegistration struct {
	desc *grpc.ServiceDesc
	impl interface{}
}

// New creates a new gRPC server with the given configuration
func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Server{
		config: cfg,
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	// Create listener
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Address, err)
	}
	s.listener = listener

	// Build server options
	opts, err := s.buildServerOptions()
	if err != nil {
		listener.Close()
		return fmt.Errorf("failed to build server options: %w", err)
	}

	// Create gRPC server
	s.grpcSrv = grpc.NewServer(opts...)

	// Register services queued before server start.
	for _, reg := range s.pending {
		s.grpcSrv.RegisterService(reg.desc, reg.impl)
	}

	// Enable reflection if configured
	if s.config.EnableReflection {
		reflection.Register(s.grpcSrv)
	}

	// Enable health check if configured
	if s.config.EnableHealthCheck {
		s.healthServer = NewHealthServer()
		grpc_health_v1.RegisterHealthServer(s.grpcSrv, s.healthServer.GetServer())
		s.healthServer.SetServingStatusAll(grpc_health_v1.HealthCheckResponse_SERVING)
	}

	s.running = true

	// Start serving in goroutine
	go func() {
		if err := s.grpcSrv.Serve(listener); err != nil {
			// Log error (will be handled by logger when integrated)
			fmt.Printf("gRPC server error: %v\n", err)
		}
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// Create a channel to signal when graceful stop completes
	stopped := make(chan struct{})

	go func() {
		s.grpcSrv.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop or context timeout
	select {
	case <-stopped:
		// Graceful stop completed
	case <-ctx.Done():
		// Context timeout, force stop
		s.grpcSrv.Stop()
		return fmt.Errorf("graceful shutdown timeout, forced stop")
	}

	s.running = false
	return nil
}

// RegisterService registers a gRPC service with the server
func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.grpcSrv != nil {
		s.grpcSrv.RegisterService(desc, impl)
		return
	}
	s.pending = append(s.pending, serviceRegistration{desc: desc, impl: impl})
}

// GetServer returns the underlying gRPC server for advanced configuration
func (s *Server) GetServer() *grpc.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.grpcSrv
}

// Address returns the server's listening address
func (s *Server) Address() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.config.Address
}

// IsRunning returns whether the server is currently running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// buildServerOptions constructs gRPC server options from config
func (s *Server) buildServerOptions() ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption

	// TLS credentials
	if s.config.TLS != nil && s.config.TLS.Enabled {
		creds, err := s.buildTLSCredentials()
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Connection limits
	if s.config.MaxConnections > 0 {
		opts = append(opts, grpc.MaxConcurrentStreams(uint32(s.config.MaxConnections)))
	}

	// Keepalive settings
	if s.config.Keepalive != nil {
		kaParams := keepalive.ServerParameters{
			MaxConnectionIdle:     time.Duration(s.config.Keepalive.MaxIdleSeconds) * time.Second,
			MaxConnectionAge:      time.Duration(s.config.Keepalive.MaxAgeSeconds) * time.Second,
			MaxConnectionAgeGrace: time.Duration(s.config.Keepalive.MaxAgeGraceSeconds) * time.Second,
			Time:                  time.Duration(s.config.Keepalive.TimeSeconds) * time.Second,
			Timeout:               time.Duration(s.config.Keepalive.TimeoutSeconds) * time.Second,
		}
		opts = append(opts, grpc.KeepaliveParams(kaParams))

		kaPolicy := keepalive.EnforcementPolicy{
			MinTime:             time.Duration(s.config.Keepalive.MinTimeSeconds) * time.Second,
			PermitWithoutStream: s.config.Keepalive.PermitWithoutStream,
		}
		opts = append(opts, grpc.KeepaliveEnforcementPolicy(kaPolicy))
	}

	// Max message sizes
	if s.config.MaxRecvMsgSize > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(s.config.MaxRecvMsgSize))
	}
	if s.config.MaxSendMsgSize > 0 {
		opts = append(opts, grpc.MaxSendMsgSize(s.config.MaxSendMsgSize))
	}
	if s.config.EnableTracing {
		opts = append(opts, interceptors.NewChainBuilder().WithTracing().Build()...)
	}

	return opts, nil
}

// buildTLSCredentials creates TLS credentials from config
func (s *Server) buildTLSCredentials() (credentials.TransportCredentials, error) {
	tlsCfg := s.config.TLS
	if tlsCfg == nil || !tlsCfg.Enabled {
		return nil, fmt.Errorf("TLS not enabled")
	}

	// Load server certificate and key
	cert, err := credentials.NewServerTLSFromFile(tlsCfg.CertFile, tlsCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// If mTLS is not required, return basic TLS
	if !tlsCfg.ClientAuth || tlsCfg.CAFile == "" {
		return cert, nil
	}

	// For mTLS, we need to load CA and configure client auth
	// This requires using tls.Config directly
	tlsConfig, err := s.buildMTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build mTLS config: %w", err)
	}

	return credentials.NewTLS(tlsConfig), nil
}

// buildMTLSConfig creates a TLS config with mutual TLS
func (s *Server) buildMTLSConfig() (*tls.Config, error) {
	tlsCfg := s.config.TLS

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Load CA certificate for client verification
	caCert, err := os.ReadFile(tlsCfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
