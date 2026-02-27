package models

import "time"

// SagaSubmitRequest describes a Saga definition submission payload.
type SagaSubmitRequest struct {
	Name          string            `json:"name" validate:"required,min=1,max=100"`
	Policy        string            `json:"policy,omitempty" validate:"omitempty,oneof=auto manual skip"`
	TimeoutMS     int               `json:"timeout_ms,omitempty" validate:"omitempty,min=1"`
	StepTimeoutMS int               `json:"step_timeout_ms,omitempty" validate:"omitempty,min=1"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Input         map[string]any    `json:"input,omitempty"`
	Steps         []SagaStepRequest `json:"steps" validate:"required,min=1,dive"`
}

// SagaStepRequest defines one step in a submitted saga definition.
type SagaStepRequest struct {
	ID                 string   `json:"id" validate:"required,min=1,max=100"`
	DependsOn          []string `json:"depends_on,omitempty"`
	DelayMS            int      `json:"delay_ms,omitempty" validate:"omitempty,min=0"`
	ShouldFail         bool     `json:"should_fail,omitempty"`
	TimeoutMS          int      `json:"timeout_ms,omitempty" validate:"omitempty,min=1"`
	EnableCompensation bool     `json:"enable_compensation,omitempty"`
	SkipCompensation   bool     `json:"skip_compensation,omitempty"`
}

// SagaSubmitResponse is returned when a saga is accepted.
type SagaSubmitResponse struct {
	SagaID    string    `json:"saga_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// SagaStatusResponse returns current runtime information for one saga instance.
type SagaStatusResponse struct {
	SagaID         string         `json:"saga_id"`
	Name           string         `json:"name"`
	State          string         `json:"state"`
	CompletedSteps []string       `json:"completed_steps"`
	Compensated    []string       `json:"compensated_steps"`
	FailedStep     string         `json:"failed_step,omitempty"`
	FailureReason  string         `json:"failure_reason,omitempty"`
	StepResults    map[string]any `json:"step_results,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
}

// SagaSummary is one row in list response.
type SagaSummary struct {
	SagaID      string     `json:"saga_id"`
	Name        string     `json:"name"`
	State       string     `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// SagaListResponse is paginated list of saga summaries.
type SagaListResponse struct {
	Items  []SagaSummary `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// SagaCompensateRequest is used for manual compensation trigger.
type SagaCompensateRequest struct {
	Reason string `json:"reason,omitempty"`
}

// SagaRecoverRequest is used for manual recovery trigger.
type SagaRecoverRequest struct{}

// SagaActionResponse is returned by compensate/recover operations.
type SagaActionResponse struct {
	SagaID string `json:"saga_id"`
	State  string `json:"state"`
}
