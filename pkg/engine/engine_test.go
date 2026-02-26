package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/storage/memory"
)

// minConfig returns a minimal valid config for tests.
func minConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "development",
		},
		Server: config.ServerConfig{
			Port: 8080,
			GRPC: config.GRPCConfig{Port: 9090, MaxConnections: 100},
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents: 4,
			Queue:     config.QueueConfig{Type: "memory", Size: 100},
			Scheduler: config.SchedulerConfig{Type: "round_robin"},
		},
		Storage: config.StorageConfig{Type: "memory"},
	}
}

// --- StateTracker tests ---

func TestStateTracker_InitAndGet(t *testing.T) {
	tr := newStateTracker()
	tr.InitTasks([]string{"a", "b", "c"})

	for _, id := range []string{"a", "b", "c"} {
		r, ok := tr.GetResult(id)
		if !ok {
			t.Fatalf("expected result for task %q", id)
		}
		if r.State != TaskStatePending {
			t.Errorf("task %q: expected Pending, got %v", id, r.State)
		}
	}
}

func TestStateTracker_SetState(t *testing.T) {
	tr := newStateTracker()
	tr.InitTasks([]string{"t1"})

	tr.SetState("t1", TaskStateRunning)
	r, _ := tr.GetResult("t1")
	if r.State != TaskStateRunning {
		t.Errorf("expected Running, got %v", r.State)
	}
	if r.StartedAt.IsZero() {
		t.Error("StartedAt should be set when Running")
	}

	tr.SetState("t1", TaskStateCompleted)
	r, _ = tr.GetResult("t1")
	if r.State != TaskStateCompleted {
		t.Errorf("expected Completed, got %v", r.State)
	}
	if r.EndedAt.IsZero() {
		t.Error("EndedAt should be set when Completed")
	}
}

func TestStateTracker_SetFailed(t *testing.T) {
	tr := newStateTracker()
	tr.InitTasks([]string{"t1"})
	cause := errors.New("boom")

	tr.SetFailed("t1", cause, 2)
	r, _ := tr.GetResult("t1")
	if r.State != TaskStateFailed {
		t.Errorf("expected Failed, got %v", r.State)
	}
	if r.Error != cause {
		t.Errorf("expected cause error, got %v", r.Error)
	}
	if r.Retries != 2 {
		t.Errorf("expected 2 retries, got %d", r.Retries)
	}
}

func TestStateTracker_Concurrent(t *testing.T) {
	tr := newStateTracker()
	ids := make([]string, 50)
	for i := range ids {
		ids[i] = string(rune('a' + i%26))
	}
	tr.InitTasks(ids)

	done := make(chan struct{})
	for _, id := range ids {
		id := id
		go func() {
			tr.SetState(id, TaskStateRunning)
			tr.SetState(id, TaskStateCompleted)
			done <- struct{}{}
		}()
	}
	for range ids {
		<-done
	}
}

// --- taskRunner tests ---

func TestTaskRunner_Success(t *testing.T) {
	tr := newStateTracker()
	tr.InitTasks([]string{"t1"})

	called := false
	task := &dag.Task{ID: "t1", Name: "t1", Agent: "test", Lane: "default"}
	runner := newTaskRunner(task, tr, func(ctx context.Context) error {
		called = true
		return nil
	})

	if err := runner.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("fn was not called")
	}
	r, _ := tr.GetResult("t1")
	if r.State != TaskStateCompleted {
		t.Errorf("expected Completed, got %v", r.State)
	}
}

func TestTaskRunner_RetryOnFailure(t *testing.T) {
	tr := newStateTracker()
	tr.InitTasks([]string{"t1"})

	attempts := 0
	task := &dag.Task{ID: "t1", Name: "t1", Agent: "test", Lane: "default", Retries: 2}
	runner := newTaskRunner(task, tr, func(ctx context.Context) error {
		attempts++
		return errors.New("fail")
	})

	err := runner.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 3 { // 1 initial + 2 retries
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	var execErr *TaskExecutionError
	if !errors.As(err, &execErr) {
		t.Errorf("expected TaskExecutionError, got %T", err)
	}
}

func TestTaskRunner_ContextCancelled(t *testing.T) {
	tr := newStateTracker()
	tr.InitTasks([]string{"t1"})

	task := &dag.Task{ID: "t1", Name: "t1", Agent: "test", Lane: "default", Retries: 5}
	runner := newTaskRunner(task, tr, func(ctx context.Context) error {
		return errors.New("fail")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := runner.Execute(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Engine lifecycle tests ---

func TestEngine_New(t *testing.T) {
	eng, err := New(minConfig(), nil, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eng.State() != "idle" {
		t.Errorf("expected idle, got %s", eng.State())
	}
}

func TestEngine_New_NilConfig(t *testing.T) {
	_, err := New(nil, nil, memory.NewMemoryStorage())
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestEngine_StartStop(t *testing.T) {
	eng, _ := New(minConfig(), nil, memory.NewMemoryStorage())
	ctx := context.Background()

	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if eng.State() != "running" {
		t.Errorf("expected running, got %s", eng.State())
	}

	// Double start should fail.
	if err := eng.Start(ctx); err == nil {
		t.Error("expected error on second Start")
	}

	if err := eng.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if eng.State() != "stopped" {
		t.Errorf("expected stopped, got %s", eng.State())
	}
}

func TestEngine_StopIdleEngine(t *testing.T) {
	eng, _ := New(minConfig(), nil, memory.NewMemoryStorage())
	if err := eng.Stop(context.Background()); err != nil {
		t.Errorf("Stop on idle engine should return nil, got: %v", err)
	}
}

func TestEngine_SubmitNotRunning(t *testing.T) {
	eng, _ := New(minConfig(), nil, memory.NewMemoryStorage())
	_, err := eng.Submit(context.Background(), &Workflow{ID: "wf1"})
	if err == nil {
		t.Fatal("expected EngineNotRunningError")
	}
	var notRunning *EngineNotRunningError
	if !errors.As(err, &notRunning) {
		t.Errorf("expected EngineNotRunningError, got %T", err)
	}
}

// --- Integration: end-to-end 3-layer DAG ---

func TestEngine_Submit_ThreeLayerDAG(t *testing.T) {
	eng, _ := New(minConfig(), nil, memory.NewMemoryStorage())
	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer eng.Stop(ctx)

	// Layer 0: a, b (no deps)
	// Layer 1: c (depends on a, b)
	// Layer 2: d (depends on c)
	orderCh := make(chan string, 10)

	wf := &Workflow{
		ID: "wf-3layer",
		Tasks: []*dag.Task{
			{ID: "a", Name: "a", Agent: "test", Deps: []string{}},
			{ID: "b", Name: "b", Agent: "test", Deps: []string{}},
			{ID: "c", Name: "c", Agent: "test", Deps: []string{"a", "b"}},
			{ID: "d", Name: "d", Agent: "test", Deps: []string{"c"}},
		},
		TaskFns: map[string]func(context.Context) error{
			"a": func(ctx context.Context) error { orderCh <- "a"; return nil },
			"b": func(ctx context.Context) error { orderCh <- "b"; return nil },
			"c": func(ctx context.Context) error { orderCh <- "c"; return nil },
			"d": func(ctx context.Context) error { orderCh <- "d"; return nil },
		},
	}

	result, err := eng.Submit(ctx, wf)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if result.Status != WorkflowStatusSuccess {
		t.Errorf("expected Success, got %v", result.Status)
	}
	if len(result.TaskResults) != 4 {
		t.Errorf("expected 4 task results, got %d", len(result.TaskResults))
	}
	for id, r := range result.TaskResults {
		if r.State != TaskStateCompleted {
			t.Errorf("task %q: expected Completed, got %v", id, r.State)
		}
	}

	// Collect execution order.
	close(orderCh)
	executed := make([]string, 0, 4)
	for id := range orderCh {
		executed = append(executed, id)
	}
	if len(executed) != 4 {
		t.Errorf("expected 4 executions, got %d", len(executed))
	}

	// d must come after c, c must come after a and b.
	pos := make(map[string]int)
	for i, id := range executed {
		pos[id] = i
	}
	if pos["c"] <= pos["a"] || pos["c"] <= pos["b"] {
		t.Errorf("c should execute after a and b, order: %v", executed)
	}
	if pos["d"] <= pos["c"] {
		t.Errorf("d should execute after c, order: %v", executed)
	}
}

func TestEngine_Submit_TaskFailure(t *testing.T) {
	eng, _ := New(minConfig(), nil, memory.NewMemoryStorage())
	ctx := context.Background()
	eng.Start(ctx)
	defer eng.Stop(ctx)

	wf := &Workflow{
		ID: "wf-fail",
		Tasks: []*dag.Task{
			{ID: "t1", Name: "t1", Agent: "test"},
		},
		TaskFns: map[string]func(context.Context) error{
			"t1": func(ctx context.Context) error { return errors.New("task failed") },
		},
	}

	result, err := eng.Submit(ctx, wf)
	if err == nil {
		t.Fatal("expected error from failed task")
	}
	if result.Status != WorkflowStatusFailed {
		t.Errorf("expected Failed, got %v", result.Status)
	}
}

func TestEngine_Submit_CyclicDAG(t *testing.T) {
	eng, _ := New(minConfig(), nil, memory.NewMemoryStorage())
	ctx := context.Background()
	eng.Start(ctx)
	defer eng.Stop(ctx)

	wf := &Workflow{
		ID: "wf-cycle",
		Tasks: []*dag.Task{
			{ID: "a", Name: "a", Agent: "test", Deps: []string{"b"}},
			{ID: "b", Name: "b", Agent: "test", Deps: []string{"a"}},
		},
	}

	_, err := eng.Submit(ctx, wf)
	if err == nil {
		t.Fatal("expected compile error for cyclic DAG")
	}
	var compileErr *WorkflowCompileError
	if !errors.As(err, &compileErr) {
		t.Errorf("expected WorkflowCompileError, got %T", err)
	}
}

func TestEngine_Submit_ContextCancelled(t *testing.T) {
	eng, _ := New(minConfig(), nil, memory.NewMemoryStorage())
	ctx := context.Background()
	eng.Start(ctx)
	defer eng.Stop(ctx)

	submitCtx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	wf := &Workflow{
		ID: "wf-cancel",
		Tasks: []*dag.Task{
			{ID: "t1", Name: "t1", Agent: "test"},
		},
		TaskFns: map[string]func(context.Context) error{
			"t1": func(ctx context.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		},
	}

	result, _ := eng.Submit(submitCtx, wf)
	if result != nil && result.Status == WorkflowStatusSuccess {
		t.Error("expected non-success result for cancelled context")
	}
}
