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
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	channels, cleanup, err := c.subscribeCollectChannels(ctx)
	if err != nil {
		metricsRecorder().RecordSignalPattern("collect", "failed", time.Since(start))
		return nil, err
	}
	defer cleanup()

	remaining := make(map[string]struct{}, len(c.taskIDs))
	for _, taskID := range c.taskIDs {
		if !c.hasResult(taskID) {
			remaining[taskID] = struct{}{}
		}
	}

	events := c.collectEvents(ctx, channels)
	for len(remaining) > 0 {
		select {
		case <-ctx.Done():
			status := "failed"
			if ctx.Err() == context.DeadlineExceeded {
				status = "timeout"
			}
			metricsRecorder().RecordSignalPattern("collect", status, time.Since(start))
			return c.snapshotResults(), ctx.Err()
		case event, ok := <-events:
			if !ok {
				continue
			}
			if _, pending := remaining[event.taskID]; !pending {
				continue
			}
			if event.sig == nil || event.sig.Type != SignalCollect {
				continue
			}
			payload, parseErr := ParseCollectPayload(event.sig)
			if parseErr != nil {
				continue
			}
			if c.storeResult(event.taskID, payload) {
				delete(remaining, event.taskID)
			}
		}
	}

	results := c.snapshotResults()
	allFailed := len(results) == len(c.taskIDs)
	if allFailed {
		for _, payload := range results {
			if payload == nil || payload.Error == "" {
				allFailed = false
				break
			}
		}
	}
	if allFailed && len(c.taskIDs) > 0 {
		metricsRecorder().RecordSignalPattern("collect", "failed", time.Since(start))
		return results, fmt.Errorf("all tasks failed")
	}

	metricsRecorder().RecordSignalPattern("collect", "success", time.Since(start))
	return results, nil
}

// StreamCollect returns a channel that emits results as they arrive.
func (c *Collector) StreamCollect(ctx context.Context) (<-chan CollectResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)

	channels, cleanup, err := c.subscribeCollectChannels(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	out := make(chan CollectResult, len(c.taskIDs))
	events := c.collectEvents(ctx, channels)

	go func() {
		defer cancel()
		defer close(out)
		defer cleanup()

		remaining := make(map[string]struct{}, len(c.taskIDs))
		for _, taskID := range c.taskIDs {
			if !c.hasResult(taskID) {
				remaining[taskID] = struct{}{}
			}
		}

		for len(remaining) > 0 {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					continue
				}
				if _, pending := remaining[event.taskID]; !pending {
					continue
				}
				if event.sig == nil || event.sig.Type != SignalCollect {
					continue
				}
				payload, parseErr := ParseCollectPayload(event.sig)
				if parseErr != nil {
					continue
				}
				if c.storeResult(event.taskID, payload) {
					delete(remaining, event.taskID)
					out <- CollectResult{TaskID: event.taskID, Payload: payload}
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
	start := time.Now()
	payload, err := json.Marshal(CollectPayload{
		RequestID: "",
		Result:    result,
		Error:     taskErr,
	})
	if err != nil {
		metricsRecorder().RecordSignalPattern("collect", "failed", time.Since(start))
		return fmt.Errorf("failed to marshal collect payload: %w", err)
	}

	if err := bus.Publish(ctx, &Signal{
		Type:    SignalCollect,
		TaskID:  "collect:" + taskID,
		Payload: payload,
		SentAt:  time.Now(),
	}); err != nil {
		metricsRecorder().RecordSignalPattern("collect", "failed", time.Since(start))
		return err
	}
	metricsRecorder().RecordSignalPattern("collect", "success", time.Since(start))
	return nil
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

type collectEvent struct {
	taskID string
	sig    *Signal
}

func (c *Collector) subscribeCollectChannels(ctx context.Context) (map[string]<-chan *Signal, func(), error) {
	channels := make(map[string]<-chan *Signal, len(c.taskIDs))
	for _, taskID := range c.taskIDs {
		ch, err := c.bus.Subscribe(ctx, "collect:"+taskID)
		if err != nil {
			for id := range channels {
				_ = c.bus.Unsubscribe("collect:" + id)
			}
			return nil, nil, fmt.Errorf("failed to subscribe to task %s: %w", taskID, err)
		}
		channels[taskID] = ch
	}

	cleanup := func() {
		for _, taskID := range c.taskIDs {
			_ = c.bus.Unsubscribe("collect:" + taskID)
		}
	}
	return channels, cleanup, nil
}

func (c *Collector) collectEvents(ctx context.Context, channels map[string]<-chan *Signal) <-chan collectEvent {
	out := make(chan collectEvent, len(channels))

	var wg sync.WaitGroup
	wg.Add(len(channels))
	for taskID, ch := range channels {
		taskID := taskID
		ch := ch
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case sig, ok := <-ch:
					if !ok {
						return
					}
					select {
					case out <- collectEvent{taskID: taskID, sig: sig}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func (c *Collector) storeResult(taskID string, payload *CollectPayload) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.results[taskID]; exists {
		return false
	}
	c.results[taskID] = payload
	return true
}

func (c *Collector) hasResult(taskID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.results[taskID]
	return exists
}

func (c *Collector) snapshotResults() map[string]*CollectPayload {
	c.mu.Lock()
	defer c.mu.Unlock()
	results := make(map[string]*CollectPayload, len(c.results))
	for taskID, payload := range c.results {
		results[taskID] = payload
	}
	return results
}
