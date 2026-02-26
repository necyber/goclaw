package signal

import (
	"context"
	"fmt"
	"sync"
)

// LocalBus is an in-memory Signal Bus implementation using Go channels.
type LocalBus struct {
	mu          sync.RWMutex
	subscribers map[string]chan *Signal
	bufferSize  int
	closed      bool
}

// NewLocalBus creates a new in-memory Signal Bus.
func NewLocalBus(bufferSize int) *LocalBus {
	if bufferSize <= 0 {
		bufferSize = 16
	}
	return &LocalBus{
		subscribers: make(map[string]chan *Signal),
		bufferSize:  bufferSize,
	}
}

// Publish sends a signal to the target task's subscriber channel.
func (b *LocalBus) Publish(_ context.Context, sig *Signal) error {
	if sig == nil {
		metricsRecorder().RecordSignalFailed("local", "unknown", "nil_signal")
		return fmt.Errorf("signal cannot be nil")
	}
	if sig.TaskID == "" {
		metricsRecorder().RecordSignalFailed("local", string(sig.Type), "empty_task_id")
		return fmt.Errorf("signal task_id cannot be empty")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		metricsRecorder().RecordSignalFailed("local", string(sig.Type), "bus_closed")
		return fmt.Errorf("signal bus is closed")
	}

	ch, ok := b.subscribers[sig.TaskID]
	if !ok {
		metricsRecorder().RecordSignalFailed("local", string(sig.Type), "no_subscriber")
		return nil // no subscriber, silently drop
	}
	metricsRecorder().RecordSignalSent("local", string(sig.Type))

	// Non-blocking send; drop oldest if buffer full.
	select {
	case ch <- sig:
		metricsRecorder().RecordSignalReceived("local", string(sig.Type))
	default:
		metricsRecorder().RecordSignalFailed("local", string(sig.Type), "buffer_full_drop")
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- sig:
			metricsRecorder().RecordSignalReceived("local", string(sig.Type))
		default:
			metricsRecorder().RecordSignalFailed("local", string(sig.Type), "buffer_still_full")
		}
	}

	return nil
}

// Subscribe creates a buffered channel for receiving signals for the given task.
func (b *LocalBus) Subscribe(_ context.Context, taskID string) (<-chan *Signal, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id cannot be empty")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, fmt.Errorf("signal bus is closed")
	}

	if _, exists := b.subscribers[taskID]; exists {
		return nil, fmt.Errorf("task %s already subscribed", taskID)
	}

	ch := make(chan *Signal, b.bufferSize)
	b.subscribers[taskID] = ch
	return ch, nil
}

// Unsubscribe removes the subscription and closes the channel.
func (b *LocalBus) Unsubscribe(taskID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch, ok := b.subscribers[taskID]
	if !ok {
		return nil
	}

	close(ch)
	delete(b.subscribers, taskID)
	return nil
}

// Close shuts down the bus and closes all subscriber channels.
func (b *LocalBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	for taskID, ch := range b.subscribers {
		close(ch)
		delete(b.subscribers, taskID)
	}
	return nil
}

// Healthy returns true if the bus is not closed.
func (b *LocalBus) Healthy() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return !b.closed
}
