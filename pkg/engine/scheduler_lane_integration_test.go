package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/storage/memory"
)

func TestEngineSubmit_FailsWhenLaneIsMissing(t *testing.T) {
	eng, err := New(minConfig(), nil, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := eng.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer eng.Stop(context.Background())

	wf := &Workflow{
		ID: "wf-missing-lane",
		Tasks: []*dag.Task{
			{ID: "t1", Name: "task-1", Agent: "test", Lane: "missing"},
		},
		TaskFns: map[string]func(context.Context) error{
			"t1": func(context.Context) error { return nil },
		},
	}

	_, err = eng.Submit(context.Background(), wf)
	if err == nil {
		t.Fatal("expected lane submission failure")
	}
	if !strings.Contains(err.Error(), "lane submit failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
