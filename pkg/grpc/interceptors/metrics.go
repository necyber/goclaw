package interceptors

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// Metrics holds Prometheus collectors for gRPC instrumentation.
type Metrics struct {
	requests       *prometheus.CounterVec
	duration       *prometheus.HistogramVec
	inflight       *prometheus.GaugeVec
	errors         *prometheus.CounterVec
	streamMessages *prometheus.CounterVec
	streamDuration *prometheus.HistogramVec
	streamErrors   *prometheus.CounterVec
}

// NewMetrics creates gRPC metrics and registers them with the given registerer.
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	m := &Metrics{
		requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goclaw_grpc_requests_total",
				Help: "Total number of gRPC requests.",
			},
			[]string{"method", "status"},
		),
		duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "goclaw_grpc_request_duration_seconds",
				Help:    "Duration of gRPC requests.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		),
		inflight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "goclaw_grpc_in_flight",
				Help: "In-flight gRPC requests.",
			},
			[]string{"method"},
		),
		errors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goclaw_grpc_errors_total",
				Help: "Total number of gRPC errors.",
			},
			[]string{"method", "code"},
		),
		streamMessages: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goclaw_grpc_stream_messages_total",
				Help: "Total number of gRPC stream messages.",
			},
			[]string{"method", "direction"},
		),
		streamDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "goclaw_grpc_stream_duration_seconds",
				Help:    "Duration of gRPC streams.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		),
		streamErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goclaw_grpc_stream_errors_total",
				Help: "Total number of gRPC stream errors.",
			},
			[]string{"method", "code"},
		),
	}

	m.requests = registerCounterVec(registerer, m.requests)
	m.duration = registerHistogramVec(registerer, m.duration)
	m.inflight = registerGaugeVec(registerer, m.inflight)
	m.errors = registerCounterVec(registerer, m.errors)
	m.streamMessages = registerCounterVec(registerer, m.streamMessages)
	m.streamDuration = registerHistogramVec(registerer, m.streamDuration)
	m.streamErrors = registerCounterVec(registerer, m.streamErrors)

	return m
}

var defaultMetrics = NewMetrics(nil)

// MetricsUnaryInterceptor collects metrics for unary RPCs.
func MetricsUnaryInterceptor(metrics *Metrics) grpc.UnaryServerInterceptor {
	if metrics == nil {
		metrics = defaultMetrics
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		metrics.inflight.WithLabelValues(info.FullMethod).Inc()
		defer metrics.inflight.WithLabelValues(info.FullMethod).Dec()

		resp, err := handler(ctx, req)
		code := status.Code(err)
		exemplar, hasExemplar := traceExemplarLabels(ctx)

		incrementCounter(metrics.requests.WithLabelValues(info.FullMethod, code.String()), hasExemplar, exemplar)
		observeHistogram(metrics.duration.WithLabelValues(info.FullMethod), time.Since(start).Seconds(), hasExemplar, exemplar)
		if err != nil {
			incrementCounter(metrics.errors.WithLabelValues(info.FullMethod, code.String()), hasExemplar, exemplar)
		}

		return resp, err
	}
}

// MetricsStreamInterceptor collects metrics for streaming RPCs.
func MetricsStreamInterceptor(metrics *Metrics) grpc.StreamServerInterceptor {
	if metrics == nil {
		metrics = defaultMetrics
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		metrics.inflight.WithLabelValues(info.FullMethod).Inc()
		defer metrics.inflight.WithLabelValues(info.FullMethod).Dec()

		wrapped := &metricsServerStream{ServerStream: ss}
		err := handler(srv, wrapped)
		code := status.Code(err)
		exemplar, hasExemplar := traceExemplarLabels(ss.Context())

		observeHistogram(metrics.streamDuration.WithLabelValues(info.FullMethod), time.Since(start).Seconds(), hasExemplar, exemplar)
		addCounter(metrics.streamMessages.WithLabelValues(info.FullMethod, "recv"), float64(wrapped.recvCount), hasExemplar, exemplar)
		addCounter(metrics.streamMessages.WithLabelValues(info.FullMethod, "sent"), float64(wrapped.sendCount), hasExemplar, exemplar)
		if err != nil {
			incrementCounter(metrics.streamErrors.WithLabelValues(info.FullMethod, code.String()), hasExemplar, exemplar)
		}

		return err
	}
}

type metricsServerStream struct {
	grpc.ServerStream
	recvCount int64
	sendCount int64
}

type counterIncrementer interface {
	Inc()
}

type counterAdder interface {
	Add(float64)
}

type exemplarCounterAdder interface {
	AddWithExemplar(float64, prometheus.Labels)
}

type histogramObserver interface {
	Observe(float64)
}

type exemplarHistogramObserver interface {
	ObserveWithExemplar(float64, prometheus.Labels)
}

func (s *metricsServerStream) RecvMsg(m interface{}) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	s.recvCount++
	return nil
}

func (s *metricsServerStream) SendMsg(m interface{}) error {
	if err := s.ServerStream.SendMsg(m); err != nil {
		return err
	}
	s.sendCount++
	return nil
}

func traceExemplarLabels(ctx context.Context) (prometheus.Labels, bool) {
	if ctx == nil {
		return nil, false
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil, false
	}

	return prometheus.Labels{
		"trace_id": spanCtx.TraceID().String(),
		"span_id":  spanCtx.SpanID().String(),
	}, true
}

func incrementCounter(counter counterIncrementer, hasExemplar bool, exemplar prometheus.Labels) bool {
	if hasExemplar {
		if recorder, ok := any(counter).(exemplarCounterAdder); ok {
			recorder.AddWithExemplar(1, exemplar)
			return true
		}
	}
	counter.Inc()
	return false
}

func addCounter(counter counterAdder, value float64, hasExemplar bool, exemplar prometheus.Labels) bool {
	if value == 0 {
		return false
	}
	if hasExemplar {
		if recorder, ok := any(counter).(exemplarCounterAdder); ok {
			recorder.AddWithExemplar(value, exemplar)
			return true
		}
	}
	counter.Add(value)
	return false
}

func observeHistogram(observer histogramObserver, value float64, hasExemplar bool, exemplar prometheus.Labels) bool {
	if hasExemplar {
		if recorder, ok := any(observer).(exemplarHistogramObserver); ok {
			recorder.ObserveWithExemplar(value, exemplar)
			return true
		}
	}
	observer.Observe(value)
	return false
}

func registerCounterVec(registerer prometheus.Registerer, collector *prometheus.CounterVec) *prometheus.CounterVec {
	if err := registerer.Register(collector); err != nil {
		if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if vec, ok := existing.ExistingCollector.(*prometheus.CounterVec); ok {
				return vec
			}
		}
	}
	return collector
}

func registerHistogramVec(registerer prometheus.Registerer, collector *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := registerer.Register(collector); err != nil {
		if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if vec, ok := existing.ExistingCollector.(*prometheus.HistogramVec); ok {
				return vec
			}
		}
	}
	return collector
}

func registerGaugeVec(registerer prometheus.Registerer, collector *prometheus.GaugeVec) *prometheus.GaugeVec {
	if err := registerer.Register(collector); err != nil {
		if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if vec, ok := existing.ExistingCollector.(*prometheus.GaugeVec); ok {
				return vec
			}
		}
	}
	return collector
}
