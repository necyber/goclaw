package streaming

import (
	"errors"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/engine"
)

func TestCleanupStaleSubscribers_NoDeadlock(t *testing.T) {
	registry := NewSubscriberRegistry()
	sub := registry.Subscribe("wf-1", 1)
	sub.SlowConsumer = true
	sub.CreatedAt = time.Now().Add(-2 * time.Hour)

	done := make(chan struct{})
	var removed int
	go func() {
		removed = registry.CleanupStaleSubscribers(time.Minute)
		close(done)
	}()

	select {
	case <-done:
		// pass
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup stale subscribers deadlocked")
	}

	if removed != 1 {
		t.Fatalf("expected 1 subscriber removed, got %d", removed)
	}
	if got := registry.GetSubscriberCount(); got != 0 {
		t.Fatalf("expected empty registry after cleanup, got %d", got)
	}
}

func TestBroadcast_MarksSlowConsumerWithoutPanic(t *testing.T) {
	registry := NewSubscriberRegistry()
	sub := registry.Subscribe("wf-1", 0) // unbuffered: no receiver means slow consumer path

	registry.Broadcast("wf-1", "event")

	got, ok := registry.GetSubscriber(sub.ID)
	if !ok {
		t.Fatalf("expected subscriber to exist")
	}
	if !got.SlowConsumer {
		t.Fatalf("expected subscriber to be marked slow consumer")
	}
}

func TestBroadcast_TerminalEventReplacesStaleBufferedEvent(t *testing.T) {
	registry := NewSubscriberRegistry()
	sub := registry.Subscribe("wf-1", 1)

	// Fill buffer with stale non-terminal event to simulate slow consumer.
	sub.EventChan <- &SequencedEvent{Sequence: 1, Event: engine.WorkflowEvent{
		WorkflowID: "wf-1",
		EventType:  engine.WorkflowEventStarted,
		Status:     "RUNNING",
	}}

	registry.Broadcast("wf-1", engine.WorkflowEvent{
		WorkflowID: "wf-1",
		EventType:  engine.WorkflowEventCompleted,
		Status:     "COMPLETED",
	})

	raw := <-sub.EventChan
	seq, ok := raw.(*SequencedEvent)
	if !ok {
		t.Fatalf("expected sequenced event, got %T", raw)
	}
	event, ok := seq.Event.(engine.WorkflowEvent)
	if !ok {
		t.Fatalf("expected workflow event payload, got %T", seq.Event)
	}
	if event.EventType != engine.WorkflowEventCompleted {
		t.Fatalf("expected terminal completed event, got %v", event.EventType)
	}
}

func TestBroadcast_TerminalEventSignalsBackpressureWhenUndeliverable(t *testing.T) {
	registry := NewSubscriberRegistry()
	sub := registry.Subscribe("wf-1", 0) // unbuffered + no receiver => undeliverable terminal event

	registry.Broadcast("wf-1", engine.WorkflowEvent{
		WorkflowID: "wf-1",
		EventType:  engine.WorkflowEventCancelled,
		Status:     "CANCELLED",
	})

	select {
	case err := <-sub.ErrorChan:
		if !errors.Is(err, ErrTerminalBackpressure) {
			t.Fatalf("expected ErrTerminalBackpressure, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected terminal backpressure error to be signaled")
	}
}

func TestBroadcast_PreservesSequenceOrderForDeliveredEvents(t *testing.T) {
	registry := NewSubscriberRegistry()
	sub := registry.Subscribe("wf-1", 4)

	registry.Broadcast("wf-1", engine.TaskEvent{WorkflowID: "wf-1", TaskID: "t1", EventType: engine.TaskEventStarted})
	registry.Broadcast("wf-1", engine.TaskEvent{WorkflowID: "wf-1", TaskID: "t1", EventType: engine.TaskEventProgress})
	registry.Broadcast("wf-1", engine.TaskEvent{WorkflowID: "wf-1", TaskID: "t1", EventType: engine.TaskEventCompleted})

	first := (<-sub.EventChan).(*SequencedEvent)
	second := (<-sub.EventChan).(*SequencedEvent)
	third := (<-sub.EventChan).(*SequencedEvent)

	if !(first.Sequence < second.Sequence && second.Sequence < third.Sequence) {
		t.Fatalf("expected strictly increasing sequence order, got %d %d %d", first.Sequence, second.Sequence, third.Sequence)
	}
}
