package saga

import (
	"fmt"
	"time"
)

// SagaState defines lifecycle of a Saga instance.
type SagaState int

const (
	SagaStateCreated SagaState = iota
	SagaStateRunning
	SagaStateCompleted
	SagaStateCompensating
	SagaStatePendingCompensation
	SagaStateCompensated
	SagaStateCompensationFailed
	SagaStateRecovering
)

var validTransitions = map[SagaState]map[SagaState]struct{}{
	SagaStateCreated: {
		SagaStateRunning: {},
	},
	SagaStateRunning: {
		SagaStateCompleted:           {},
		SagaStateCompensating:        {},
		SagaStatePendingCompensation: {},
		SagaStateRecovering:          {},
	},
	SagaStateCompensating: {
		SagaStateCompensated:        {},
		SagaStateCompensationFailed: {},
	},
	SagaStatePendingCompensation: {
		SagaStateCompensating: {},
	},
	SagaStateRecovering: {
		SagaStateRunning:      {},
		SagaStateCompensating: {},
	},
}

// String returns the string form of SagaState.
func (s SagaState) String() string {
	switch s {
	case SagaStateCreated:
		return "created"
	case SagaStateRunning:
		return "running"
	case SagaStateCompleted:
		return "completed"
	case SagaStateCompensating:
		return "compensating"
	case SagaStatePendingCompensation:
		return "pending-compensation"
	case SagaStateCompensated:
		return "compensated"
	case SagaStateCompensationFailed:
		return "compensation-failed"
	case SagaStateRecovering:
		return "recovering"
	default:
		return "unknown"
	}
}

// IsTerminal reports whether the state is terminal.
func (s SagaState) IsTerminal() bool {
	switch s {
	case SagaStateCompleted, SagaStateCompensated, SagaStateCompensationFailed:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks whether a state transition is valid.
func (s SagaState) CanTransitionTo(next SagaState) bool {
	if s == next {
		return true
	}
	validNext, ok := validTransitions[s]
	if !ok {
		return false
	}
	_, ok = validNext[next]
	return ok
}

// ValidateTransition validates transition semantics.
func ValidateTransition(current, next SagaState) error {
	if !current.CanTransitionTo(next) {
		return fmt.Errorf("invalid saga state transition: %s -> %s", current, next)
	}
	return nil
}

// SagaInstance is a runtime state snapshot for one saga execution.
type SagaInstance struct {
	ID             string
	DefinitionName string
	State          SagaState
	CompletedSteps []string
	Compensated    []string
	FailedStep     string
	FailureReason  string
	StepResults    map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
}

// NewSagaInstance creates a new runtime instance.
func NewSagaInstance(id string, def *SagaDefinition) *SagaInstance {
	now := time.Now().UTC()
	name := ""
	if def != nil {
		name = def.Name
	}
	return &SagaInstance{
		ID:             id,
		DefinitionName: name,
		State:          SagaStateCreated,
		CompletedSteps: make([]string, 0),
		Compensated:    make([]string, 0),
		StepResults:    make(map[string]any),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// TransitionTo applies a state transition.
func (i *SagaInstance) TransitionTo(next SagaState) error {
	if i == nil {
		return fmt.Errorf("saga instance cannot be nil")
	}
	if err := ValidateTransition(i.State, next); err != nil {
		return err
	}

	now := time.Now().UTC()
	if i.State == SagaStateCreated && next == SagaStateRunning {
		start := now
		i.StartedAt = &start
	}
	if next.IsTerminal() {
		done := now
		i.CompletedAt = &done
	}
	i.State = next
	i.UpdatedAt = now
	return nil
}

// MarkStepCompleted records a completed step and output.
func (i *SagaInstance) MarkStepCompleted(stepID string, result any) {
	if i == nil {
		return
	}
	i.CompletedSteps = append(i.CompletedSteps, stepID)
	if i.StepResults == nil {
		i.StepResults = make(map[string]any)
	}
	i.StepResults[stepID] = result
	i.UpdatedAt = time.Now().UTC()
}

// MarkStepCompensated records a compensated step.
func (i *SagaInstance) MarkStepCompensated(stepID string) {
	if i == nil {
		return
	}
	i.Compensated = append(i.Compensated, stepID)
	i.UpdatedAt = time.Now().UTC()
}

// SetFailure records failed step and error details.
func (i *SagaInstance) SetFailure(stepID string, err error) {
	if i == nil {
		return
	}
	i.FailedStep = stepID
	if err != nil {
		i.FailureReason = err.Error()
	}
	i.UpdatedAt = time.Now().UTC()
}
