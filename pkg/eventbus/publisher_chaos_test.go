package eventbus

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type flakyTransport struct {
	bus       *MemoryBus
	failCount atomic.Int32
}

func (t *flakyTransport) Publish(ctx context.Context, subject string, payload []byte) error {
	if t.failCount.Load() > 0 {
		t.failCount.Add(-1)
		return errors.New("simulated nats outage")
	}
	return t.bus.Publish(ctx, subject, payload)
}

type telemetryProbe struct {
	outages    atomic.Int32
	recoveries atomic.Int32
	retries    atomic.Int32
}

func (p *telemetryProbe) RecordPublish(status string) {}
func (p *telemetryProbe) RecordRetry()                { p.retries.Add(1) }
func (p *telemetryProbe) SetDegradedMode(active bool) {}
func (p *telemetryProbe) RecordOutage()               { p.outages.Add(1) }
func (p *telemetryProbe) RecordRecovery()             { p.recoveries.Add(1) }

func TestChaos_PublisherDegradedModeOutageRecovery(t *testing.T) {
	transport := &flakyTransport{bus: NewMemoryBus()}
	transport.failCount.Store(4)

	telemetry := &telemetryProbe{}
	publisher, err := NewPublisher("node-1", transport, RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 5 * time.Millisecond,
		MaxBackoff:     20 * time.Millisecond,
		BackoffFactor:  2,
	}, telemetry)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	_, err = publisher.PublishLifecycleEvent(context.Background(), LifecycleEvent{
		Domain:     DomainWorkflow,
		EventType:  "failed",
		ShardKey:   "s1",
		WorkflowID: "wf-chaos",
		Payload:    map[string]any{"status": "failed"},
	})
	if err == nil {
		t.Fatal("expected publish failure during outage")
	}
	if !publisher.Degraded() {
		t.Fatal("expected publisher to enter degraded mode")
	}
	if telemetry.outages.Load() == 0 {
		t.Fatal("expected outage telemetry to increment")
	}
	if telemetry.retries.Load() == 0 {
		t.Fatal("expected retry telemetry to increment")
	}

	transport.failCount.Store(0)
	_, err = publisher.PublishLifecycleEvent(context.Background(), LifecycleEvent{
		Domain:     DomainWorkflow,
		EventType:  "recovered",
		ShardKey:   "s1",
		WorkflowID: "wf-chaos",
		Payload:    map[string]any{"status": "ok"},
	})
	if err != nil {
		t.Fatalf("expected publish success after recovery, got %v", err)
	}
	if publisher.Degraded() {
		t.Fatal("expected publisher to leave degraded mode after recovery")
	}
	if telemetry.recoveries.Load() == 0 {
		t.Fatal("expected recovery telemetry to increment")
	}
}
