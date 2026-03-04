package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
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

type flakyRuntimeTransport struct {
	bus       *eventbus.MemoryBus
	failCount atomic.Int32
}

func (t *flakyRuntimeTransport) Publish(ctx context.Context, subject string, payload []byte) error {
	if t.failCount.Load() > 0 {
		t.failCount.Add(-1)
		return errors.New("simulated outage")
	}
	return t.bus.Publish(ctx, subject, payload)
}

type runtimeTelemetryProbe struct {
	retries    atomic.Int32
	outages    atomic.Int32
	recoveries atomic.Int32
}

func (p *runtimeTelemetryProbe) RecordPublish(string)        {}
func (p *runtimeTelemetryProbe) RecordRetry()                { p.retries.Add(1) }
func (p *runtimeTelemetryProbe) SetDegradedMode(active bool) { _ = active }
func (p *runtimeTelemetryProbe) RecordOutage()               { p.outages.Add(1) }
func (p *runtimeTelemetryProbe) RecordRecovery()             { p.recoveries.Add(1) }

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

func TestRuntimeEventBroadcaster_PublishFailureDegradesAndRecovers(t *testing.T) {
	bus := eventbus.NewMemoryBus()
	busSub, err := bus.Subscribe(eventbus.SubjectPrefix+".>", 16)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	defer busSub.Close()

	transport := &flakyRuntimeTransport{bus: bus}
	transport.failCount.Store(3)
	telemetry := &runtimeTelemetryProbe{}
	publisher, err := eventbus.NewPublisher("node-a", transport, eventbus.RetryConfig{
		MaxRetries:     1,
		InitialBackoff: 5 * time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
		BackoffFactor:  1,
	}, telemetry)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	web := events.NewBroadcaster()
	webSub := web.Subscribe(4)
	defer web.Unsubscribe(webSub)

	asyncPublisher := newAsyncLifecyclePublisher(publisher, nil, 8)
	defer asyncPublisher.Close()
	broadcaster := newRuntimeEventBroadcaster(web, nil, asyncPublisher)

	// Outage path: local broadcast should still be immediate while canonical publish degrades.
	start := time.Now()
	broadcaster.BroadcastWorkflowStateChanged("wf-degraded", "wf", "pending", "running", time.Now().UTC())
	waitWebEvent(t, webSub, "workflow.state_changed")
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("local broadcast blocked by canonical publish path, elapsed=%v", elapsed)
	}

	deadline := time.Now().Add(time.Second)
	for !publisher.Degraded() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !publisher.Degraded() {
		t.Fatal("expected publisher degraded mode during transport outage")
	}

	// Recovery path: next publish should clear degraded mode and reach bus.
	transport.failCount.Store(0)
	broadcaster.BroadcastWorkflowStateChanged("wf-degraded", "wf", "running", "completed", time.Now().UTC())
	waitEnvelope(t, busSub.C())

	deadline = time.Now().Add(time.Second)
	for publisher.Degraded() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if publisher.Degraded() {
		t.Fatal("expected publisher to recover from degraded mode")
	}
	if telemetry.retries.Load() == 0 || telemetry.outages.Load() == 0 || telemetry.recoveries.Load() == 0 {
		t.Fatalf(
			"expected retry/outage/recovery telemetry increments, got retries=%d outages=%d recoveries=%d",
			telemetry.retries.Load(),
			telemetry.outages.Load(),
			telemetry.recoveries.Load(),
		)
	}
}

func TestRuntimeEventBroadcaster_CrossNodePublishBridgeStreamFlow(t *testing.T) {
	canonicalBus := eventbus.NewMemoryBus()
	publisher, err := eventbus.NewPublisher("node-a", canonicalBus, eventbus.DefaultRetryConfig(), nil)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	asyncPublisher := newAsyncLifecyclePublisher(publisher, nil, 8)
	defer asyncPublisher.Close()

	remoteRegistry := grpcstreaming.NewSubscriberRegistry()
	remoteSub := remoteRegistry.Subscribe("wf-cross-node", 8)
	defer remoteRegistry.Unsubscribe(remoteSub.ID)

	bridge, err := grpcstreaming.NewEventBusBridge(remoteRegistry, eventbus.NewSchemaRouter())
	if err != nil {
		t.Fatalf("NewEventBusBridge() error = %v", err)
	}
	if err := bridge.Start(canonicalBus); err != nil {
		t.Fatalf("bridge.Start() error = %v", err)
	}
	defer bridge.Stop()

	broadcaster := newRuntimeEventBroadcaster(nil, nil, asyncPublisher)
	broadcaster.BroadcastWorkflowStateChanged(
		"wf-cross-node",
		"wf-cross-node",
		"pending",
		"running",
		time.Now().UTC(),
	)

	seqEvent := waitStreamEvent(t, remoteSub.EventChan)
	workflowEvent, ok := seqEvent.Event.(engine.WorkflowEvent)
	if !ok {
		t.Fatalf("event type = %T, want engine.WorkflowEvent", seqEvent.Event)
	}
	if workflowEvent.WorkflowID != "wf-cross-node" {
		t.Fatalf("workflow id = %s, want wf-cross-node", workflowEvent.WorkflowID)
	}
	if workflowEvent.EventType != engine.WorkflowEventStarted {
		t.Fatalf("event type = %s, want %s", workflowEvent.EventType, engine.WorkflowEventStarted)
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
