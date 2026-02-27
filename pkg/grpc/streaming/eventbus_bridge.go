package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/goclaw/goclaw/pkg/eventbus"
)

// EventBusBridge consumes canonical distributed event-bus messages and fan-outs to stream subscribers.
type EventBusBridge struct {
	registry *SubscriberRegistry
	consumer *eventbus.EnvelopeConsumer

	mu     sync.Mutex
	sub    *eventbus.Subscription
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewEventBusBridge creates a bridge from event bus updates into streaming subscribers.
func NewEventBusBridge(registry *SubscriberRegistry, router *eventbus.SchemaRouter) (*EventBusBridge, error) {
	if registry == nil {
		return nil, fmt.Errorf("streaming: subscriber registry cannot be nil")
	}
	return &EventBusBridge{
		registry: registry,
		consumer: eventbus.NewEnvelopeConsumer(router),
	}, nil
}

// Start subscribes to canonical lifecycle subjects and starts bridge loop.
func (b *EventBusBridge) Start(bus *eventbus.MemoryBus) error {
	if bus == nil {
		return fmt.Errorf("streaming: event bus cannot be nil")
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sub != nil {
		return nil
	}

	sub, err := bus.Subscribe(eventbus.SubjectPrefix+".>", 256)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.sub = sub
	b.cancel = cancel
	b.wg.Add(1)

	go b.loop(ctx)
	return nil
}

func (b *EventBusBridge) loop(ctx context.Context) {
	defer b.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-b.sub.C():
			if !ok {
				return
			}

			envelope, decoded, duplicate, err := b.consumer.DecodeAndValidate(msg.Payload)
			if err != nil || duplicate {
				continue
			}
			if envelope.WorkflowID == "" {
				continue
			}

			// Prefer schema-routed decoded value, fall back to envelope map for loose consumers.
			event := decoded
			if decoded == nil {
				var raw map[string]any
				if uErr := json.Unmarshal(msg.Payload, &raw); uErr == nil {
					event = raw
				} else {
					event = envelope
				}
			}
			b.registry.Broadcast(envelope.WorkflowID, event)
		}
	}
}

// Stop stops event-bus bridge and releases resources.
func (b *EventBusBridge) Stop() error {
	b.mu.Lock()
	sub := b.sub
	cancel := b.cancel
	b.sub = nil
	b.cancel = nil
	b.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if sub != nil {
		_ = sub.Close()
	}
	b.wg.Wait()
	return nil
}
