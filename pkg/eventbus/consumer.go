package eventbus

import (
	"encoding/json"
	"fmt"
	"sync"
)

// EnvelopeConsumer validates/routs envelopes and suppresses duplicate deliveries.
type EnvelopeConsumer struct {
	router *SchemaRouter

	mu         sync.Mutex
	seenEvents map[string]struct{}
}

// NewEnvelopeConsumer creates a schema-aware consumer.
func NewEnvelopeConsumer(router *SchemaRouter) *EnvelopeConsumer {
	return &EnvelopeConsumer{
		router:     router,
		seenEvents: make(map[string]struct{}),
	}
}

// DecodeAndValidate decodes raw event bytes, validates schema routing, and suppresses duplicates.
func (c *EnvelopeConsumer) DecodeAndValidate(raw []byte) (Envelope, any, bool, error) {
	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Envelope{}, nil, false, fmt.Errorf("eventbus: invalid envelope json: %w", err)
	}

	if c.router != nil {
		if err := c.router.ValidateIncoming(envelope); err != nil {
			return Envelope{}, nil, false, err
		}
	}

	c.mu.Lock()
	if _, exists := c.seenEvents[envelope.EventID]; exists {
		c.mu.Unlock()
		return envelope, nil, true, nil
	}
	c.seenEvents[envelope.EventID] = struct{}{}
	c.mu.Unlock()

	var decoded any = envelope
	var err error
	if c.router != nil {
		decoded, err = c.router.Decode(envelope)
		if err != nil {
			return Envelope{}, nil, false, err
		}
	}
	return envelope, decoded, false, nil
}
