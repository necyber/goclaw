package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRedisBus_PublishSubscribeAcrossBuses(t *testing.T) {
	client := requireRedisBusClient(t)
	prefix := fmt.Sprintf("goclaw:test:signal:%d:", time.Now().UnixNano())

	pubBus := NewRedisBus(client, prefix, 16)
	defer pubBus.Close()
	subBus := NewRedisBus(client, prefix, 16)
	defer subBus.Close()

	taskID := "cross-node-task"
	ch, err := subBus.Subscribe(context.Background(), taskID)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer subBus.Unsubscribe(taskID)

	// Give Redis subscription loop a moment to attach before publishing.
	time.Sleep(50 * time.Millisecond)

	payload, _ := json.Marshal(map[string]string{"mode": "fast"})
	if err := pubBus.Publish(context.Background(), &Signal{
		Type:    SignalSteer,
		TaskID:  taskID,
		Payload: payload,
		SentAt:  time.Now(),
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case got := <-ch:
		if got == nil {
			t.Fatal("expected signal, got nil")
		}
		if got.Type != SignalSteer {
			t.Fatalf("expected steer signal, got %s", got.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for redis signal")
	}
}

func TestRedisBus_PublishAfterCloseReturnsError(t *testing.T) {
	client := requireRedisBusClient(t)
	bus := NewRedisBus(client, fmt.Sprintf("goclaw:test:signal:closed:%d:", time.Now().UnixNano()), 16)

	if err := bus.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	err := bus.Publish(context.Background(), &Signal{
		Type:    SignalSteer,
		TaskID:  "task-1",
		Payload: json.RawMessage(`{"v":1}`),
		SentAt:  time.Now(),
	})
	if err == nil {
		t.Fatal("expected publish to fail after close")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Fatalf("expected closed error, got: %v", err)
	}
}

func TestRedisBus_Healthy(t *testing.T) {
	client := requireRedisBusClient(t)
	bus := NewRedisBus(client, fmt.Sprintf("goclaw:test:signal:health:%d:", time.Now().UnixNano()), 16)

	if !bus.Healthy() {
		t.Fatal("expected redis bus to be healthy")
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if bus.Healthy() {
		t.Fatal("expected closed redis bus to be unhealthy")
	}
}

func TestRedisBus_SubscribeValidationAndDuplicate(t *testing.T) {
	client := requireRedisBusClient(t)
	bus := NewRedisBus(client, fmt.Sprintf("goclaw:test:signal:sub:%d:", time.Now().UnixNano()), 8)
	defer bus.Close()

	if _, err := bus.Subscribe(context.Background(), ""); err == nil {
		t.Fatal("expected subscribe with empty task ID to fail")
	}

	ch, err := bus.Subscribe(context.Background(), "task-dup")
	if err != nil {
		t.Fatalf("first subscribe failed: %v", err)
	}
	_ = ch
	if _, err := bus.Subscribe(context.Background(), "task-dup"); err == nil {
		t.Fatal("expected duplicate subscribe to fail")
	}

	if err := bus.Unsubscribe("task-not-exists"); err != nil {
		t.Fatalf("unsubscribe non-existent task should be nil, got: %v", err)
	}

	if err := bus.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if _, err := bus.Subscribe(context.Background(), "task-after-close"); err == nil {
		t.Fatal("expected subscribe on closed bus to fail")
	}
}

func TestRedisBus_PublishValidation(t *testing.T) {
	client := requireRedisBusClient(t)
	bus := NewRedisBus(client, fmt.Sprintf("goclaw:test:signal:pub:%d:", time.Now().UnixNano()), 8)
	defer bus.Close()

	if err := bus.Publish(context.Background(), nil); err == nil {
		t.Fatal("expected nil publish to fail")
	}
	if err := bus.Publish(context.Background(), &Signal{
		Type:   SignalSteer,
		TaskID: "",
		SentAt: time.Now(),
	}); err == nil {
		t.Fatal("expected empty task_id publish to fail")
	}
}
