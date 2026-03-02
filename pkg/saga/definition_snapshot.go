package saga

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DefinitionSnapshot stores a serializable Saga definition by saga ID.
type DefinitionSnapshot struct {
	Name          string                   `json:"name"`
	Policy        string                   `json:"policy,omitempty"`
	TimeoutMS     int                      `json:"timeout_ms,omitempty"`
	StepTimeoutMS int                      `json:"step_timeout_ms,omitempty"`
	Metadata      map[string]string        `json:"metadata,omitempty"`
	Input         map[string]any           `json:"input,omitempty"`
	Steps         []DefinitionStepSnapshot `json:"steps"`
}

// DefinitionStepSnapshot stores one serializable step configuration.
type DefinitionStepSnapshot struct {
	ID                 string   `json:"id"`
	DependsOn          []string `json:"depends_on,omitempty"`
	DelayMS            int      `json:"delay_ms,omitempty"`
	ShouldFail         bool     `json:"should_fail,omitempty"`
	TimeoutMS          int      `json:"timeout_ms,omitempty"`
	EnableCompensation bool     `json:"enable_compensation,omitempty"`
	SkipCompensation   bool     `json:"skip_compensation,omitempty"`
}

// BuildDefinitionFromSnapshot rebuilds an executable SagaDefinition from a serialized snapshot.
func BuildDefinitionFromSnapshot(snapshot *DefinitionSnapshot) (*SagaDefinition, any, error) {
	if snapshot == nil {
		return nil, nil, fmt.Errorf("definition snapshot cannot be nil")
	}
	if strings.TrimSpace(snapshot.Name) == "" {
		return nil, nil, fmt.Errorf("definition snapshot name cannot be empty")
	}
	if len(snapshot.Steps) == 0 {
		return nil, nil, fmt.Errorf("definition snapshot steps cannot be empty")
	}

	builder := New(snapshot.Name)
	if snapshot.TimeoutMS > 0 {
		builder = builder.WithTimeout(time.Duration(snapshot.TimeoutMS) * time.Millisecond)
	}
	if snapshot.StepTimeoutMS > 0 {
		builder = builder.WithDefaultStepTimeout(time.Duration(snapshot.StepTimeoutMS) * time.Millisecond)
	}

	switch strings.ToLower(strings.TrimSpace(snapshot.Policy)) {
	case "", "auto":
		builder = builder.WithCompensationPolicy(AutoCompensate)
	case "manual":
		builder = builder.WithCompensationPolicy(ManualCompensate)
	case "skip":
		builder = builder.WithCompensationPolicy(SkipCompensate)
	default:
		return nil, nil, fmt.Errorf("unsupported policy: %s", snapshot.Policy)
	}

	for _, stepSpec := range snapshot.Steps {
		stepSpec := stepSpec
		options := []StepOption{
			Action(func(ctx context.Context, stepCtx *StepContext) (any, error) {
				if stepSpec.DelayMS > 0 {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(time.Duration(stepSpec.DelayMS) * time.Millisecond):
					}
				}
				if stepSpec.ShouldFail {
					return nil, fmt.Errorf("step %s failed by request", stepSpec.ID)
				}
				return map[string]any{
					"step_id": stepSpec.ID,
					"saga_id": stepCtx.SagaID,
					"status":  "ok",
				}, nil
			}),
		}

		if len(stepSpec.DependsOn) > 0 {
			options = append(options, DependsOn(stepSpec.DependsOn...))
		}
		if stepSpec.TimeoutMS > 0 {
			options = append(options, StepTimeout(time.Duration(stepSpec.TimeoutMS)*time.Millisecond))
		}
		if stepSpec.SkipCompensation {
			options = append(options, WithStepCompensationPolicy(SkipCompensate))
		}
		if stepSpec.EnableCompensation {
			options = append(options, Compensate(func(ctx context.Context, _ *CompensationContext) error {
				if stepSpec.DelayMS > 0 {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(time.Duration(stepSpec.DelayMS) * time.Millisecond):
					}
				}
				return nil
			}))
		}

		builder = builder.Step(stepSpec.ID, options...)
	}

	definition, err := builder.Build()
	if err != nil {
		return nil, nil, err
	}

	return definition, cloneAnyMap(snapshot.Input), nil
}

// Clone returns a deep-enough copy of the snapshot for safe mutation by callers.
func (s *DefinitionSnapshot) Clone() *DefinitionSnapshot {
	if s == nil {
		return nil
	}
	steps := make([]DefinitionStepSnapshot, len(s.Steps))
	for i, step := range s.Steps {
		steps[i] = DefinitionStepSnapshot{
			ID:                 step.ID,
			DependsOn:          append([]string(nil), step.DependsOn...),
			DelayMS:            step.DelayMS,
			ShouldFail:         step.ShouldFail,
			TimeoutMS:          step.TimeoutMS,
			EnableCompensation: step.EnableCompensation,
			SkipCompensation:   step.SkipCompensation,
		}
	}
	return &DefinitionSnapshot{
		Name:          s.Name,
		Policy:        s.Policy,
		TimeoutMS:     s.TimeoutMS,
		StepTimeoutMS: s.StepTimeoutMS,
		Metadata:      cloneStringMap(s.Metadata),
		Input:         cloneAnyMap(s.Input),
		Steps:         steps,
	}
}

func cloneAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	copied := make(map[string]any, len(source))
	for k, v := range source {
		copied[k] = v
	}
	return copied
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	copied := make(map[string]string, len(source))
	for k, v := range source {
		copied[k] = v
	}
	return copied
}
