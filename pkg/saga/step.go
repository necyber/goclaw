// Package saga provides orchestration-based distributed transaction primitives.
package saga

import (
	"context"
	"time"
)

// ActionFunc executes a forward step in a Saga.
type ActionFunc func(ctx context.Context, stepCtx *StepContext) (any, error)

// CompensationFunc executes the reverse operation for a step.
type CompensationFunc func(ctx context.Context, compensationCtx *CompensationContext) error

// StepContext carries runtime information for forward step execution.
type StepContext struct {
	SagaID  string
	StepID  string
	Input   any
	Results map[string]any
}

// CompensationContext carries runtime information for compensation execution.
type CompensationContext struct {
	SagaID     string
	StepID     string
	FailedStep string
	Failure    error
	Input      any
	Result     any
	AllResults map[string]any
}

// Step defines one executable unit in a Saga.
type Step struct {
	ID                 string
	Action             ActionFunc
	Compensation       CompensationFunc
	Dependencies       []string
	Timeout            time.Duration
	CompensationPolicy CompensationPolicy
}

// StepOption configures a step definition.
type StepOption func(step *Step) error

// Action configures the forward action function.
func Action(fn ActionFunc) StepOption {
	return func(step *Step) error {
		step.Action = fn
		return nil
	}
}

// Compensate configures the compensation function.
func Compensate(fn CompensationFunc) StepOption {
	return func(step *Step) error {
		step.Compensation = fn
		return nil
	}
}

// DependsOn configures upstream dependencies.
func DependsOn(stepIDs ...string) StepOption {
	return func(step *Step) error {
		step.Dependencies = append(step.Dependencies, stepIDs...)
		return nil
	}
}

// StepTimeout configures per-step timeout.
func StepTimeout(timeout time.Duration) StepOption {
	return func(step *Step) error {
		step.Timeout = timeout
		return nil
	}
}

// WithStepCompensationPolicy overrides compensation behavior for this step.
func WithStepCompensationPolicy(policy CompensationPolicy) StepOption {
	return func(step *Step) error {
		step.CompensationPolicy = policy
		return nil
	}
}
