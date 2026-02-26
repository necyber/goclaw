package signal

import "context"

// Bus defines the interface for signal delivery between tasks.
type Bus interface {
	// Publish sends a signal to the specified task.
	Publish(ctx context.Context, signal *Signal) error

	// Subscribe creates a channel that receives signals for the given task ID.
	Subscribe(ctx context.Context, taskID string) (<-chan *Signal, error)

	// Unsubscribe removes the subscription for the given task ID.
	Unsubscribe(taskID string) error

	// Close shuts down the signal bus and releases resources.
	Close() error

	// Healthy returns true if the signal bus is operational.
	Healthy() bool
}
