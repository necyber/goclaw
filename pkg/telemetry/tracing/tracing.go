package tracing

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace/noop"
)

// ShutdownFunc shuts down tracing provider resources.
type ShutdownFunc func(ctx context.Context) error

var reportExporterFailure = func(err error, exporter, endpoint string, spanCount int) {
	logger.Warn("tracing exporter failed",
		"error", err,
		"exporter", exporter,
		"endpoint", endpoint,
		"span_count", spanCount,
	)
}

var newOTLPExporter = func(ctx context.Context, cfg config.TracingConfig) (sdktrace.SpanExporter, error) {
	endpoint := normalizeEndpoint(cfg.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("tracing endpoint cannot be empty")
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(cfg.Timeout),
		otlptracegrpc.WithInsecure(),
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
	}

	return otlptracegrpc.New(ctx, opts...)
}

type isolatingExporter struct {
	exporter sdktrace.SpanExporter
	kind     string
	endpoint string
}

func (e *isolatingExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if err := e.exporter.ExportSpans(ctx, spans); err != nil {
		reportExporterFailure(err, e.kind, e.endpoint, len(spans))
		return nil
	}
	return nil
}

func (e *isolatingExporter) Shutdown(ctx context.Context) error {
	return e.exporter.Shutdown(ctx)
}

// Init initializes process-wide OpenTelemetry tracing.
func Init(ctx context.Context, cfg config.TracingConfig, serviceName, serviceVersion string) (ShutdownFunc, error) {
	if !cfg.Enabled {
		otel.SetTracerProvider(noop.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return func(context.Context) error { return nil }, nil
	}

	if strings.TrimSpace(cfg.Exporter) == "" {
		return nil, fmt.Errorf("tracing exporter cannot be empty")
	}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, fmt.Errorf("tracing endpoint cannot be empty")
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("tracing timeout must be > 0")
	}

	exp, err := newOTLPExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create tracing exporter: %w", err)
	}
	exp = &isolatingExporter{
		exporter: exp,
		kind:     strings.ToLower(strings.TrimSpace(cfg.Exporter)),
		endpoint: normalizeEndpoint(cfg.Endpoint),
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		_ = exp.Shutdown(ctx)
		return nil, fmt.Errorf("create tracing resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(selectSampler(cfg)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(shutdownCtx context.Context) error {
		if err := tp.ForceFlush(shutdownCtx); err != nil {
			_ = tp.Shutdown(shutdownCtx)
			return fmt.Errorf("force flush tracing provider: %w", err)
		}
		if err := tp.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown tracing provider: %w", err)
		}
		return nil
	}, nil
}

func selectSampler(cfg config.TracingConfig) sdktrace.Sampler {
	switch strings.ToLower(strings.TrimSpace(cfg.Sampler)) {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))
	}
}

func normalizeEndpoint(endpoint string) string {
	raw := strings.TrimSpace(endpoint)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if parsed.Host != "" {
		return parsed.Host
	}
	return raw
}
