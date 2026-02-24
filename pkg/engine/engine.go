// Package engine provides the core orchestration engine for multi-agent systems.
package engine

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/lane"
)

// appLogger is the subset of the logger.Logger interface used by the engine.
// Using an interface avoids a circular import with pkg/logger.
type appLogger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

const defaultLaneName = "default"

// engineState represents the lifecycle state of the engine.
type engineState int32

const (
	stateIdle    engineState = iota
	stateRunning
	stateStopped
	stateError
)

// WorkflowStatus represents the overall result of a workflow execution.
type WorkflowStatus int

const (
	WorkflowStatusSuccess WorkflowStatus = iota
	WorkflowStatusFailed
	WorkflowStatusCancelled
)

// Workflow is the unit submitted to the engine for execution.
type Workflow struct {
	// ID is a unique identifier for this workflow instance.
	ID string
	// Tasks is the list of tasks that form the DAG.
	Tasks []*dag.Task
	// TaskFns maps task IDs to their execution functions.
	// Tasks without an entry will be executed as no-ops.
	TaskFns map[string]func(context.Context) error
}

// WorkflowResult holds the outcome of a completed workflow.
type WorkflowResult struct {
	WorkflowID  string
	Status      WorkflowStatus
	TaskResults map[string]*TaskResult
	Error       error
}

// Engine is the core orchestration engine.
type Engine struct {
	cfg         *config.Config
	logger      appLogger
	laneManager *lane.Manager
	scheduler   *Scheduler
	state       atomic.Int32
}

// New creates a new Engine from the given configuration and logger.
func New(cfg *config.Config, logger appLogger) (*Engine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		logger = &nopLogger{}
	}
	e := &Engine{
		cfg:    cfg,
		logger: logger,
	}
	e.state.Store(int32(stateIdle))
	return e, nil
}

// Start initialises the engine and starts the lane manager.
func (e *Engine) Start(ctx context.Context) error {
	if engineState(e.state.Load()) == stateRunning {
		return fmt.Errorf("engine is already running")
	}

	e.logger.Info("starting engine", "app", e.cfg.App.Name)

	// Create lane manager and register the default lane.
	e.laneManager = lane.NewManager()

	queueSize := e.cfg.Orchestration.Queue.Size
	if queueSize <= 0 {
		queueSize = 1000
	}
	concurrency := e.cfg.Orchestration.MaxAgents
	if concurrency <= 0 {
		concurrency = 4
	}

	defaultCfg := &lane.Config{
		Name:           defaultLaneName,
		Capacity:       queueSize,
		MaxConcurrency: concurrency,
		Backpressure:   lane.Block,
	}
	if _, err := e.laneManager.Register(defaultCfg); err != nil {
		e.state.Store(int32(stateError))
		return fmt.Errorf("failed to register default lane: %w", err)
	}

	// Create scheduler (tracker is per-workflow, created in Submit).
	e.scheduler = newScheduler(newStateTracker(), e.logger)

	e.state.Store(int32(stateRunning))
	e.logger.Info("engine started")
	return nil
}

// Stop gracefully shuts down the engine.
func (e *Engine) Stop(ctx context.Context) error {
	if engineState(e.state.Load()) != stateRunning {
		return nil
	}

	e.logger.Info("stopping engine")

	if e.laneManager != nil {
		if err := e.laneManager.Close(ctx); err != nil {
			e.state.Store(int32(stateError))
			return fmt.Errorf("error closing lane manager: %w", err)
		}
	}

	e.state.Store(int32(stateStopped))
	e.logger.Info("engine stopped")
	return nil
}

// Submit compiles the workflow DAG and executes it, returning the result.
// Submit blocks until the workflow completes or ctx is cancelled.
func (e *Engine) Submit(ctx context.Context, wf *Workflow) (*WorkflowResult, error) {
	if engineState(e.state.Load()) != stateRunning {
		return nil, &EngineNotRunningError{}
	}

	e.logger.Info("submitting workflow", "workflow_id", wf.ID, "tasks", len(wf.Tasks))

	// Build DAG graph.
	g := dag.NewGraph()
	for _, t := range wf.Tasks {
		// Assign default lane if not set.
		if t.Lane == "" {
			t.Lane = defaultLaneName
		}
		if err := g.AddTask(t); err != nil {
			return nil, &WorkflowCompileError{WorkflowID: wf.ID, Cause: err}
		}
	}

	// Compile to execution plan.
	plan, err := g.Compile()
	if err != nil {
		return nil, &WorkflowCompileError{WorkflowID: wf.ID, Cause: err}
	}

	e.logger.Debug("workflow compiled",
		"workflow_id", wf.ID,
		"layers", plan.TotalLayers,
		"max_parallel", plan.MaxParallel,
	)

	// Initialise per-workflow state tracker.
	tracker := newStateTracker()
	taskIDs := make([]string, 0, len(wf.Tasks))
	for _, t := range wf.Tasks {
		taskIDs = append(taskIDs, t.ID)
	}
	tracker.InitTasks(taskIDs)

	// Create a scheduler with this workflow's tracker.
	sched := newScheduler(tracker, e.logger)

	taskFns := wf.TaskFns
	if taskFns == nil {
		taskFns = make(map[string]func(context.Context) error)
	}

	// Execute.
	schedErr := sched.Schedule(ctx, plan, taskFns)

	status := WorkflowStatusSuccess
	if schedErr != nil {
		if ctx.Err() != nil {
			status = WorkflowStatusCancelled
		} else {
			status = WorkflowStatusFailed
		}
	}

	result := &WorkflowResult{
		WorkflowID:  wf.ID,
		Status:      status,
		TaskResults: tracker.Results(),
		Error:       schedErr,
	}

	e.logger.Info("workflow complete",
		"workflow_id", wf.ID,
		"status", status,
		"error", schedErr,
	)

	return result, schedErr
}

// State returns the current engine state as a string.
func (e *Engine) State() string {
	switch engineState(e.state.Load()) {
	case stateIdle:
		return "idle"
	case stateRunning:
		return "running"
	case stateStopped:
		return "stopped"
	case stateError:
		return "error"
	default:
		return "unknown"
	}
}

// nopLogger is a no-op implementation of appLogger used when no logger is provided.
type nopLogger struct{}

func (n *nopLogger) Debug(msg string, args ...any) {}
func (n *nopLogger) Info(msg string, args ...any)  {}
func (n *nopLogger) Warn(msg string, args ...any)  {}
func (n *nopLogger) Error(msg string, args ...any) {}
