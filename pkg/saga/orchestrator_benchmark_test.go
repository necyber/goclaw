package saga

import (
	"context"
	"errors"
	"testing"
)

func BenchmarkSagaExecuteSuccess(b *testing.B) {
	def, err := New("bench-success").
		Step("a", Action(func(context.Context, *StepContext) (any, error) { return "a", nil })).
		Step("b", Action(func(context.Context, *StepContext) (any, error) { return "b", nil }), DependsOn("a")).
		Step("c", Action(func(context.Context, *StepContext) (any, error) { return "c", nil }), DependsOn("a")).
		Build()
	if err != nil {
		b.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, execErr := orchestrator.Execute(context.Background(), def, nil); execErr != nil {
			b.Fatalf("Execute() error = %v", execErr)
		}
	}
}

func BenchmarkSagaExecuteCompensation(b *testing.B) {
	def, err := New("bench-compensate").
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "a", nil }),
			Compensate(func(context.Context, *CompensationContext) error { return nil }),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("fail") }),
			DependsOn("a"),
		).
		Build()
	if err != nil {
		b.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, execErr := orchestrator.Execute(context.Background(), def, nil); execErr == nil {
			b.Fatal("expected execute error")
		}
	}
}
