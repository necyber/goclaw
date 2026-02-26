package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	"github.com/goclaw/goclaw/pkg/signal"
)

func TestSignalServiceServer_SignalTask_Steer(t *testing.T) {
	bus := signal.NewLocalBus(16)
	defer bus.Close()

	ch, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	server := NewSignalServiceServer(bus)
	resp, err := server.SignalTask(context.Background(), &pb.SignalTaskRequest{
		Type:       pb.SignalType_SIGNAL_TYPE_STEER,
		TaskId:     "task-1",
		Parameters: map[string]string{"rate": "0.5"},
	})
	if err != nil {
		t.Fatalf("SignalTask: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %v", resp.Error)
	}

	select {
	case sig := <-ch:
		payload, err := signal.ParseSteerPayload(sig)
		if err != nil {
			t.Fatalf("ParseSteerPayload: %v", err)
		}
		if payload.Parameters["rate"] != "0.5" {
			t.Fatalf("expected rate=0.5, got %v", payload.Parameters["rate"])
		}
	case <-time.After(time.Second):
		t.Fatal("signal not received")
	}
}

func TestSignalServiceServer_SignalTask_Collect(t *testing.T) {
	bus := signal.NewLocalBus(16)
	defer bus.Close()

	server := NewSignalServiceServer(bus)
	taskIDs := []string{"task-a", "task-b"}

	go func() {
		time.Sleep(20 * time.Millisecond)
		for _, taskID := range taskIDs {
			result, _ := json.Marshal(map[string]string{"task": taskID})
			_ = signal.SendCollectResult(context.Background(), bus, taskID, result, "")
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := server.SignalTask(ctx, &pb.SignalTaskRequest{
		Type:      pb.SignalType_SIGNAL_TYPE_COLLECT,
		TaskIds:   taskIDs,
		TimeoutMs: 500,
	})
	if err != nil {
		t.Fatalf("SignalTask: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %v", resp.Error)
	}
	if len(resp.Results) != len(taskIDs) {
		t.Fatalf("expected %d results, got %d", len(taskIDs), len(resp.Results))
	}
}
