package streaming

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/eventbus"
)

func TestEventBusBridge_BroadcastsWorkflowUpdates(t *testing.T) {
	registry := NewSubscriberRegistry()
	sub := registry.Subscribe("wf-bridge", 8)
	defer registry.Unsubscribe(sub.ID)

	bridge, err := NewEventBusBridge(registry, nil)
	if err != nil {
		t.Fatalf("NewEventBusBridge() error = %v", err)
	}
	bus := eventbus.NewMemoryBus()
	if err := bridge.Start(bus); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer bridge.Stop()

	publisher, err := eventbus.NewPublisher("node-a", bus, eventbus.DefaultRetryConfig(), nil)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	_, err = publisher.PublishLifecycleEvent(context.Background(), eventbus.LifecycleEvent{
		Domain:     eventbus.DomainWorkflow,
		EventType:  "started",
		ShardKey:   "shard-x",
		WorkflowID: "wf-bridge",
		Payload:    map[string]any{"status": "running"},
	})
	if err != nil {
		t.Fatalf("PublishLifecycleEvent() error = %v", err)
	}

	select {
	case event := <-sub.EventChan:
		seqEvent, ok := event.(*SequencedEvent)
		if !ok {
			t.Fatalf("expected *SequencedEvent, got %T", event)
		}
		workflowEvent, ok := seqEvent.Event.(engine.WorkflowEvent)
		if !ok {
			t.Fatalf("expected engine.WorkflowEvent, got %T", seqEvent.Event)
		}
		if workflowEvent.WorkflowID != "wf-bridge" {
			t.Fatalf("expected workflow id wf-bridge, got %s", workflowEvent.WorkflowID)
		}
		if workflowEvent.EventType != engine.WorkflowEventStarted {
			t.Fatalf("expected started event type, got %s", workflowEvent.EventType)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bridged event")
	}
}

func TestEventBusBridge_UnsupportedSchemaIncrementsDecodeErrors(t *testing.T) {
	registry := NewSubscriberRegistry()
	sub := registry.Subscribe("wf-unsupported", 8)
	defer registry.Unsubscribe(sub.ID)

	router := eventbus.NewSchemaRouter()
	bridge, err := NewEventBusBridge(registry, router)
	if err != nil {
		t.Fatalf("NewEventBusBridge() error = %v", err)
	}
	bus := eventbus.NewMemoryBus()
	if err := bridge.Start(bus); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer bridge.Stop()

	envelope, err := eventbus.BuildEnvelope(eventbus.BuildEnvelopeInput{
		EventType:     "running",
		SchemaVersion: "v99",
		NodeID:        "node-a",
		ShardKey:      "s1",
		WorkflowID:    "wf-unsupported",
		OrderingKey:   "wf-unsupported",
		Sequence:      1,
		Payload:       map[string]any{"status": "RUNNING"},
	})
	if err != nil {
		t.Fatalf("BuildEnvelope() error = %v", err)
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := bus.Publish(context.Background(), eventbus.WorkflowSubject("s1", "running"), body); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	select {
	case <-sub.EventChan:
		t.Fatal("expected unsupported schema event to be dropped")
	case <-time.After(200 * time.Millisecond):
	}

	if bridge.DecodeErrorCount() == 0 {
		t.Fatal("expected decode error count to increase for unsupported schema")
	}
}
