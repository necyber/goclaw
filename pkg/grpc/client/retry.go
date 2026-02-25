package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// withRetry executes a function with retry logic
func withRetry[T any](c *Client, ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var zero T

	if c.retryPolicy == nil || c.retryPolicy.MaxAttempts <= 1 {
		return fn(ctx)
	}

	var lastErr error
	backoff := c.retryPolicy.InitialBackoff

	for attempt := 1; attempt <= c.retryPolicy.MaxAttempts; attempt++ {
		resp, err := fn(ctx)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if !c.isRetryableError(err) {
			return zero, err
		}

		// Don't retry on last attempt
		if attempt == c.retryPolicy.MaxAttempts {
			break
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(backoff):
			// Calculate next backoff
			backoff = time.Duration(float64(backoff) * c.retryPolicy.BackoffMultiplier)
			if backoff > c.retryPolicy.MaxBackoff {
				backoff = c.retryPolicy.MaxBackoff
			}
		}
	}

	return zero, fmt.Errorf("max retry attempts reached: %w", lastErr)
}

// withRetry is a convenience wrapper for the generic withRetry function
func (c *Client) withRetry(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	return withRetry(c, ctx, fn)
}

// isRetryableError checks if an error is retryable
func (c *Client) isRetryableError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	// Check against retryable error codes
	for _, retryableCode := range c.retryPolicy.RetryableErrors {
		if strings.EqualFold(st.Code().String(), retryableCode) {
			return true
		}
	}

	return false
}

// ClientError represents a typed client error
type ClientError struct {
	Code    codes.Code
	Message string
	Details []interface{}
}

// Error implements error interface
func (e *ClientError) Error() string {
	return fmt.Sprintf("gRPC error [%s]: %s", e.Code, e.Message)
}

// IsNotFound checks if error is NotFound
func IsNotFound(err error) bool {
	return hasCode(err, codes.NotFound)
}

// IsInvalidArgument checks if error is InvalidArgument
func IsInvalidArgument(err error) bool {
	return hasCode(err, codes.InvalidArgument)
}

// IsUnavailable checks if error is Unavailable
func IsUnavailable(err error) bool {
	return hasCode(err, codes.Unavailable)
}

// IsDeadlineExceeded checks if error is DeadlineExceeded
func IsDeadlineExceeded(err error) bool {
	return hasCode(err, codes.DeadlineExceeded)
}

// IsPermissionDenied checks if error is PermissionDenied
func IsPermissionDenied(err error) bool {
	return hasCode(err, codes.PermissionDenied)
}

// IsResourceExhausted checks if error is ResourceExhausted
func IsResourceExhausted(err error) bool {
	return hasCode(err, codes.ResourceExhausted)
}

// hasCode checks if error has specific gRPC code
func hasCode(err error, code codes.Code) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	return st.Code() == code
}

// GetErrorCode extracts gRPC error code
func GetErrorCode(err error) codes.Code {
	st, ok := status.FromError(err)
	if !ok {
		return codes.Unknown
	}
	return st.Code()
}

// GetErrorMessage extracts error message
func GetErrorMessage(err error) string {
	st, ok := status.FromError(err)
	if !ok {
		return err.Error()
	}
	return st.Message()
}

// WrapError wraps an error with additional context
func WrapError(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}
