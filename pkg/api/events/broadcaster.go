package events

import (
	"sync"
	"time"
)

// Event is the canonical event payload broadcast to websocket subscribers.
type Event struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

// Broadcaster broadcasts events to in-process subscribers.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
}

// NewBroadcaster creates a broadcaster instance.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan Event]struct{}),
	}
}

// Subscribe subscribes to events with a buffered channel.
func (b *Broadcaster) Subscribe(buffer int) chan Event {
	if buffer <= 0 {
		buffer = 16
	}
	ch := make(chan Event, buffer)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscription and closes its channel.
func (b *Broadcaster) Unsubscribe(ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subscribers[ch]; !ok {
		return
	}
	delete(b.subscribers, ch)
	close(ch)
}

// Broadcast broadcasts a generic event to all subscribers.
func (b *Broadcaster) Broadcast(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	b.mu.RLock()
	subs := make([]chan Event, 0, len(b.subscribers))
	for ch := range b.subscribers {
		subs = append(subs, ch)
	}
	b.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Drop on overflow to keep broadcasters non-blocking.
		}
	}
}

// BroadcastWorkflowStateChanged emits a workflow state change event.
func (b *Broadcaster) BroadcastWorkflowStateChanged(
	workflowID, name, oldState, newState string,
	updatedAt time.Time,
) {
	b.Broadcast(Event{
		Type: "workflow.state_changed",
		Payload: map[string]any{
			"workflow_id": workflowID,
			"name":        name,
			"old_state":   oldState,
			"new_state":   newState,
			"updated_at":  updatedAt.UTC().Format(time.RFC3339Nano),
		},
	})
}

// BroadcastTaskStateChanged emits a task state change event.
func (b *Broadcaster) BroadcastTaskStateChanged(
	workflowID, taskID, taskName, oldState, newState, errorMessage string,
	result any,
	updatedAt time.Time,
) {
	payload := map[string]any{
		"workflow_id": workflowID,
		"task_id":     taskID,
		"task_name":   taskName,
		"old_state":   oldState,
		"new_state":   newState,
		"updated_at":  updatedAt.UTC().Format(time.RFC3339Nano),
	}
	if errorMessage != "" {
		payload["error"] = errorMessage
	}
	if result != nil {
		payload["result"] = result
	}

	b.Broadcast(Event{
		Type:    "task.state_changed",
		Payload: payload,
	})
}

// Close closes all subscriber channels.
func (b *Broadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscribers {
		close(ch)
		delete(b.subscribers, ch)
	}
}
