package streaming

import (
	"fmt"
	"sync"
	"time"

	"github.com/goclaw/goclaw/pkg/engine"
)

// Subscriber represents a workflow event subscriber
type Subscriber struct {
	ID            string
	WorkflowID    string
	EventChan     chan interface{}
	ErrorChan     chan error
	LastSequence  int64
	CreatedAt     time.Time
	BufferSize    int
	SlowConsumer  bool
}

// SubscriberRegistry manages streaming subscribers
type SubscriberRegistry struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscriber // subscriberID -> Subscriber
	byWorkflow  map[string][]string    // workflowID -> []subscriberID
	sequence    int64
}

// NewSubscriberRegistry creates a new subscriber registry
func NewSubscriberRegistry() *SubscriberRegistry {
	return &SubscriberRegistry{
		subscribers: make(map[string]*Subscriber),
		byWorkflow:  make(map[string][]string),
		sequence:    0,
	}
}

// Subscribe creates a new subscriber for a workflow
func (r *SubscriberRegistry) Subscribe(workflowID string, bufferSize int) *Subscriber {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub := &Subscriber{
		ID:           generateSubscriberID(),
		WorkflowID:   workflowID,
		EventChan:    make(chan interface{}, bufferSize),
		ErrorChan:    make(chan error, 1),
		LastSequence: r.sequence,
		CreatedAt:    time.Now(),
		BufferSize:   bufferSize,
		SlowConsumer: false,
	}

	r.subscribers[sub.ID] = sub
	r.byWorkflow[workflowID] = append(r.byWorkflow[workflowID], sub.ID)

	return sub
}

// Unsubscribe removes a subscriber
func (r *SubscriberRegistry) Unsubscribe(subscriberID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, exists := r.subscribers[subscriberID]
	if !exists {
		return
	}

	// Remove from workflow index
	workflowSubs := r.byWorkflow[sub.WorkflowID]
	for i, id := range workflowSubs {
		if id == subscriberID {
			r.byWorkflow[sub.WorkflowID] = append(workflowSubs[:i], workflowSubs[i+1:]...)
			break
		}
	}

	// Clean up empty workflow entries
	if len(r.byWorkflow[sub.WorkflowID]) == 0 {
		delete(r.byWorkflow, sub.WorkflowID)
	}

	// Close channels and remove subscriber
	close(sub.EventChan)
	close(sub.ErrorChan)
	delete(r.subscribers, subscriberID)
}

// GetSubscriber retrieves a subscriber by ID
func (r *SubscriberRegistry) GetSubscriber(subscriberID string) (*Subscriber, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sub, exists := r.subscribers[subscriberID]
	return sub, exists
}

// GetWorkflowSubscribers returns all subscribers for a workflow
func (r *SubscriberRegistry) GetWorkflowSubscribers(workflowID string) []*Subscriber {
	r.mu.RLock()
	defer r.mu.RUnlock()

	subIDs := r.byWorkflow[workflowID]
	subs := make([]*Subscriber, 0, len(subIDs))

	for _, id := range subIDs {
		if sub, exists := r.subscribers[id]; exists {
			subs = append(subs, sub)
		}
	}

	return subs
}

// Broadcast sends an event to all subscribers of a workflow
func (r *SubscriberRegistry) Broadcast(workflowID string, event interface{}) {
	r.mu.Lock()
	r.sequence++
	seq := r.sequence
	r.mu.Unlock()

	subs := r.GetWorkflowSubscribers(workflowID)

	for _, sub := range subs {
		select {
		case sub.EventChan <- &SequencedEvent{
			Sequence: seq,
			Event:    event,
		}:
			// Event sent successfully
		default:
			// Channel full - mark as slow consumer
			sub.SlowConsumer = true
		}
	}
}

// SequencedEvent wraps an event with a sequence number
type SequencedEvent struct {
	Sequence int64
	Event    interface{}
}

// WorkflowStreamObserver implements engine.WorkflowObserver for streaming
type WorkflowStreamObserver struct {
	registry *SubscriberRegistry
}

// NewWorkflowStreamObserver creates a new workflow stream observer
func NewWorkflowStreamObserver(registry *SubscriberRegistry) *WorkflowStreamObserver {
	return &WorkflowStreamObserver{
		registry: registry,
	}
}

// OnWorkflowEvent handles workflow events
func (o *WorkflowStreamObserver) OnWorkflowEvent(event engine.WorkflowEvent) {
	o.registry.Broadcast(event.WorkflowID, event)
}

// OnTaskEvent handles task events
func (o *WorkflowStreamObserver) OnTaskEvent(event engine.TaskEvent) {
	o.registry.Broadcast(event.WorkflowID, event)
}

// GetSubscriberCount returns the total number of subscribers
func (r *SubscriberRegistry) GetSubscriberCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.subscribers)
}

// CleanupStaleSubscribers removes subscribers that haven't been active
func (r *SubscriberRegistry) CleanupStaleSubscribers(maxAge time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, sub := range r.subscribers {
		if now.Sub(sub.CreatedAt) > maxAge && sub.SlowConsumer {
			// Remove stale slow consumer
			r.Unsubscribe(id)
			removed++
		}
	}

	return removed
}

var subscriberIDCounter int64

func generateSubscriberID() string {
	subscriberIDCounter++
	return fmt.Sprintf("%s-%d", time.Now().Format("20060102150405"), subscriberIDCounter)
}
