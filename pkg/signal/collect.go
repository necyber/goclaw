package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Collector aggregates results from multiple tasks.
type Collector struct {
	bus       Bus
	taskIDs   []string
	results   map[string]*CollectPayload
	mu        sync.Mutex
	timeout   time.Duration
	requestID string
}

// NewCollector creates a new result collector for the given tasks.
func NewCollector(bus Bus, taskIDs []string, timeout time.Duration) *Collector {
	return &Collector{
		bus:       bus,
		taskIDs:   taskIDs,
		results:   make(map[string]*CollectPayload, len(taskIDs)),
		timeout:   timeout,
		requestID: fmt.Sprintf("collect-%d", time.Now().UnixNano()),
	}
}

// Collect waits for results from all tasks or until timeout.
// Returns partial results if timeout is reached.
func (c *Collector) Collect(ctx context.Context) (map[string]*CollectPayload, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Subscribe to all tasks
	channels := make(map[string]<-chan *Signal, len(c.taskIDs))
	for _, taskID := range c.taskIDs {
		ch, err := c.bus.Subscribe(ctx, "collect:"+taskID)
		if err != nil {
			// Clean up already subscribed
			for id := range channels {
				_ = c.bus.Unsubscribe("collect:" + id)
			}
			return nil, fmt.Errorf("failed to subscribe to task %s: %w", taskID, err)
		}
		channels[taskID] = ch
	}

	defer func() {
		for _, taskID := range c.taskIDs {
			_ = c.bus.Unsubscribe("collect:" + taskID)
		}
	}()

	// Wait for results
	remaining := len(c.taskIDs)
	for remaining > 0 {
		select {
		case <-ctx.Done():
			return c.results, ctx.Err()
		default:
			for taskID, ch := range channels {
				if _, done := c.results[taskID]; done {
					continue
				}
				select {
				case sig, ok := <-ch:
					if !ok {
						continue
					}
					if sig.Type == SignalCollect {
						payload, err := ParseCollectPayload(sig)
						if err == nil {
							c.mu.Lock()
							c.results[taskID] = payload
							c.mu.Unlock()
							remaining--
						}
					}
				case <-ctx.Done():
					return c.results, ctx.Err()
				}
			}
		}
	}

	return c.results, nil
}

// StreamCollect returns a channel that emits results as they arrive.
func (c *Collector) StreamCollect(ctx context.Context) (<-chan CollectResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)

	// Subscribe to all tasks
	channels := make(map[string]<-chan *Signal, len(c.taskIDs))
	for _, taskID := range c.taskIDs {
		ch, err := c.bus.Subscribe(ctx, "collect:"+taskID)
		if err != nil {
			cancel()
			for id := range channels {
				_ = c.bus.Unsubscribe("collect:" + id)
			}
			return nil, fmt.Errorf("failed to subscribe to task %s: %w", taskID, err)
		}
		channels[taskID] = ch
	}

	out := make(chan CollectResult, len(c.taskIDs))

	go func() {
		defer cancel()
		defer close(out)
		defer func() {
			for _, taskID := range c.taskIDs {
				_ = c.bus.Unsubscribe("collect:" + taskID)
			}
		}()

		remaining := len(c.taskIDs)
		for remaining > 0 {
			for taskID, ch := range channels {
				if _, done := c.results[taskID]; done {
					continue
				}
				select {
				case sig, ok := <-ch:
					if !ok {
						continue
					}
					if sig.Type == SignalCollect {
						payload, err := ParseCollectPayload(sig)
						if err == nil {
							c.mu.Lock()
							c.results[taskID] = payload
							c.mu.Unlock()
							remaining--
							out <- CollectResult{TaskID: taskID, Payload: payload}
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

// CollectResult represents a single task's collected result.
type CollectResult struct {
	TaskID  string
	Payload *CollectPayload
}

// SendCollectResult sends a task's result back to the collector.
func SendCollectResult(ctx context.Context, bus Bus, taskID string, result json.RawMessage, taskErr string) error {
	payload, err := json.Marshal(CollectPayload{
		RequestID: "",
		Result:    result,
		Error:     taskErr,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal collect payload: %w", err)
	}

	return bus.Publish(ctx, &Signal{
		Type:    SignalCollect,
		TaskID:  "collect:" + taskID,
		Payload: payload,
		SentAt:  time.Now(),
	})
}

// ParseCollectPayload extracts the CollectPayload from a signal.
func ParseCollectPayload(sig *Signal) (*CollectPayload, error) {
	if sig.Type != SignalCollect {
		return nil, fmt.Errorf("expected collect signal, got %s", sig.Type)
	}
	var p CollectPayload
	if err := json.Unmarshal(sig.Payload, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collect payload: %w", err)
	}
	return &p, nil
}
