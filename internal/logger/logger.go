package logger

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	// Global logger instance
	globalLogger *slog.Logger
	// Log level controlled by environment variable
	logLevel slog.Level
	// Whether detailed logging is enabled
	detailedLogging bool
	// Whether tracing is enabled
	tracingEnabled bool
	// OpenTelemetry tracer
	tracer trace.Tracer
	// Tracer provider for shutdown
	tracerProvider *sdktrace.TracerProvider
)

// LogConfig holds logging configuration
type LogConfig struct {
	Level           string // DEBUG, INFO, WARN, ERROR
	Format          string // json or text
	DetailedLogging bool   // Enable detailed logs
	TracingEnabled  bool   // Enable OpenTelemetry tracing
}

// Init initializes the global logger and tracer based on environment variables
func Init() error {
	config := LoadConfigFromEnv()
	return InitWithConfig(config)
}

// LoadConfigFromEnv loads logging configuration from environment variables
func LoadConfigFromEnv() LogConfig {
	config := LogConfig{
		Level:           getEnvOrDefault("LOG_LEVEL", "INFO"),
		Format:          getEnvOrDefault("LOG_FORMAT", "json"),
		DetailedLogging: getEnvOrDefault("LOG_DETAILED", "false") == "true",
		TracingEnabled:  getEnvOrDefault("LOG_TRACING_ENABLED", "true") == "true",
	}
	return config
}

// InitWithConfig initializes the logger and tracer with specific configuration
func InitWithConfig(config LogConfig) error {
	// Parse log level
	logLevel = parseLogLevel(config.Level)
	detailedLogging = config.DetailedLogging
	tracingEnabled = config.TracingEnabled

	// Create handler options
	// Note: We manually add source information in logWithTrace to get correct caller location
	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: false, // We manually add source to get correct caller location
	}

	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)

	// Initialize OpenTelemetry tracer if tracing is enabled
	if tracingEnabled {
		if err := initTracer(); err != nil {
			globalLogger.Warn("Failed to initialize OpenTelemetry tracer, tracing disabled", "error", err)
			tracingEnabled = false
		}
	}

	return nil
}

// initTracer initializes the OpenTelemetry tracer
func initTracer() error {
	// Create stdout exporter for traces
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return err
	}

	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("llm-trading-bot"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return err
	}

	// Create tracer provider
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)

	// Get tracer instance
	tracer = otel.Tracer("llm-trading-bot")

	return nil
}

// Shutdown gracefully shuts down the tracer provider
func Shutdown(ctx context.Context) error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(ctx)
	}
	return nil
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// StartSpan starts a new OpenTelemetry span
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !tracingEnabled || tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, spanName, opts...)
}

// getTraceAttrs extracts trace ID and span ID from context for logging
func getTraceAttrs(ctx context.Context) []any {
	if !tracingEnabled {
		return nil
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}

	return []any{
		"trace_id", span.SpanContext().TraceID().String(),
		"span_id", span.SpanContext().SpanID().String(),
	}
}

// Debug logs a debug message
func Debug(ctx context.Context, msg string, args ...any) {
	if !detailedLogging {
		return
	}
	logWithTrace(ctx, slog.LevelDebug, msg, 2, args...)
}

// Info logs an info message
func Info(ctx context.Context, msg string, args ...any) {
	logWithTrace(ctx, slog.LevelInfo, msg, 2, args...)
}

// Warn logs a warning message
func Warn(ctx context.Context, msg string, args ...any) {
	logWithTrace(ctx, slog.LevelWarn, msg, 2, args...)
}

// Error logs an error message
func Error(ctx context.Context, msg string, args ...any) {
	logWithTrace(ctx, slog.LevelError, msg, 2, args...)
}

// ErrorWithErr logs an error message with an error object
func ErrorWithErr(ctx context.Context, msg string, err error, args ...any) {
	// Record error in span if present
	if tracingEnabled {
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}

	allArgs := append([]any{"error", err}, args...)
	logWithTrace(ctx, slog.LevelError, msg, 2, allArgs...)
}

// logWithTrace logs a message with trace ID and span ID if available
// skip parameter indicates how many stack frames to skip to get the actual caller
func logWithTrace(ctx context.Context, level slog.Level, msg string, skip int, args ...any) {
	if traceAttrs := getTraceAttrs(ctx); traceAttrs != nil {
		args = append(traceAttrs, args...)
	}

	// Add source information if detailed logging is enabled
	if detailedLogging {
		// Skip frames: runtime.Caller -> logWithTrace -> wrapper (Debug/Info/etc) -> actual caller
		// So we need to skip 'skip' frames to get to the actual caller
		if pc, file, line, ok := runtime.Caller(skip); ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				args = append(args, "source", slog.GroupValue(
					slog.String("function", fn.Name()),
					slog.String("file", file),
					slog.Int("line", line),
				))
			}
		}
	}

	globalLogger.Log(ctx, level, msg, args...)
}

// OperationTimer helps measure operation duration with OpenTelemetry spans
type OperationTimer struct {
	ctx    context.Context
	span   trace.Span
	start  time.Time
	fields []any
}

// StartOperation starts timing an operation with an OpenTelemetry span
func StartOperation(ctx context.Context, operation string, fields ...any) *OperationTimer {
	var span trace.Span
	if tracingEnabled {
		ctx, span = StartSpan(ctx, operation)

		// Add fields as span attributes
		attrs := make([]attribute.KeyValue, 0)
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				key := fields[i].(string)
				switch v := fields[i+1].(type) {
				case string:
					attrs = append(attrs, attribute.String(key, v))
				case int:
					attrs = append(attrs, attribute.Int(key, v))
				case int64:
					attrs = append(attrs, attribute.Int64(key, v))
				case float64:
					attrs = append(attrs, attribute.Float64(key, v))
				case bool:
					attrs = append(attrs, attribute.Bool(key, v))
				}
			}
		}
		span.SetAttributes(attrs...)
	}

	if detailedLogging {
		Debug(ctx, "Operation started", append([]any{"operation", operation}, fields...)...)
	}

	return &OperationTimer{
		ctx:    ctx,
		span:   span,
		start:  time.Now(),
		fields: fields,
	}
}

// End completes the operation timer and logs the duration
func (ot *OperationTimer) End(additionalFields ...any) {
	duration := time.Since(ot.start)

	if tracingEnabled && ot.span != nil {
		ot.span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
		for i := 0; i < len(additionalFields); i += 2 {
			if i+1 < len(additionalFields) {
				key := additionalFields[i].(string)
				switch v := additionalFields[i+1].(type) {
				case string:
					ot.span.SetAttributes(attribute.String(key, v))
				case int:
					ot.span.SetAttributes(attribute.Int(key, v))
				case float64:
					ot.span.SetAttributes(attribute.Float64(key, v))
				}
			}
		}
		ot.span.SetStatus(codes.Ok, "completed")
		ot.span.End()
	}

	if detailedLogging {
		fields := append(ot.fields, "duration_ms", duration.Milliseconds())
		fields = append(fields, additionalFields...)
		Debug(ot.ctx, "Operation completed", fields...)
	}
}

// EndWithError completes the operation timer with an error
func (ot *OperationTimer) EndWithError(err error, additionalFields ...any) {
	duration := time.Since(ot.start)

	if tracingEnabled && ot.span != nil {
		ot.span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
		ot.span.RecordError(err)
		ot.span.SetStatus(codes.Error, err.Error())
		ot.span.End()
	}

	fields := append(ot.fields, "duration_ms", duration.Milliseconds(), "error", err)
	fields = append(fields, additionalFields...)
	Error(ot.ctx, "Operation failed", fields...)
}

// GetContext returns the context with the span
func (ot *OperationTimer) GetContext() context.Context {
	return ot.ctx
}

// Decision logs a trading decision (always logged regardless of level)
func Decision(ctx context.Context, symbol, action string, confidence float64, reason string, fields ...any) {
	if tracingEnabled {
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			span.AddEvent("trading_decision", trace.WithAttributes(
				attribute.String("symbol", symbol),
				attribute.String("action", action),
				attribute.Float64("confidence", confidence),
				attribute.String("reason", reason),
			))
		}
	}

	allFields := append([]any{
		"type", "DECISION",
		"symbol", symbol,
		"action", action,
		"confidence", confidence,
		"reason", reason,
	}, fields...)
	logWithTrace(ctx, slog.LevelInfo, "Trading decision made", 2, allFields...)
}

// Trade logs a trade execution (always logged regardless of level)
func Trade(ctx context.Context, symbol, side string, qty int, price float64, orderID string, fields ...any) {
	if tracingEnabled {
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			span.AddEvent("trade_executed", trace.WithAttributes(
				attribute.String("symbol", symbol),
				attribute.String("side", side),
				attribute.Int("quantity", qty),
				attribute.Float64("price", price),
				attribute.String("order_id", orderID),
			))
		}
	}

	allFields := append([]any{
		"type", "TRADE",
		"symbol", symbol,
		"side", side,
		"quantity", qty,
		"price", price,
		"order_id", orderID,
	}, fields...)
	logWithTrace(ctx, slog.LevelInfo, "Trade executed", 2, allFields...)
}

// Risk logs a risk management event
func Risk(ctx context.Context, symbol, eventType string, fields ...any) {
	if tracingEnabled {
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			span.AddEvent("risk_event", trace.WithAttributes(
				attribute.String("symbol", symbol),
				attribute.String("event_type", eventType),
			))
		}
	}

	allFields := append([]any{
		"type", "RISK",
		"symbol", symbol,
		"event_type", eventType,
	}, fields...)
	logWithTrace(ctx, slog.LevelWarn, "Risk event", 2, allFields...)
}

// IsDebugEnabled returns whether debug logging is enabled
func IsDebugEnabled() bool {
	return detailedLogging
}

// IsTracingEnabled returns whether tracing is enabled
func IsTracingEnabled() bool {
	return tracingEnabled
}
