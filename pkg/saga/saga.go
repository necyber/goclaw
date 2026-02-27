package saga

import (
	"fmt"
	"sort"
	"time"
)

// CompensationPolicy controls behavior after step failure.
type CompensationPolicy int

const (
	// AutoCompensate triggers reverse compensation immediately.
	AutoCompensate CompensationPolicy = iota
	// ManualCompensate waits for an explicit trigger before compensation.
	ManualCompensate
	// SkipCompensate does not execute compensation.
	SkipCompensate
)

// CompensationRetryConfig controls retry behavior for compensation execution.
type CompensationRetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// SagaDefinition describes a declarative Saga.
type SagaDefinition struct {
	Name               string
	Steps              map[string]*Step
	StepOrder          []string
	Timeout            time.Duration
	DefaultStepTimeout time.Duration
	Policy             CompensationPolicy
	Retry              CompensationRetryConfig
	MaxConcurrent      int
}

// Builder incrementally constructs SagaDefinition instances.
type Builder struct {
	def  *SagaDefinition
	errs []error
}

// New creates a Saga definition builder.
func New(name string) *Builder {
	return &Builder{
		def: &SagaDefinition{
			Name:               name,
			Steps:              make(map[string]*Step),
			StepOrder:          make([]string, 0),
			Policy:             AutoCompensate,
			MaxConcurrent:      100,
			DefaultStepTimeout: 30 * time.Second,
			Retry: CompensationRetryConfig{
				MaxRetries:     3,
				InitialBackoff: 100 * time.Millisecond,
				MaxBackoff:     5 * time.Second,
				BackoffFactor:  2.0,
			},
		},
	}
}

// Step appends a step to the saga definition.
func (b *Builder) Step(id string, opts ...StepOption) *Builder {
	step := &Step{
		ID:                 id,
		CompensationPolicy: AutoCompensate,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(step); err != nil {
			b.errs = append(b.errs, fmt.Errorf("step %q: %w", id, err))
		}
	}

	if _, exists := b.def.Steps[id]; exists {
		b.errs = append(b.errs, fmt.Errorf("duplicate step ID: %s", id))
		return b
	}

	b.def.Steps[id] = step
	b.def.StepOrder = append(b.def.StepOrder, id)
	return b
}

// WithTimeout sets the Saga-level timeout.
func (b *Builder) WithTimeout(timeout time.Duration) *Builder {
	b.def.Timeout = timeout
	return b
}

// WithDefaultStepTimeout sets default timeout for steps without explicit timeout.
func (b *Builder) WithDefaultStepTimeout(timeout time.Duration) *Builder {
	b.def.DefaultStepTimeout = timeout
	return b
}

// WithCompensationPolicy configures saga-level compensation policy.
func (b *Builder) WithCompensationPolicy(policy CompensationPolicy) *Builder {
	b.def.Policy = policy
	return b
}

// WithRetryConfig configures compensation retries.
func (b *Builder) WithRetryConfig(cfg CompensationRetryConfig) *Builder {
	b.def.Retry = cfg
	return b
}

// WithMaxConcurrent configures max concurrent execution within this Saga.
func (b *Builder) WithMaxConcurrent(max int) *Builder {
	b.def.MaxConcurrent = max
	return b
}

// Build validates and returns the saga definition.
func (b *Builder) Build() (*SagaDefinition, error) {
	if len(b.errs) > 0 {
		return nil, b.errs[0]
	}
	if err := b.def.Validate(); err != nil {
		return nil, err
	}
	return b.def.clone(), nil
}

// Validate validates saga structure and dependency DAG.
func (d *SagaDefinition) Validate() error {
	if d == nil {
		return fmt.Errorf("saga definition cannot be nil")
	}
	if d.Name == "" {
		return fmt.Errorf("saga name cannot be empty")
	}
	if len(d.Steps) == 0 {
		return fmt.Errorf("saga must define at least one step")
	}
	if d.MaxConcurrent <= 0 {
		return fmt.Errorf("max concurrent must be greater than 0")
	}
	if d.DefaultStepTimeout < 0 {
		return fmt.Errorf("default step timeout cannot be negative")
	}
	if d.Retry.MaxRetries < 0 {
		return fmt.Errorf("compensation max retries cannot be negative")
	}
	if d.Retry.BackoffFactor < 1 {
		return fmt.Errorf("compensation backoff factor must be >= 1")
	}

	for _, id := range d.StepOrder {
		step := d.Steps[id]
		if step == nil {
			return fmt.Errorf("step %q is nil", id)
		}
		if step.ID == "" {
			return fmt.Errorf("step ID cannot be empty")
		}
		if step.Action == nil {
			return fmt.Errorf("step %q missing action", step.ID)
		}
		if step.Timeout < 0 {
			return fmt.Errorf("step %q timeout cannot be negative", step.ID)
		}

		seenDeps := make(map[string]struct{}, len(step.Dependencies))
		for _, dep := range step.Dependencies {
			if dep == step.ID {
				return fmt.Errorf("step %q cannot depend on itself", step.ID)
			}
			if _, ok := d.Steps[dep]; !ok {
				return fmt.Errorf("step %q depends on unknown step %q", step.ID, dep)
			}
			if _, dup := seenDeps[dep]; dup {
				return fmt.Errorf("step %q has duplicate dependency %q", step.ID, dep)
			}
			seenDeps[dep] = struct{}{}
		}
	}

	_, err := d.TopologicalLayers()
	return err
}

// TopologicalLayers returns execution layers in topological order.
func (d *SagaDefinition) TopologicalLayers() ([][]string, error) {
	if d == nil {
		return nil, fmt.Errorf("saga definition cannot be nil")
	}

	indegree := make(map[string]int, len(d.Steps))
	edges := make(map[string][]string, len(d.Steps))
	for id := range d.Steps {
		indegree[id] = 0
	}

	for id, step := range d.Steps {
		for _, dep := range step.Dependencies {
			edges[dep] = append(edges[dep], id)
			indegree[id]++
		}
	}

	current := make([]string, 0)
	for id, deg := range indegree {
		if deg == 0 {
			current = append(current, id)
		}
	}
	sort.Strings(current)

	visited := 0
	layers := make([][]string, 0)
	for len(current) > 0 {
		layer := make([]string, len(current))
		copy(layer, current)
		layers = append(layers, layer)

		nextSet := make(map[string]struct{})
		for _, id := range current {
			visited++
			for _, to := range edges[id] {
				indegree[to]--
				if indegree[to] == 0 {
					nextSet[to] = struct{}{}
				}
			}
		}

		next := make([]string, 0, len(nextSet))
		for id := range nextSet {
			next = append(next, id)
		}
		sort.Strings(next)
		current = next
	}

	if visited != len(d.Steps) {
		return nil, fmt.Errorf("saga step dependencies contain a cycle")
	}

	return layers, nil
}

func (d *SagaDefinition) clone() *SagaDefinition {
	steps := make(map[string]*Step, len(d.Steps))
	for id, step := range d.Steps {
		if step == nil {
			continue
		}
		deps := make([]string, len(step.Dependencies))
		copy(deps, step.Dependencies)
		steps[id] = &Step{
			ID:                 step.ID,
			Action:             step.Action,
			Compensation:       step.Compensation,
			Dependencies:       deps,
			Timeout:            step.Timeout,
			CompensationPolicy: step.CompensationPolicy,
		}
	}

	order := make([]string, len(d.StepOrder))
	copy(order, d.StepOrder)

	return &SagaDefinition{
		Name:               d.Name,
		Steps:              steps,
		StepOrder:          order,
		Timeout:            d.Timeout,
		DefaultStepTimeout: d.DefaultStepTimeout,
		Policy:             d.Policy,
		Retry:              d.Retry,
		MaxConcurrent:      d.MaxConcurrent,
	}
}
