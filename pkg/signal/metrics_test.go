package signal

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type testSignalMetrics struct {
	mu sync.Mutex

	sent     int
	received int
	failed   int
	patterns map[string]int
}

func newTestSignalMetrics() *testSignalMetrics {
	return &testSignalMetrics{patterns: make(map[string]int)}
}

func (m *testSignalMetrics) RecordSignalSent(mode string, signalType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent++
}

func (m *testSignalMetrics) RecordSignalReceived(mode string, signalType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.received++
}

func (m *testSignalMetrics) RecordSignalFailed(mode string, signalType string, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failed++
}

func (m *testSignalMetrics) RecordSignalPattern(pattern string, status string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.patterns[pattern+":"+status]++
}

func TestLocalBus_RecordsMetrics(t *testing.T) {
	rec := newTestSignalMetrics()
	SetMetricsRecorder(rec)
	t.Cleanup(func() { SetMetricsRecorder(nil) })

	bus := NewLocalBus(2)
	defer bus.Close()

	ch, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(SteerPayload{Parameters: map[string]any{"rate": 0.7}})
	if err := bus.Publish(context.Background(), &Signal{
		Type:    SignalSteer,
		TaskID:  "task-1",
		Payload: payload,
		SentAt:  time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	<-ch

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.sent == 0 {
		t.Fatal("expected sent metric to be recorded")
	}
	if rec.received == 0 {
		t.Fatal("expected received metric to be recorded")
	}
}

func TestMessagePattern_RecordsMetrics(t *testing.T) {
	rec := newTestSignalMetrics()
	SetMetricsRecorder(rec)
	t.Cleanup(func() { SetMetricsRecorder(nil) })

	bus := NewLocalBus(2)
	defer bus.Close()

	_, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}

	if err := SendSteer(context.Background(), bus, "task-1", map[string]any{"x": 1}); err != nil {
		t.Fatal(err)
	}
	if err := SendInterrupt(context.Background(), bus, "task-1", true, "test", time.Second); err != nil {
		t.Fatal(err)
	}
	_ = SendSteer(context.Background(), bus, "", map[string]any{"x": 1})

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.patterns["steer:success"] == 0 {
		t.Fatal("expected steer success metric")
	}
	if rec.patterns["interrupt:success"] == 0 {
		t.Fatal("expected interrupt success metric")
	}
	if rec.patterns["steer:failed"] == 0 {
		t.Fatal("expected steer failed metric")
	}
}

