package streaming

import (
	"testing"
	"time"
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
