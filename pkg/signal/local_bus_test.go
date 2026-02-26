package signal

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestLocalBus_PublishSubscribe(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	ch, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(SteerPayload{Parameters: map[string]interface{}{"rate": 0.5}})
	err = bus.Publish(context.Background(), &Signal{
		Type:    SignalSteer,
		TaskID:  "task-1",
		Payload: payload,
		SentAt:  time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case sig := <-ch:
		if sig.Type != SignalSteer {
			t.Errorf("expected steer signal, got %s", sig.Type)
		}
		if sig.TaskID != "task-1" {
			t.Errorf("expected task-1, got %s", sig.TaskID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for signal")
	}
}

func TestLocalBus_Unsubscribe(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	_, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}

	err = bus.Unsubscribe("task-1")
	if err != nil {
		t.Fatal(err)
	}

	// Publishing to unsubscribed task should not error
	err = bus.Publish(context.Background(), &Signal{
		Type:   SignalSteer,
		TaskID: "task-1",
		SentAt: time.Now(),
	})
	if err != nil {
		t.Errorf("expected no error publishing to unsubscribed task, got %v", err)
	}
}

func TestLocalBus_DuplicateSubscribe(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	_, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = bus.Subscribe(context.Background(), "task-1")
	if err == nil {
		t.Error("expected error on duplicate subscribe")
	}
}

func TestLocalBus_Close(t *testing.T) {
	bus := NewLocalBus(16)

	ch, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}

	err = bus.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed")
	}

	// Operations on closed bus should fail
	_, err = bus.Subscribe(context.Background(), "task-2")
	if err == nil {
		t.Error("expected error subscribing to closed bus")
	}

	err = bus.Publish(context.Background(), &Signal{TaskID: "task-1", SentAt: time.Now()})
	if err == nil {
		t.Error("expected error publishing to closed bus")
	}

	if bus.Healthy() {
		t.Error("expected closed bus to be unhealthy")
	}
}

func TestLocalBus_BufferOverflow(t *testing.T) {
	bus := NewLocalBus(2) // small buffer
	defer bus.Close()

	ch, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}

	// Send 3 signals to a buffer of 2 — should drop oldest
	for i := 0; i < 3; i++ {
		payload, _ := json.Marshal(map[string]int{"seq": i})
		err := bus.Publish(context.Background(), &Signal{
			Type:    SignalSteer,
			TaskID:  "task-1",
			Payload: payload,
			SentAt:  time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Should get the latest signals (oldest dropped)
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count < 2 {
		t.Errorf("expected at least 2 signals in buffer, got %d", count)
	}
}

func TestLocalBus_NilSignal(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	err := bus.Publish(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil signal")
	}
}

func TestLocalBus_EmptyTaskID(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	err := bus.Publish(context.Background(), &Signal{TaskID: "", SentAt: time.Now()})
	if err == nil {
		t.Error("expected error for empty task_id")
	}

	_, err = bus.Subscribe(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty task_id subscribe")
	}
}

func TestLocalBus_Healthy(t *testing.T) {
	bus := NewLocalBus(16)
	if !bus.Healthy() {
		t.Error("expected new bus to be healthy")
	}
	bus.Close()
	if bus.Healthy() {
		t.Error("expected closed bus to be unhealthy")
	}
}

func TestFromContext(t *testing.T) {
	ch := make(chan *Signal, 1)
	ctx := WithSignalChannel(context.Background(), ch)

	got := FromContext(ctx)
	if got == nil {
		t.Fatal("expected signal channel from context")
	}

	// Send a signal through
	ch <- &Signal{Type: SignalSteer, TaskID: "t1"}
	sig := <-got
	if sig.TaskID != "t1" {
		t.Errorf("expected task t1, got %s", sig.TaskID)
	}
}

func TestFromContext_Missing(t *testing.T) {
	got := FromContext(context.Background())
	if got != nil {
		t.Error("expected nil from context without signal channel")
	}
}
