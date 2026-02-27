package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Transport publishes bytes to a subject.
type Transport interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}

// Telemetry records event-bus pipeline health and publish behavior.
type Telemetry interface {
	RecordPublish(status string)
	RecordRetry()
	SetDegradedMode(active bool)
	RecordOutage()
	RecordRecovery()
}

type nopTelemetry struct{}

func (nopTelemetry) RecordPublish(status string) {}
func (nopTelemetry) RecordRetry()                {}
func (nopTelemetry) SetDegradedMode(active bool) {}
func (nopTelemetry) RecordOutage()               {}
func (nopTelemetry) RecordRecovery()             {}

// RetryConfig controls retry/backoff behavior for publish attempts.
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// DefaultRetryConfig returns default retry policy.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
		BackoffFactor:  2,
	}
}

// LifecycleEvent is the publish input for workflow/task lifecycle updates.
type LifecycleEvent struct {
	Domain      Domain
	EventType   string
	ShardKey    string
	WorkflowID  string
	TaskID      string
	Schema      string
	Payload     any
	OrderingKey string
}

// Publisher publishes canonical distributed lifecycle events.
type Publisher struct {
	transport Transport
	nodeID    string
	retry     RetryConfig
	telemetry Telemetry

	mu        sync.Mutex
	sequences map[string]int64
	degraded  bool
}

// NewPublisher creates a lifecycle publisher.
func NewPublisher(nodeID string, transport Transport, retry RetryConfig, telemetry Telemetry) (*Publisher, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("eventbus: node id cannot be empty")
	}
	if transport == nil {
		return nil, fmt.Errorf("eventbus: transport cannot be nil")
	}
	if retry.MaxRetries < 0 {
		return nil, fmt.Errorf("eventbus: max retries cannot be negative")
	}
	if retry.InitialBackoff <= 0 || retry.MaxBackoff <= 0 || retry.BackoffFactor < 1 {
		return nil, fmt.Errorf("eventbus: invalid retry config")
	}
	if telemetry == nil {
		telemetry = nopTelemetry{}
	}
	return &Publisher{
		transport: transport,
		nodeID:    nodeID,
		retry:     retry,
		telemetry: telemetry,
		sequences: make(map[string]int64),
	}, nil
}

// PublishLifecycleEvent publishes a canonical lifecycle event with retry/backoff and degraded mode handling.
func (p *Publisher) PublishLifecycleEvent(ctx context.Context, event LifecycleEvent) (Envelope, error) {
	if err := ctx.Err(); err != nil {
		return Envelope{}, err
	}
	subject, orderingKey, err := buildSubjectAndOrdering(event)
	if err != nil {
		return Envelope{}, err
	}
	seq := p.nextSequence(orderingKey)

	envelope, err := BuildEnvelope(BuildEnvelopeInput{
		EventType:     event.EventType,
		SchemaVersion: event.Schema,
		NodeID:        p.nodeID,
		ShardKey:      event.ShardKey,
		WorkflowID:    event.WorkflowID,
		TaskID:        event.TaskID,
		OrderingKey:   orderingKey,
		Sequence:      seq,
		Payload:       event.Payload,
	})
	if err != nil {
		return Envelope{}, err
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return Envelope{}, fmt.Errorf("eventbus: marshal envelope: %w", err)
	}

	backoff := p.retry.InitialBackoff
	var publishErr error
	for attempt := 0; attempt <= p.retry.MaxRetries; attempt++ {
		publishErr = p.transport.Publish(ctx, subject, body)
		if publishErr == nil {
			p.telemetry.RecordPublish("success")
			p.onPublishRecovered()
			return envelope, nil
		}
		if attempt == p.retry.MaxRetries {
			break
		}
		p.telemetry.RecordRetry()
		p.onPublishOutage()

		select {
		case <-ctx.Done():
			return Envelope{}, ctx.Err()
		case <-time.After(backoff):
		}
		backoff = nextBackoff(backoff, p.retry.MaxBackoff, p.retry.BackoffFactor)
	}

	p.telemetry.RecordPublish("failed")
	p.onPublishOutage()
	return Envelope{}, fmt.Errorf("eventbus: publish failed: %w", publishErr)
}

// Degraded reports whether the publisher currently considers the bus degraded.
func (p *Publisher) Degraded() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.degraded
}

func (p *Publisher) nextSequence(orderingKey string) int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sequences[orderingKey]++
	return p.sequences[orderingKey]
}

func (p *Publisher) onPublishOutage() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.degraded {
		return
	}
	p.degraded = true
	p.telemetry.SetDegradedMode(true)
	p.telemetry.RecordOutage()
}

func (p *Publisher) onPublishRecovered() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.degraded {
		return
	}
	p.degraded = false
	p.telemetry.SetDegradedMode(false)
	p.telemetry.RecordRecovery()
}

func buildSubjectAndOrdering(event LifecycleEvent) (string, string, error) {
	if event.EventType == "" {
		return "", "", fmt.Errorf("eventbus: event type cannot be empty")
	}

	orderingKey := event.OrderingKey
	if orderingKey == "" {
		if event.WorkflowID != "" {
			orderingKey = event.WorkflowID
		} else if event.TaskID != "" {
			orderingKey = event.TaskID
		} else {
			orderingKey = event.ShardKey
		}
	}
	if orderingKey == "" {
		return "", "", fmt.Errorf("eventbus: ordering key cannot be empty")
	}

	switch event.Domain {
	case DomainWorkflow:
		return WorkflowSubject(event.ShardKey, event.EventType), orderingKey, nil
	case DomainTask:
		return TaskSubject(event.ShardKey, event.EventType), orderingKey, nil
	default:
		return "", "", fmt.Errorf("eventbus: unsupported domain %q", event.Domain)
	}
}

func nextBackoff(current, max time.Duration, factor float64) time.Duration {
	next := time.Duration(float64(current) * factor)
	if next > max {
		return max
	}
	return next
}
