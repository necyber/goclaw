package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

// Client is the gRPC client for Goclaw
type Client struct {
	conn            *grpc.ClientConn
	workflowClient  pb.WorkflowServiceClient
	streamingClient pb.StreamingServiceClient
	batchClient     pb.BatchServiceClient
	adminClient     pb.AdminServiceClient
	signalClient    pb.SignalServiceClient
	healthClient    grpc_health_v1.HealthClient
	opts            *Options
	retryPolicy     *RetryPolicy
}

// Options contains client configuration options
type Options struct {
	// Address is the server address (host:port)
	Address string

	// TLS configuration
	TLSEnabled bool
	CertFile   string
	KeyFile    string
	CAFile     string
	ServerName string

	// Connection options
	MaxRecvMsgSize int
	MaxSendMsgSize int
	Timeout        time.Duration
	KeepAlive      *KeepAliveOptions

	// Retry policy
	RetryPolicy *RetryPolicy

	// Additional dial options
	DialOptions []grpc.DialOption
}

// KeepAliveOptions contains keepalive configuration
type KeepAliveOptions struct {
	Time                time.Duration
	Timeout             time.Duration
	PermitWithoutStream bool
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	RetryableErrors   []string
}

// DefaultOptions returns default client options
func DefaultOptions(address string) *Options {
	return &Options{
		Address:        address,
		TLSEnabled:     false,
		MaxRecvMsgSize: 4 * 1024 * 1024, // 4MB
		MaxSendMsgSize: 4 * 1024 * 1024, // 4MB
		Timeout:        30 * time.Second,
		KeepAlive: &KeepAliveOptions{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		},
		RetryPolicy: DefaultRetryPolicy(),
	}
}

// DefaultRetryPolicy returns default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableErrors: []string{
			"Unavailable",
			"DeadlineExceeded",
			"ResourceExhausted",
		},
	}
}

// NewClient creates a new Goclaw gRPC client
func NewClient(opts *Options) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	if opts.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	// Build dial options
	dialOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(opts.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(opts.MaxSendMsgSize),
		),
	}

	// Add keepalive options
	if opts.KeepAlive != nil {
		dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                opts.KeepAlive.Time,
			Timeout:             opts.KeepAlive.Timeout,
			PermitWithoutStream: opts.KeepAlive.PermitWithoutStream,
		}))
	}

	// Add TLS credentials
	if opts.TLSEnabled {
		creds, err := loadTLSCredentials(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add custom dial options
	dialOpts = append(dialOpts, opts.DialOptions...)

	// Create connection
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, opts.Address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", opts.Address, err)
	}

	client := &Client{
		conn:            conn,
		workflowClient:  pb.NewWorkflowServiceClient(conn),
		streamingClient: pb.NewStreamingServiceClient(conn),
		batchClient:     pb.NewBatchServiceClient(conn),
		adminClient:     pb.NewAdminServiceClient(conn),
		signalClient:    pb.NewSignalServiceClient(conn),
		healthClient:    grpc_health_v1.NewHealthClient(conn),
		opts:            opts,
		retryPolicy:     opts.RetryPolicy,
	}

	return client, nil
}

// loadTLSCredentials loads TLS credentials from files
func loadTLSCredentials(opts *Options) (credentials.TransportCredentials, error) {
	// Load CA certificate
	var certPool *x509.CertPool
	if opts.CAFile != "" {
		caCert, err := os.ReadFile(opts.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		certPool = x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
	}

	// Load client certificate and key for mTLS
	var certificates []tls.Certificate
	if opts.CertFile != "" && opts.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		certificates = append(certificates, cert)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: certificates,
		RootCAs:      certPool,
		ServerName:   opts.ServerName,
		MinVersion:   tls.VersionTLS12,
	}

	return credentials.NewTLS(tlsConfig), nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// HealthCheck checks if the server is healthy
func (c *Client) HealthCheck(ctx context.Context) error {
	req := &grpc_health_v1.HealthCheckRequest{
		Service: "goclaw.v1.WorkflowService",
	}

	resp, err := c.healthClient.Check(ctx, req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("service not healthy: %s", resp.Status)
	}

	return nil
}

// WaitForReady waits for the connection to be ready
func (c *Client) WaitForReady(ctx context.Context) error {
	for {
		state := c.conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if !c.conn.WaitForStateChange(ctx, state) {
			return ctx.Err()
		}
	}
}

// GetConnection returns the underlying gRPC connection
func (c *Client) GetConnection() *grpc.ClientConn {
	return c.conn
}

// WorkflowClient returns the workflow service client
func (c *Client) WorkflowClient() pb.WorkflowServiceClient {
	return c.workflowClient
}

// StreamingClient returns the streaming service client
func (c *Client) StreamingClient() pb.StreamingServiceClient {
	return c.streamingClient
}

// BatchClient returns the batch service client
func (c *Client) BatchClient() pb.BatchServiceClient {
	return c.batchClient
}

// AdminClient returns the admin service client
func (c *Client) AdminClient() pb.AdminServiceClient {
	return c.adminClient
}

// SignalClient returns the signal service client.
func (c *Client) SignalClient() pb.SignalServiceClient {
	return c.signalClient
}
