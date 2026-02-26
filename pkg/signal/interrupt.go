package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SendInterrupt sends an Interrupt signal to cancel a task.
func SendInterrupt(ctx context.Context, bus Bus, taskID string, graceful bool, reason string, timeout time.Duration) error {
	if taskID == "" {
		return fmt.Errorf("task_id cannot be empty")
	}

	payload, err := json.Marshal(InterruptPayload{
		Graceful: graceful,
		Reason:   reason,
		Timeout:  timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal interrupt payload: %w", err)
	}

	return bus.Publish(ctx, &Signal{
		Type:    SignalInterrupt,
		TaskID:  taskID,
		Payload: payload,
		SentAt:  time.Now(),
	})
}

// ParseInterruptPayload extracts the InterruptPayload from a signal.
func ParseInterruptPayload(sig *Signal) (*InterruptPayload, error) {
	if sig.Type != SignalInterrupt {
		return nil, fmt.Errorf("expected interrupt signal, got %s", sig.Type)
	}
	var p InterruptPayload
	if err := json.Unmarshal(sig.Payload, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal interrupt payload: %w", err)
	}
	return &p, nil
}
