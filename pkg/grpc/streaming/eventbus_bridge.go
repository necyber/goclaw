package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/eventbus"
)

// EventBusBridge consumes canonical distributed event-bus messages and fan-outs to stream subscribers.
type EventBusBridge struct {
	registry *SubscriberRegistry
	consumer *eventbus.EnvelopeConsumer
	router   *eventbus.SchemaRouter

	mu     sync.Mutex
	sub    *eventbus.Subscription
	cancel context.CancelFunc
	wg     sync.WaitGroup

	decodeErrors atomic.Uint64
	delivered    atomic.Uint64
}

// NewEventBusBridge creates a bridge from event bus updates into streaming subscribers.
func NewEventBusBridge(registry *SubscriberRegistry, router *eventbus.SchemaRouter) (*EventBusBridge, error) {
	if registry == nil {
		return nil, fmt.Errorf("streaming: subscriber registry cannot be nil")
	}
	return &EventBusBridge{
		registry: registry,
		consumer: eventbus.NewEnvelopeConsumer(router),
		router:   router,
	}, nil
}

// Start subscribes to canonical lifecycle subjects and starts bridge loop.
func (b *EventBusBridge) Start(bus *eventbus.MemoryBus) error {
	if bus == nil {
		return fmt.Errorf("streaming: event bus cannot be nil")
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sub != nil {
		return nil
	}

	sub, err := bus.Subscribe(eventbus.SubjectPrefix+".>", 256)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.sub = sub
	b.cancel = cancel
	b.wg.Add(1)

	go b.loop(ctx, sub)
	return nil
}

func (b *EventBusBridge) loop(ctx context.Context, sub *eventbus.Subscription) {
	defer b.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sub.C():
			if !ok {
				return
			}

			envelope, decoded, duplicate, err := b.consumer.DecodeAndValidate(msg.Payload)
			if err != nil {
				b.decodeErrors.Add(1)
				continue
			}
			if duplicate {
				continue
			}
			if envelope.WorkflowID == "" {
				continue
			}

			event, ok := b.toLifecycleEvent(envelope, decoded)
			if !ok {
				b.decodeErrors.Add(1)
				continue
			}
			b.registry.Broadcast(envelope.WorkflowID, event)
			b.delivered.Add(1)
		}
	}
}

// Stop stops event-bus bridge and releases resources.
func (b *EventBusBridge) Stop() error {
	b.mu.Lock()
	sub := b.sub
	cancel := b.cancel
	b.sub = nil
	b.cancel = nil
	b.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if sub != nil {
		_ = sub.Close()
	}
	b.wg.Wait()
	return nil
}

// DecodeErrorCount returns the number of bridge decode/translation failures.
func (b *EventBusBridge) DecodeErrorCount() uint64 {
	return b.decodeErrors.Load()
}

// DeliveredCount returns the number of events delivered to the streaming registry.
func (b *EventBusBridge) DeliveredCount() uint64 {
	return b.delivered.Load()
}

func (b *EventBusBridge) toLifecycleEvent(envelope eventbus.Envelope, decoded any) (any, bool) {
	// Bridge MUST not silently accept unknown schema versions when schema routing is configured.
	if b.router != nil && envelope.SchemaVersion != "" && envelope.SchemaVersion != eventbus.SchemaVersionV1 {
		return nil, false
	}

	switch value := decoded.(type) {
	case engine.WorkflowEvent:
		return value, true
	case engine.TaskEvent:
		return value, true
	case eventbus.Envelope:
		return decodeEnvelopeToLifecycleEvent(value)
	default:
		return decodeEnvelopeToLifecycleEvent(envelope)
	}
}

func decodeEnvelopeToLifecycleEvent(envelope eventbus.Envelope) (any, bool) {
	payload := map[string]any{}
	if len(envelope.Payload) > 0 {
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return nil, false
		}
	}

	status := strings.TrimSpace(strFromPayload(payload, "status"))
	message := strings.TrimSpace(strFromPayload(payload, "message"))
	timestamp := tsFromPayload(payload, envelope.Timestamp.Unix())
	eventType := strings.TrimSpace(strings.ToLower(envelope.EventType))

	if envelope.TaskID != "" {
		taskID := envelope.TaskID
		if taskID == "" {
			taskID = strFromPayload(payload, "task_id")
		}
		if taskID == "" {
			return nil, false
		}
		if status == "" {
			status = strings.ToUpper(eventType)
		}
		if message == "" {
			message = "task state changed"
		}
		progress := numberFromPayload(payload, "progress")

		mapped, ok := mapTaskLifecycleEventType(eventType)
		if !ok {
			return nil, false
		}
		return engine.TaskEvent{
			WorkflowID: envelope.WorkflowID,
			TaskID:     taskID,
			EventType:  mapped,
			Status:     status,
			Message:    message,
			Progress:   int(progress),
			Timestamp:  timestamp,
		}, true
	}

	if status == "" {
		status = strings.ToUpper(eventType)
	}
	if message == "" {
		message = "workflow state changed"
	}
	mapped, ok := mapWorkflowLifecycleEventType(eventType)
	if !ok {
		return nil, false
	}
	return engine.WorkflowEvent{
		WorkflowID: envelope.WorkflowID,
		EventType:  mapped,
		Status:     status,
		Message:    message,
		Timestamp:  timestamp,
	}, true
}

func mapWorkflowLifecycleEventType(eventType string) (engine.WorkflowEventType, bool) {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "pending", "submitted":
		return engine.WorkflowEventSubmitted, true
	case "running", "started", "start":
		return engine.WorkflowEventStarted, true
	case "completed":
		return engine.WorkflowEventCompleted, true
	case "failed":
		return engine.WorkflowEventFailed, true
	case "cancelled":
		return engine.WorkflowEventCancelled, true
	default:
		return engine.WorkflowEventStarted, false
	}
}

func mapTaskLifecycleEventType(eventType string) (engine.TaskEventType, bool) {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "running", "started", "start":
		return engine.TaskEventStarted, true
	case "progress":
		return engine.TaskEventProgress, true
	case "completed":
		return engine.TaskEventCompleted, true
	case "failed":
		return engine.TaskEventFailed, true
	case "cancelled":
		return engine.TaskEventCancelled, true
	default:
		return engine.TaskEventProgress, false
	}
}

func strFromPayload(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func numberFromPayload(payload map[string]any, key string) float64 {
	if payload == nil {
		return 0
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		v, _ := typed.Float64()
		return v
	case string:
		v, err := strconv.ParseFloat(typed, 64)
		if err == nil {
			return v
		}
	}
	return 0
}

func tsFromPayload(payload map[string]any, fallback int64) int64 {
	if payload == nil {
		return fallback
	}
	value, ok := payload["timestamp"]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case string:
		if parsed, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return parsed
		}
		if ts, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return ts.Unix()
		}
	}
	return fallback
}
