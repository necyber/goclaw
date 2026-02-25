package interceptors

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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

		metrics.requests.WithLabelValues(info.FullMethod, code.String()).Inc()
		metrics.duration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
		if err != nil {
			metrics.errors.WithLabelValues(info.FullMethod, code.String()).Inc()
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

		metrics.streamDuration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
		metrics.streamMessages.WithLabelValues(info.FullMethod, "recv").Add(float64(wrapped.recvCount))
		metrics.streamMessages.WithLabelValues(info.FullMethod, "sent").Add(float64(wrapped.sendCount))
		if err != nil {
			metrics.streamErrors.WithLabelValues(info.FullMethod, code.String()).Inc()
		}

		return err
	}
}

type metricsServerStream struct {
	grpc.ServerStream
	recvCount int64
	sendCount int64
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
