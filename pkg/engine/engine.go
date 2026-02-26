// Package engine provides the core orchestration engine for multi-agent systems.
package engine

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/lane"
	"github.com/goclaw/goclaw/pkg/storage"
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
	stateIdle engineState = iota
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

// MetricsRecorder defines the interface for recording engine metrics.
type MetricsRecorder interface {
	RecordWorkflowSubmission(status string)
	RecordWorkflowDuration(status string, duration time.Duration)
	IncActiveWorkflows(status string)
	DecActiveWorkflows(status string)
	IncQueueDepth(laneName string)
	DecQueueDepth(laneName string)
	RecordWaitDuration(laneName string, duration time.Duration)
	RecordThroughput(laneName string)
}

// MemoryHub is the interface for the memory system used by the engine.
type MemoryHub interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Engine is the core orchestration engine.
type Engine struct {
	cfg         *config.Config
	logger      appLogger
	storage     storage.Storage
	laneManager *lane.Manager
	scheduler   *Scheduler
	metrics     MetricsRecorder
	memoryHub   MemoryHub
	state       atomic.Int32
}

// New creates a new Engine from the given configuration, logger, and storage.
func New(cfg *config.Config, logger appLogger, store storage.Storage, opts ...Option) (*Engine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		logger = &nopLogger{}
	}
	if store == nil {
		return nil, fmt.Errorf("storage cannot be nil")
	}
	e := &Engine{
		cfg:     cfg,
		logger:  logger,
		storage: store,
		metrics: &nopMetrics{},
	}
	e.state.Store(int32(stateIdle))

	// Apply options
	for _, opt := range opts {
		opt(e)
	}

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
	defaultLane, err := e.laneManager.Register(defaultCfg)
	if err != nil {
		e.state.Store(int32(stateError))
		return fmt.Errorf("failed to register default lane: %w", err)
	}

	// Set metrics on the default lane
	if channelLane, ok := defaultLane.(*lane.ChannelLane); ok {
		channelLane.SetMetrics(e.metrics)
	}

	// Create scheduler (tracker is per-workflow, created in Submit).
	e.scheduler = newScheduler(newStateTracker(), e.logger)

	// Start memory hub if configured
	if e.memoryHub != nil {
		if err := e.memoryHub.Start(ctx); err != nil {
			e.logger.Warn("failed to start memory hub", "error", err)
		} else {
			e.logger.Info("memory hub started")
		}
	}

	e.state.Store(int32(stateRunning))
	e.logger.Info("engine started")

	// Recover workflows from storage
	if err := e.RecoverWorkflows(ctx); err != nil {
		e.logger.Warn("workflow recovery completed with errors", "error", err)
	}

	return nil
}

// Stop gracefully shuts down the engine.
func (e *Engine) Stop(ctx context.Context) error {
	if engineState(e.state.Load()) != stateRunning {
		return nil
	}

	e.logger.Info("stopping engine")

	// Stop memory hub first
	if e.memoryHub != nil {
		if err := e.memoryHub.Stop(ctx); err != nil {
			e.logger.Warn("error stopping memory hub", "error", err)
		}
	}

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

	// Record workflow submission
	e.metrics.RecordWorkflowSubmission("pending")
	e.metrics.IncActiveWorkflows("running")
	defer e.metrics.DecActiveWorkflows("running")

	start := time.Now()

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
	statusStr := "completed"
	if schedErr != nil {
		if ctx.Err() != nil {
			status = WorkflowStatusCancelled
			statusStr = "cancelled"
		} else {
			status = WorkflowStatusFailed
			statusStr = "failed"
		}
	}

	// Record workflow duration
	duration := time.Since(start)
	e.metrics.RecordWorkflowDuration(statusStr, duration)
	e.metrics.RecordWorkflowSubmission(statusStr)

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

// RecoverWorkflows loads and resubmits workflows that were pending or running when the engine stopped.
func (e *Engine) RecoverWorkflows(ctx context.Context) error {
	e.logger.Info("starting workflow recovery")

	// List workflows with pending or running status
	filter := &storage.WorkflowFilter{
		Status: []string{"pending", "running"},
		Limit:  1000, // Reasonable batch size
		Offset: 0,
	}

	workflows, total, err := e.storage.ListWorkflows(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list workflows for recovery: %w", err)
	}

	if total == 0 {
		e.logger.Info("no workflows to recover")
		return nil
	}

	e.logger.Info("found workflows to recover", "count", total)

	var recoveryErrors []error
	recovered := 0
	skipped := 0

	for _, wf := range workflows {
		// Reset running tasks to pending for re-execution
		for _, task := range wf.TaskStatus {
			if task.Status == "running" {
				task.Status = "pending"
				task.StartedAt = nil
				task.CompletedAt = nil
				task.Error = ""
			}
		}

		// Reset workflow status to pending
		wf.Status = "pending"
		wf.StartedAt = nil
		wf.CompletedAt = nil
		wf.Error = ""

		// Save updated workflow state
		if err := e.storage.SaveWorkflow(ctx, wf); err != nil {
			e.logger.Error("failed to reset workflow for recovery",
				"workflow_id", wf.ID,
				"error", err)
			recoveryErrors = append(recoveryErrors, fmt.Errorf("workflow %s: %w", wf.ID, err))
			skipped++
			continue
		}

		e.logger.Info("recovered workflow", "workflow_id", wf.ID, "name", wf.Name)
		recovered++
	}

	e.logger.Info("workflow recovery completed",
		"recovered", recovered,
		"skipped", skipped,
		"total", total)

	if len(recoveryErrors) > 0 {
		return fmt.Errorf("recovery completed with %d errors", len(recoveryErrors))
	}

	return nil
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

// nopMetrics is a no-op implementation of MetricsRecorder used when no metrics are provided.
type nopMetrics struct{}

func (n *nopMetrics) RecordWorkflowSubmission(status string)                       {}
func (n *nopMetrics) RecordWorkflowDuration(status string, duration time.Duration) {}
func (n *nopMetrics) IncActiveWorkflows(status string)                             {}
func (n *nopMetrics) DecActiveWorkflows(status string)                             {}
func (n *nopMetrics) IncQueueDepth(laneName string)                                {}
func (n *nopMetrics) DecQueueDepth(laneName string)                                {}
func (n *nopMetrics) RecordWaitDuration(laneName string, duration time.Duration)   {}
func (n *nopMetrics) RecordThroughput(laneName string)                             {}
