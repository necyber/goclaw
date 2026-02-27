package eventbus

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestIntegration_PublishConsumeOrderingAndDedup(t *testing.T) {
	bus := NewMemoryBus()
	sub, err := bus.Subscribe(SubjectPrefix+".>", 16)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	defer sub.Close()

	publisher, err := NewPublisher("node-1", bus, DefaultRetryConfig(), nil)
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := publisher.PublishLifecycleEvent(ctx, LifecycleEvent{
			Domain:     DomainWorkflow,
			EventType:  "state_changed",
			ShardKey:   "shard-a",
			WorkflowID: "wf-1",
			Payload: map[string]any{
				"step": i + 1,
			},
		})
		if err != nil {
			t.Fatalf("PublishLifecycleEvent() error = %v", err)
		}
	}

	sequences := make([]int64, 0, 3)
	var firstRaw []byte
	for len(sequences) < 3 {
		select {
		case msg := <-sub.C():
			if firstRaw == nil {
				firstRaw = append([]byte(nil), msg.Payload...)
			}
			var env Envelope
			if err := json.Unmarshal(msg.Payload, &env); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			sequences = append(sequences, env.Sequence)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for messages, got=%d", len(sequences))
		}
	}
	if sequences[0] != 1 || sequences[1] != 2 || sequences[2] != 3 {
		t.Fatalf("expected sequence [1 2 3], got %v", sequences)
	}

	consumer := NewEnvelopeConsumer(nil)
	_, _, duplicate, err := consumer.DecodeAndValidate(firstRaw)
	if err != nil {
		t.Fatalf("DecodeAndValidate() error = %v", err)
	}
	if duplicate {
		t.Fatal("expected first decode not duplicate")
	}

	_, _, duplicate, err = consumer.DecodeAndValidate(firstRaw)
	if err != nil {
		t.Fatalf("DecodeAndValidate() error = %v", err)
	}
	if !duplicate {
		t.Fatal("expected second decode duplicate=true")
	}
}
