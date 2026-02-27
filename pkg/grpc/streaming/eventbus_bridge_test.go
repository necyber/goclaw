package streaming

import (
	"context"
	"testing"
	"time"

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
		envelope, ok := seqEvent.Event.(eventbus.Envelope)
		if !ok {
			t.Fatalf("expected eventbus.Envelope, got %T", seqEvent.Event)
		}
		if envelope.WorkflowID != "wf-bridge" {
			t.Fatalf("expected workflow id wf-bridge, got %s", envelope.WorkflowID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bridged event")
	}
}
