package events

import (
	"testing"
	"time"
)

func TestBroadcaster_SubscribeBroadcastUnsubscribe(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Subscribe(1)

	b.Broadcast(Event{
		Type: "workflow.state_changed",
		Payload: map[string]any{
			"workflow_id": "wf-1",
		},
	})

	select {
	case event := <-ch:
		if event.Type != "workflow.state_changed" {
			t.Fatalf("type = %q, want workflow.state_changed", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast event")
	}

	b.Unsubscribe(ch)
}

func TestBroadcaster_WorkflowAndTaskHelpers(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Subscribe(2)

	b.BroadcastWorkflowStateChanged("wf-1", "demo", "pending", "running", time.Now().UTC())
	b.BroadcastTaskStateChanged("wf-1", "task-1", "Task 1", "pending", "running", "", nil, time.Now().UTC())

	var received int
	for received < 2 {
		select {
		case <-ch:
			received++
		case <-time.After(time.Second):
			t.Fatalf("expected 2 helper events, got %d", received)
		}
	}
}
