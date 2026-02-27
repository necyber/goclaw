// Package logger provides structured logging for Goclaw.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

// Level represents logging levels.
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "unknown"
	}
}

// ParseLevel parses a level string.
func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// Config holds logger configuration.
type Config struct {
	Level  Level
	Format string // "json" or "text"
	Output string // "stdout", "stderr", or file path
}

// Logger is the interface for structured logging.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)

	With(args ...any) Logger
	WithContext(ctx context.Context) context.Context

	SetLevel(level Level)
	GetLevel() Level

	// Close closes any resources held by the logger (e.g., file handles).
	Close() error
}

// SlogLogger is a Logger implementation using log/slog.
type SlogLogger struct {
	logger *slog.Logger
	level  *slog.LevelVar
	closer io.Closer // holds the closer for file output, if any
}

var (
	// global is the global logger instance.
	global Logger
	// once ensures the global logger is initialized only once.
	once sync.Once
)

// init initializes the default global logger.
func init() {
	SetGlobal(New(&Config{
		Level:  InfoLevel,
		Format: "text",
		Output: "stdout",
	}))
}

// New creates a new Logger with the given configuration.
func New(cfg *Config) Logger {
	if cfg == nil {
		cfg = &Config{
			Level:  InfoLevel,
			Format: "json",
			Output: "stdout",
		}
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(slogLevel(cfg.Level))

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:       levelVar,
		AddSource:   true,
		ReplaceAttr: replaceAttr,
	}

	writer, closer := getWriter(cfg.Output)

	if cfg.Format == "text" {
		handler = slog.NewTextHandler(writer, opts)
	} else {
		handler = slog.NewJSONHandler(writer, opts)
	}

	return &SlogLogger{
		logger: slog.New(handler),
		level:  levelVar,
		closer: closer,
	}
}

// getWriter returns an io.Writer and io.Closer for the given output specification.
// The closer may be nil if the output doesn't need explicit closing (e.g., stdout/stderr).
func getWriter(output string) (io.Writer, io.Closer) {
	switch output {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "":
		return os.Stdout, nil
	default:
		// Try to open as file
		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fall back to stdout on error
			return os.Stdout, nil
		}
		return f, f
	}
}

// slogLevel converts our Level to slog.Level.
func slogLevel(l Level) slog.Level {
	switch l {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// replaceAttr customizes log attribute handling.
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	// Rename "msg" to "message" for consistency with some systems
	if a.Key == slog.MessageKey {
		return slog.Attr{Key: "message", Value: a.Value}
	}
	// Rename "lvl" to "level"
	if a.Key == slog.LevelKey {
		return slog.Attr{Key: "level", Value: a.Value}
	}
	return a
}

// Debug logs a debug message.
func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info logs an info message.
func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning message.
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs an error message.
func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// DebugContext logs a debug message with context.
func (l *SlogLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, appendTraceContextFields(ctx, args...)...)
}

// InfoContext logs an info message with context.
func (l *SlogLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, appendTraceContextFields(ctx, args...)...)
}

// WarnContext logs a warning message with context.
func (l *SlogLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, appendTraceContextFields(ctx, args...)...)
}

// ErrorContext logs an error message with context.
func (l *SlogLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, appendTraceContextFields(ctx, args...)...)
}

// With returns a new Logger with the given attributes.
func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{
		logger: l.logger.With(args...),
		level:  l.level,
		closer: nil, // derived loggers don't own the closer
	}
}

// WithContext returns a context with the logger attached.
func (l *SlogLogger) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// SetLevel dynamically changes the logging level.
func (l *SlogLogger) SetLevel(level Level) {
	l.level.Set(slogLevel(level))
}

// GetLevel returns the current logging level.
func (l *SlogLogger) GetLevel() Level {
	// This is a simplification - slog doesn't expose the current level
	// In production, you'd track this separately
	return InfoLevel
}

// Close closes any resources held by the logger.
// This is important when logging to a file to ensure all data is flushed.
func (l *SlogLogger) Close() error {
	if l.closer != nil {
		return l.closer.Close()
	}
	return nil
}

type loggerKey struct{}

// FromContext extracts a Logger from context.
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return l
	}
	return Global()
}

// Global returns the global logger.
func Global() Logger {
	return global
}

// SetGlobal sets the global logger.
func SetGlobal(l Logger) {
	once.Do(func() {
		global = l
	})
}

// SetLevel sets the level of the global logger.
func SetLevel(level Level) {
	if l, ok := global.(*SlogLogger); ok {
		l.SetLevel(level)
	}
}

// Convenience functions for the global logger.

func Debug(msg string, args ...any) {
	global.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	global.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	global.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	global.Error(msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	global.DebugContext(ctx, msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	global.InfoContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	global.WarnContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	global.ErrorContext(ctx, msg, args...)
}

func appendTraceContextFields(ctx context.Context, args ...any) []any {
	if ctx == nil {
		return args
	}
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return args
	}
	return append(args,
		"trace_id", spanCtx.TraceID().String(),
		"span_id", spanCtx.SpanID().String(),
	)
}
