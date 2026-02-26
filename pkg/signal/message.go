// Package signal provides the Signal Bus for inter-task communication.
//
// Signal Bus supports three message patterns:
//   - Steer: runtime parameter modification
//   - Interrupt: graceful/forced task cancellation
//   - Collect: fan-in aggregation of task outputs
package signal

import (
	"context"
	"encoding/json"
	"time"
)

// SignalType defines the type of signal.
type SignalType string

const (
	// SignalSteer is a runtime parameter modification signal.
	SignalSteer SignalType = "steer"
	// SignalInterrupt is a task cancellation signal.
	SignalInterrupt SignalType = "interrupt"
	// SignalCollect is a result collection signal.
	SignalCollect SignalType = "collect"
)

// Signal represents a message sent through the Signal Bus.
type Signal struct {
	// Type is the signal type.
	Type SignalType `json:"type"`

	// TaskID is the target task identifier.
	TaskID string `json:"task_id"`

	// Payload is the signal-specific data.
	Payload json.RawMessage `json:"payload"`

	// SentAt is the timestamp when the signal was sent.
	SentAt time.Time `json:"sent_at"`
}

// SteerPayload is the payload for a Steer signal.
type SteerPayload struct {
	// Parameters is a map of parameter names to new values.
	Parameters map[string]interface{} `json:"parameters"`
}

// InterruptPayload is the payload for an Interrupt signal.
type InterruptPayload struct {
	// Graceful indicates whether to allow cleanup time.
	Graceful bool `json:"graceful"`

	// Reason is the reason for the interruption.
	Reason string `json:"reason"`

	// Timeout is the cleanup timeout for graceful interrupts.
	Timeout time.Duration `json:"timeout"`
}

// CollectPayload is the payload for a Collect signal.
type CollectPayload struct {
	// RequestID identifies the collection request.
	RequestID string `json:"request_id"`

	// Result is the task output data.
	Result json.RawMessage `json:"result,omitempty"`

	// Error is the error message if the task failed.
	Error string `json:"error,omitempty"`
}

// signalChKey is the context key for the signal channel.
type signalChKey struct{}

// WithSignalChannel returns a new context with the signal channel attached.
func WithSignalChannel(ctx context.Context, ch <-chan *Signal) context.Context {
	return context.WithValue(ctx, signalChKey{}, ch)
}

// FromContext returns the signal channel from the context, or nil if not present.
func FromContext(ctx context.Context) <-chan *Signal {
	ch, _ := ctx.Value(signalChKey{}).(<-chan *Signal)
	return ch
}
