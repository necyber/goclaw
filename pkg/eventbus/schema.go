package eventbus

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// PayloadSchema describes payload contract for an event type + schema version.
type PayloadSchema struct {
	SchemaVersion string
	EventType     string
	Required      []string
	Optional      []string
}

// EnvelopeDecoder decodes envelope into a version-specific consumer view.
type EnvelopeDecoder func(envelope Envelope) (any, error)

// SchemaRouter performs schema version routing and payload validation.
type SchemaRouter struct {
	mu sync.RWMutex

	payloadSchemas map[string]PayloadSchema // key: version:eventType
	decoders       map[string]EnvelopeDecoder
}

// NewSchemaRouter creates a schema router.
func NewSchemaRouter() *SchemaRouter {
	return &SchemaRouter{
		payloadSchemas: make(map[string]PayloadSchema),
		decoders:       make(map[string]EnvelopeDecoder),
	}
}

// RegisterPayloadSchema registers a payload schema contract.
func (r *SchemaRouter) RegisterPayloadSchema(schema PayloadSchema) error {
	if schema.SchemaVersion == "" || schema.EventType == "" {
		return fmt.Errorf("eventbus: schema version and event type are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payloadSchemas[schemaKey(schema.SchemaVersion, schema.EventType)] = schema
	return nil
}

// RegisterDecoder registers a version-specific envelope decoder.
func (r *SchemaRouter) RegisterDecoder(schemaVersion string, decoder EnvelopeDecoder) error {
	if schemaVersion == "" {
		return fmt.Errorf("eventbus: schema version is required")
	}
	if decoder == nil {
		return fmt.Errorf("eventbus: decoder cannot be nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decoders[schemaVersion] = decoder
	return nil
}

// ValidateOutgoing validates a publisher envelope against registered schema contracts.
func (r *SchemaRouter) ValidateOutgoing(envelope Envelope) error {
	return r.validateEnvelope(envelope)
}

// ValidateIncoming validates a consumer envelope against registered schema contracts.
func (r *SchemaRouter) ValidateIncoming(envelope Envelope) error {
	return r.validateEnvelope(envelope)
}

func (r *SchemaRouter) validateEnvelope(envelope Envelope) error {
	if envelope.EventID == "" || envelope.EventType == "" || envelope.SchemaVersion == "" {
		return fmt.Errorf("eventbus: missing required envelope fields")
	}
	if envelope.NodeID == "" || envelope.OrderingKey == "" || envelope.Sequence <= 0 {
		return fmt.Errorf("eventbus: missing required identity/ordering fields")
	}

	r.mu.RLock()
	schema, exists := r.payloadSchemas[schemaKey(envelope.SchemaVersion, envelope.EventType)]
	r.mu.RUnlock()
	if !exists {
		return nil
	}
	return validatePayloadAgainstSchema(envelope.Payload, schema)
}

// Decode routes envelope by schema version and decodes it for consumers.
func (r *SchemaRouter) Decode(envelope Envelope) (any, error) {
	r.mu.RLock()
	decoder := r.decoders[envelope.SchemaVersion]
	r.mu.RUnlock()
	if decoder == nil {
		return envelope, nil
	}
	return decoder(envelope)
}

func validatePayloadAgainstSchema(payload json.RawMessage, schema PayloadSchema) error {
	var payloadMap map[string]json.RawMessage
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return fmt.Errorf("eventbus: invalid payload json: %w", err)
	}
	for _, field := range schema.Required {
		if _, ok := payloadMap[field]; !ok {
			return fmt.Errorf("eventbus: required payload field %q missing", field)
		}
	}
	return nil
}

func schemaKey(version, eventType string) string {
	return version + ":" + eventType
}

// FieldSchema describes one field in a versioned payload schema.
type FieldSchema struct {
	Name     string
	Type     string
	Required bool
}

// VersionedSchema defines a full schema used by compatibility checks.
type VersionedSchema struct {
	SchemaVersion string
	Fields        []FieldSchema
}

// CompatibilityReport reports additive/breaking changes between schema versions.
type CompatibilityReport struct {
	Compatible    bool
	Additive      bool
	AddedOptional []string
	AddedRequired []string
	Removed       []string
	TypeChanged   []string
}

// CheckCompatibility compares two schemas and classifies additive vs breaking evolution.
func CheckCompatibility(previous, next VersionedSchema) CompatibilityReport {
	prevMap := make(map[string]FieldSchema, len(previous.Fields))
	nextMap := make(map[string]FieldSchema, len(next.Fields))
	for _, field := range previous.Fields {
		prevMap[field.Name] = field
	}
	for _, field := range next.Fields {
		nextMap[field.Name] = field
	}

	report := CompatibilityReport{
		Compatible: true,
		Additive:   true,
	}

	for name, prevField := range prevMap {
		nextField, ok := nextMap[name]
		if !ok {
			report.Compatible = false
			report.Additive = false
			report.Removed = append(report.Removed, name)
			continue
		}
		if prevField.Type != nextField.Type {
			report.Compatible = false
			report.Additive = false
			report.TypeChanged = append(report.TypeChanged, name)
			continue
		}
		if prevField.Required && !nextField.Required {
			report.Compatible = false
			report.Additive = false
			report.TypeChanged = append(report.TypeChanged, name+":requiredness")
		}
	}

	for name, nextField := range nextMap {
		if _, exists := prevMap[name]; exists {
			continue
		}
		if nextField.Required {
			report.Compatible = false
			report.Additive = false
			report.AddedRequired = append(report.AddedRequired, name)
			continue
		}
		report.AddedOptional = append(report.AddedOptional, name)
	}

	sort.Strings(report.AddedOptional)
	sort.Strings(report.AddedRequired)
	sort.Strings(report.Removed)
	sort.Strings(report.TypeChanged)
	return report
}
