package signal

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSendSteer(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	ch, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}

	err = SendSteer(context.Background(), bus, "task-1", map[string]interface{}{"rate": 0.5})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case sig := <-ch:
		p, err := ParseSteerPayload(sig)
		if err != nil {
			t.Fatal(err)
		}
		if p.Parameters["rate"] != 0.5 {
			t.Errorf("expected rate=0.5, got %v", p.Parameters["rate"])
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestSendSteer_EmptyTaskID(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	err := SendSteer(context.Background(), bus, "", map[string]interface{}{"x": 1})
	if err == nil {
		t.Error("expected error for empty task_id")
	}
}

func TestSendSteer_EmptyParams(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	err := SendSteer(context.Background(), bus, "task-1", nil)
	if err == nil {
		t.Error("expected error for empty params")
	}
}

func TestSendInterrupt(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	ch, err := bus.Subscribe(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}

	err = SendInterrupt(context.Background(), bus, "task-1", true, "user requested", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case sig := <-ch:
		p, err := ParseInterruptPayload(sig)
		if err != nil {
			t.Fatal(err)
		}
		if !p.Graceful {
			t.Error("expected graceful=true")
		}
		if p.Reason != "user requested" {
			t.Errorf("expected reason 'user requested', got %q", p.Reason)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestSendInterrupt_EmptyTaskID(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	err := SendInterrupt(context.Background(), bus, "", true, "reason", time.Second)
	if err == nil {
		t.Error("expected error for empty task_id")
	}
}

func TestSendCollectResult(t *testing.T) {
	bus := NewLocalBus(16)
	defer bus.Close()

	ch, err := bus.Subscribe(context.Background(), "collect:task-1")
	if err != nil {
		t.Fatal(err)
	}

	result, _ := json.Marshal(map[string]string{"output": "done"})
	err = SendCollectResult(context.Background(), bus, "task-1", result, "")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case sig := <-ch:
		p, err := ParseCollectPayload(sig)
		if err != nil {
			t.Fatal(err)
		}
		if p.Error != "" {
			t.Errorf("expected no error, got %q", p.Error)
		}
		var out map[string]string
		if err := json.Unmarshal(p.Result, &out); err != nil {
			t.Fatal(err)
		}
		if out["output"] != "done" {
			t.Errorf("expected output=done, got %v", out["output"])
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestParseSteerPayload_WrongType(t *testing.T) {
	sig := &Signal{Type: SignalInterrupt}
	_, err := ParseSteerPayload(sig)
	if err == nil {
		t.Error("expected error for wrong signal type")
	}
}

func TestParseInterruptPayload_WrongType(t *testing.T) {
	sig := &Signal{Type: SignalSteer}
	_, err := ParseInterruptPayload(sig)
	if err == nil {
		t.Error("expected error for wrong signal type")
	}
}

func TestParseCollectPayload_WrongType(t *testing.T) {
	sig := &Signal{Type: SignalSteer}
	_, err := ParseCollectPayload(sig)
	if err == nil {
		t.Error("expected error for wrong signal type")
	}
}
