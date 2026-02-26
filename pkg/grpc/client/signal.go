package client

import (
	"context"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
)

// SignalOperations provides signal-related operations.
type SignalOperations struct {
	client *Client
}

// Signals returns signal operations.
func (c *Client) Signals() *SignalOperations {
	return &SignalOperations{client: c}
}

// SignalTask sends a signal to a task or collects results.
func (s *SignalOperations) SignalTask(ctx context.Context, req *pb.SignalTaskRequest) (*pb.SignalTaskResponse, error) {
	return withRetry(s.client, ctx, func(ctx context.Context) (*pb.SignalTaskResponse, error) {
		return s.client.signalClient.SignalTask(ctx, req)
	})
}
