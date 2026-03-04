package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/api/events"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/eventbus"
	grpcstreaming "github.com/goclaw/goclaw/pkg/grpc/streaming"
)

type alwaysFailTransport struct{}

func (alwaysFailTransport) Publish(context.Context, string, []byte) error {
	return errors.New("simulated transport failure")
}

func TestRuntimeEventBroadcaster_BroadcastsLocalAndCanonicalEvents(t *testing.T) {
	web := events.NewBroadcaster()
	webSub := web.Subscribe(8)
	defer web.Unsubscribe(webSub)

	registry := grpcstreaming.NewSubscriberRegistry()
	streamSub := registry.Subscribe("wf-1", 8)
	defer registry.Unsubscribe(streamSub.ID)
	observer := grpcstreaming.NewWorkflowStreamObserver(registry)

	bus := eventbus.NewMemoryBus()
	busSub, err := bus.Subscribe(eventbus.SubjectPrefix+".>", 16)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	defer busSub.Close()

	publisher, err := eventbus.NewPublisher("node-a", bus, eventbus.DefaultRetryConfig(), nil)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	asyncPublisher := newAsyncLifecyclePublisher(publisher, nil, 8)
	defer asyncPublisher.Close()

	broadcaster := newRuntimeEventBroadcaster(web, observer, asyncPublisher)
	now := time.Now().UTC()
	broadcaster.BroadcastWorkflowStateChanged("wf-1", "demo", "pending", "running", now)
	broadcaster.BroadcastTaskStateChanged("wf-1", "task-1", "task", "scheduled", "running", "", nil, now)

	// Local web event path must still receive events.
	waitWebEvent(t, webSub, "workflow.state_changed")
	waitWebEvent(t, webSub, "task.state_changed")

	// Local streaming observer path must still receive typed engine events.
	seq1 := waitStreamEvent(t, streamSub.EventChan)
	if _, ok := seq1.Event.(engine.WorkflowEvent); !ok {
		t.Fatalf("first stream event type = %T, want engine.WorkflowEvent", seq1.Event)
	}
	seq2 := waitStreamEvent(t, streamSub.EventChan)
	if _, ok := seq2.Event.(engine.TaskEvent); !ok {
		t.Fatalf("second stream event type = %T, want engine.TaskEvent", seq2.Event)
	}

	// Canonical event-bus publish path must emit envelope events.
	env1 := waitEnvelope(t, busSub.C())
	env2 := waitEnvelope(t, busSub.C())
	if env1.WorkflowID != "wf-1" || env2.WorkflowID != "wf-1" {
		t.Fatalf("unexpected workflow ids from canonical publish: %s, %s", env1.WorkflowID, env2.WorkflowID)
	}
}

func TestRuntimeEventBroadcaster_CanonicalPublishIsNonBlocking(t *testing.T) {
	publisher, err := eventbus.NewPublisher("node-a", alwaysFailTransport{}, eventbus.RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 250 * time.Millisecond,
		MaxBackoff:     250 * time.Millisecond,
		BackoffFactor:  1,
	}, nil)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	asyncPublisher := newAsyncLifecyclePublisher(publisher, nil, 4)
	defer asyncPublisher.Close()

	broadcaster := newRuntimeEventBroadcaster(nil, nil, asyncPublisher)
	start := time.Now()
	broadcaster.BroadcastWorkflowStateChanged("wf-nonblock", "demo", "pending", "running", time.Now().UTC())
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("BroadcastWorkflowStateChanged() blocked for %v", elapsed)
	}
}

func waitWebEvent(t *testing.T, ch <-chan events.Event, expectedType string) {
	t.Helper()
	select {
	case event := <-ch:
		if event.Type != expectedType {
			t.Fatalf("event type = %s, want %s", event.Type, expectedType)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for web event %s", expectedType)
	}
}

func waitStreamEvent(t *testing.T, ch <-chan interface{}) *grpcstreaming.SequencedEvent {
	t.Helper()
	select {
	case event := <-ch:
		seqEvent, ok := event.(*grpcstreaming.SequencedEvent)
		if !ok {
			t.Fatalf("stream event type = %T, want *SequencedEvent", event)
		}
		return seqEvent
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stream event")
	}
	return nil
}

func waitEnvelope(t *testing.T, ch <-chan eventbus.Message) eventbus.Envelope {
	t.Helper()
	select {
	case msg := <-ch:
		var envelope eventbus.Envelope
		if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		return envelope
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for canonical envelope event")
	}
	return eventbus.Envelope{}
}
