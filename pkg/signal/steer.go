package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SendSteer sends a Steer signal to modify runtime parameters of a task.
func SendSteer(ctx context.Context, bus Bus, taskID string, params map[string]interface{}) error {
	if taskID == "" {
		return fmt.Errorf("task_id cannot be empty")
	}
	if len(params) == 0 {
		return fmt.Errorf("steer parameters cannot be empty")
	}

	payload, err := json.Marshal(SteerPayload{Parameters: params})
	if err != nil {
		return fmt.Errorf("failed to marshal steer payload: %w", err)
	}

	return bus.Publish(ctx, &Signal{
		Type:    SignalSteer,
		TaskID:  taskID,
		Payload: payload,
		SentAt:  time.Now(),
	})
}

// ParseSteerPayload extracts the SteerPayload from a signal.
func ParseSteerPayload(sig *Signal) (*SteerPayload, error) {
	if sig.Type != SignalSteer {
		return nil, fmt.Errorf("expected steer signal, got %s", sig.Type)
	}
	var p SteerPayload
	if err := json.Unmarshal(sig.Payload, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal steer payload: %w", err)
	}
	return &p, nil
}
