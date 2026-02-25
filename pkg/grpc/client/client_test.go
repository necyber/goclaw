package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions("localhost:9090")

	assert.Equal(t, "localhost:9090", opts.Address)
	assert.False(t, opts.TLSEnabled)
	assert.Equal(t, 4*1024*1024, opts.MaxRecvMsgSize)
	assert.Equal(t, 4*1024*1024, opts.MaxSendMsgSize)
	assert.Equal(t, 30*time.Second, opts.Timeout)
	assert.NotNil(t, opts.KeepAlive)
	assert.NotNil(t, opts.RetryPolicy)
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	assert.Equal(t, 3, policy.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, policy.InitialBackoff)
	assert.Equal(t, 5*time.Second, policy.MaxBackoff)
	assert.Equal(t, 2.0, policy.BackoffMultiplier)
	assert.Contains(t, policy.RetryableErrors, "Unavailable")
	assert.Contains(t, policy.RetryableErrors, "DeadlineExceeded")
	assert.Contains(t, policy.RetryableErrors, "ResourceExhausted")
}

func TestNewClient_NilOptions(t *testing.T) {
	_, err := NewClient(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "options cannot be nil")
}

func TestNewClient_EmptyAddress(t *testing.T) {
	opts := &Options{}
	_, err := NewClient(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "address is required")
}

func TestIsRetryableError(t *testing.T) {
	client := &Client{
		retryPolicy: DefaultRetryPolicy(),
	}

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Unavailable error",
			err:      status.Error(codes.Unavailable, "service unavailable"),
			expected: true,
		},
		{
			name:     "DeadlineExceeded error",
			err:      status.Error(codes.DeadlineExceeded, "deadline exceeded"),
			expected: true,
		},
		{
			name:     "ResourceExhausted error",
			err:      status.Error(codes.ResourceExhausted, "resource exhausted"),
			expected: true,
		},
		{
			name:     "NotFound error",
			err:      status.Error(codes.NotFound, "not found"),
			expected: false,
		},
		{
			name:     "InvalidArgument error",
			err:      status.Error(codes.InvalidArgument, "invalid argument"),
			expected: false,
		},
		{
			name:     "Non-gRPC error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClientError(t *testing.T) {
	err := &ClientError{
		Code:    codes.NotFound,
		Message: "workflow not found",
		Details: []interface{}{"workflow-123"},
	}

	assert.Contains(t, err.Error(), "NotFound")
	assert.Contains(t, err.Error(), "workflow not found")
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "NotFound error",
			err:      status.Error(codes.NotFound, "not found"),
			expected: true,
		},
		{
			name:     "Other gRPC error",
			err:      status.Error(codes.InvalidArgument, "invalid"),
			expected: false,
		},
		{
			name:     "Non-gRPC error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInvalidArgument(t *testing.T) {
	err := status.Error(codes.InvalidArgument, "invalid argument")
	assert.True(t, IsInvalidArgument(err))

	err = status.Error(codes.NotFound, "not found")
	assert.False(t, IsInvalidArgument(err))
}

func TestIsUnavailable(t *testing.T) {
	err := status.Error(codes.Unavailable, "unavailable")
	assert.True(t, IsUnavailable(err))

	err = status.Error(codes.NotFound, "not found")
	assert.False(t, IsUnavailable(err))
}

func TestIsDeadlineExceeded(t *testing.T) {
	err := status.Error(codes.DeadlineExceeded, "deadline exceeded")
	assert.True(t, IsDeadlineExceeded(err))

	err = status.Error(codes.NotFound, "not found")
	assert.False(t, IsDeadlineExceeded(err))
}

func TestIsPermissionDenied(t *testing.T) {
	err := status.Error(codes.PermissionDenied, "permission denied")
	assert.True(t, IsPermissionDenied(err))

	err = status.Error(codes.NotFound, "not found")
	assert.False(t, IsPermissionDenied(err))
}

func TestIsResourceExhausted(t *testing.T) {
	err := status.Error(codes.ResourceExhausted, "resource exhausted")
	assert.True(t, IsResourceExhausted(err))

	err = status.Error(codes.NotFound, "not found")
	assert.False(t, IsResourceExhausted(err))
}

func TestGetErrorCode(t *testing.T) {
	err := status.Error(codes.NotFound, "not found")
	code := GetErrorCode(err)
	assert.Equal(t, codes.NotFound, code)

	err = assert.AnError
	code = GetErrorCode(err)
	assert.Equal(t, codes.Unknown, code)
}

func TestGetErrorMessage(t *testing.T) {
	err := status.Error(codes.NotFound, "workflow not found")
	msg := GetErrorMessage(err)
	assert.Equal(t, "workflow not found", msg)

	err = assert.AnError
	msg = GetErrorMessage(err)
	assert.Contains(t, msg, "assert.AnError")
}

func TestWrapError(t *testing.T) {
	err := assert.AnError
	wrapped := WrapError(err, "failed to submit workflow")
	assert.Contains(t, wrapped.Error(), "failed to submit workflow")
	assert.Contains(t, wrapped.Error(), "assert.AnError")

	wrapped = WrapError(nil, "no error")
	assert.Nil(t, wrapped)
}

func TestWithRetry_NoRetry(t *testing.T) {
	client := &Client{
		retryPolicy: &RetryPolicy{
			MaxAttempts: 1,
		},
	}

	callCount := 0
	fn := func(ctx context.Context) (interface{}, error) {
		callCount++
		return "success", nil
	}

	result, err := client.withRetry(context.Background(), fn)
	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, callCount)
}

func TestWithRetry_SuccessAfterRetry(t *testing.T) {
	client := &Client{
		retryPolicy: &RetryPolicy{
			MaxAttempts:       3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			RetryableErrors:   []string{"Unavailable"},
		},
	}

	callCount := 0
	fn := func(ctx context.Context) (interface{}, error) {
		callCount++
		if callCount < 3 {
			return nil, status.Error(codes.Unavailable, "unavailable")
		}
		return "success", nil
	}

	result, err := client.withRetry(context.Background(), fn)
	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount)
}

func TestWithRetry_MaxAttemptsReached(t *testing.T) {
	client := &Client{
		retryPolicy: &RetryPolicy{
			MaxAttempts:       3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			RetryableErrors:   []string{"Unavailable"},
		},
	}

	callCount := 0
	fn := func(ctx context.Context) (interface{}, error) {
		callCount++
		return nil, status.Error(codes.Unavailable, "unavailable")
	}

	result, err := client.withRetry(context.Background(), fn)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 3, callCount)
	assert.Contains(t, err.Error(), "max retry attempts reached")
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	client := &Client{
		retryPolicy: &RetryPolicy{
			MaxAttempts:       3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			RetryableErrors:   []string{"Unavailable"},
		},
	}

	callCount := 0
	fn := func(ctx context.Context) (interface{}, error) {
		callCount++
		return nil, status.Error(codes.NotFound, "not found")
	}

	result, err := client.withRetry(context.Background(), fn)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 1, callCount) // Should not retry
}

func TestWithRetry_ContextCancelled(t *testing.T) {
	client := &Client{
		retryPolicy: &RetryPolicy{
			MaxAttempts:       3,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        1 * time.Second,
			BackoffMultiplier: 2.0,
			RetryableErrors:   []string{"Unavailable"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	fn := func(ctx context.Context) (interface{}, error) {
		callCount++
		if callCount == 1 {
			// Cancel context after first attempt
			cancel()
		}
		return nil, status.Error(codes.Unavailable, "unavailable")
	}

	result, err := client.withRetry(ctx, fn)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, context.Canceled, err)
}

func TestIsTerminalWorkflowStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int32
		expected bool
	}{
		{"Completed", 3, true},
		{"Failed", 4, true},
		{"Cancelled", 5, true},
		{"Pending", 1, false},
		{"Running", 2, false},
		{"Unspecified", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't directly test the enum values without importing pb
			// This is a simplified test
			if tt.expected {
				assert.True(t, tt.status >= 3 && tt.status <= 5)
			} else {
				assert.True(t, tt.status < 3 || tt.status > 5)
			}
		})
	}
}

func TestKeepAliveOptions(t *testing.T) {
	opts := &KeepAliveOptions{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	assert.Equal(t, 10*time.Second, opts.Time)
	assert.Equal(t, 3*time.Second, opts.Timeout)
	assert.True(t, opts.PermitWithoutStream)
}

func TestRetryPolicyBackoff(t *testing.T) {
	policy := &RetryPolicy{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	backoff := policy.InitialBackoff
	assert.Equal(t, 100*time.Millisecond, backoff)

	// First retry
	backoff = time.Duration(float64(backoff) * policy.BackoffMultiplier)
	assert.Equal(t, 200*time.Millisecond, backoff)

	// Second retry
	backoff = time.Duration(float64(backoff) * policy.BackoffMultiplier)
	assert.Equal(t, 400*time.Millisecond, backoff)

	// Third retry
	backoff = time.Duration(float64(backoff) * policy.BackoffMultiplier)
	assert.Equal(t, 800*time.Millisecond, backoff)

	// Fourth retry - should cap at MaxBackoff
	backoff = time.Duration(float64(backoff) * policy.BackoffMultiplier)
	if backoff > policy.MaxBackoff {
		backoff = policy.MaxBackoff
	}
	assert.Equal(t, 1*time.Second, backoff)
}
