package eventbus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// SchemaVersionV1 is the initial distributed event schema.
	SchemaVersionV1 = "v1"
)

// Envelope is the canonical distributed lifecycle event envelope.
type Envelope struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	Timestamp     time.Time       `json:"timestamp"`
	SchemaVersion string          `json:"schema_version"`
	NodeID        string          `json:"node_id"`
	ShardKey      string          `json:"shard_key"`
	WorkflowID    string          `json:"workflow_id,omitempty"`
	TaskID        string          `json:"task_id,omitempty"`
	OrderingKey   string          `json:"ordering_key"`
	Sequence      int64           `json:"sequence"`
	Payload       json.RawMessage `json:"payload"`
}

// BuildEnvelopeInput is used to construct a new envelope.
type BuildEnvelopeInput struct {
	EventType     string
	SchemaVersion string
	NodeID        string
	ShardKey      string
	WorkflowID    string
	TaskID        string
	OrderingKey   string
	Sequence      int64
	Payload       any
}

// BuildEnvelope creates a canonical envelope with generated event identity.
func BuildEnvelope(input BuildEnvelopeInput) (Envelope, error) {
	if input.EventType == "" {
		return Envelope{}, fmt.Errorf("eventbus: event type is required")
	}
	if input.NodeID == "" {
		return Envelope{}, fmt.Errorf("eventbus: node id is required")
	}
	if input.OrderingKey == "" {
		return Envelope{}, fmt.Errorf("eventbus: ordering key is required")
	}
	if input.Sequence <= 0 {
		return Envelope{}, fmt.Errorf("eventbus: sequence must be > 0")
	}
	if input.SchemaVersion == "" {
		input.SchemaVersion = SchemaVersionV1
	}

	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return Envelope{}, fmt.Errorf("eventbus: marshal payload: %w", err)
	}

	return Envelope{
		EventID:       uuid.NewString(),
		EventType:     input.EventType,
		Timestamp:     time.Now().UTC(),
		SchemaVersion: input.SchemaVersion,
		NodeID:        input.NodeID,
		ShardKey:      input.ShardKey,
		WorkflowID:    input.WorkflowID,
		TaskID:        input.TaskID,
		OrderingKey:   input.OrderingKey,
		Sequence:      input.Sequence,
		Payload:       payload,
	}, nil
}
