package signal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCollectorCollect_AllFailedReturnsError(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	collector := NewCollector(bus, []string{"task-1", "task-2"}, time.Second)

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = SendCollectResult(context.Background(), bus, "task-1", nil, "failed-a")
		_ = SendCollectResult(context.Background(), bus, "task-2", nil, "failed-b")
	}()

	results, err := collector.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error when all tasks failed")
	}
	if !strings.Contains(err.Error(), "all tasks failed") {
		t.Fatalf("expected all failed error, got: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestCollectorCollect_PartialFailure(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	collector := NewCollector(bus, []string{"task-1", "task-2"}, time.Second)

	go func() {
		time.Sleep(20 * time.Millisecond)
		okResult, _ := json.Marshal(map[string]string{"status": "ok"})
		_ = SendCollectResult(context.Background(), bus, "task-1", okResult, "")
		_ = SendCollectResult(context.Background(), bus, "task-2", nil, "failed")
	}()

	results, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("partial failure should not return aggregate error, got: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results["task-1"] == nil || results["task-1"].Error != "" {
		t.Fatalf("expected task-1 success payload, got: %+v", results["task-1"])
	}
	if results["task-2"] == nil || results["task-2"].Error == "" {
		t.Fatalf("expected task-2 failure payload, got: %+v", results["task-2"])
	}
}

func TestCollectorCollect_TimeoutReturnsPartialResults(t *testing.T) {
	bus := newStubCollectBus()

	collector := NewCollector(bus, []string{"task-1", "task-2"}, 400*time.Millisecond)

	go func() {
		okResult, _ := json.Marshal(map[string]string{"status": "ok"})
		payload, _ := json.Marshal(CollectPayload{Result: okResult})
		deadline := time.Now().Add(200 * time.Millisecond)
		for time.Now().Before(deadline) {
			ch := bus.mustGet("collect:task-1")
			if ch != nil {
				ch <- &Signal{
					Type:    SignalCollect,
					TaskID:  "collect:task-1",
					Payload: payload,
					SentAt:  time.Now(),
				}
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	results, err := collector.Collect(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got: %v", err)
	}
	if len(results) < 1 {
		t.Fatalf("expected at least 1 partial result, got %d", len(results))
	}
	if _, ok := results["task-1"]; !ok {
		t.Fatalf("expected task-1 result in partial results, got: %+v", results)
	}
}

func TestCollectorStreamCollect(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	collector := NewCollector(bus, []string{"task-1", "task-2"}, time.Second)
	stream, err := collector.StreamCollect(context.Background())
	if err != nil {
		t.Fatalf("StreamCollect failed: %v", err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = SendCollectResult(context.Background(), bus, "task-1", json.RawMessage(`{"v":1}`), "")
		_ = SendCollectResult(context.Background(), bus, "task-2", json.RawMessage(`{"v":2}`), "")
	}()

	seen := map[string]bool{}
	for r := range stream {
		seen[r.TaskID] = true
	}

	if !seen["task-1"] || !seen["task-2"] {
		t.Fatalf("expected both streamed task results, got: %+v", seen)
	}
}

func TestCollectorCollect_SubscribeFailure(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	// Pre-subscribe to force duplicate subscription error in collector.
	ch, err := bus.Subscribe(context.Background(), "collect:task-1")
	if err != nil {
		t.Fatalf("pre-subscribe failed: %v", err)
	}
	defer bus.Unsubscribe("collect:task-1")
	_ = ch

	collector := NewCollector(bus, []string{"task-1"}, time.Second)
	if _, err := collector.Collect(context.Background()); err == nil {
		t.Fatal("expected collect to fail on subscribe conflict")
	}
}

type stubCollectBus struct {
	subs map[string]chan *Signal
}

func newStubCollectBus() *stubCollectBus {
	return &stubCollectBus{
		subs: make(map[string]chan *Signal),
	}
}

func (b *stubCollectBus) Publish(context.Context, *Signal) error {
	return nil
}

func (b *stubCollectBus) Subscribe(_ context.Context, taskID string) (<-chan *Signal, error) {
	ch := make(chan *Signal, 1)
	b.subs[taskID] = ch
	return ch, nil
}

func (b *stubCollectBus) Unsubscribe(taskID string) error {
	if ch, ok := b.subs[taskID]; ok {
		close(ch)
		delete(b.subs, taskID)
	}
	return nil
}

func (b *stubCollectBus) Close() error {
	for taskID := range b.subs {
		_ = b.Unsubscribe(taskID)
	}
	return nil
}

func (b *stubCollectBus) Healthy() bool {
	return true
}

func (b *stubCollectBus) mustGet(taskID string) chan *Signal {
	return b.subs[taskID]
}
