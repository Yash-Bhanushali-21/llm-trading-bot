package logger

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger   *zap.SugaredLogger
	tracingEnabled bool
	tracer         trace.Tracer
	tracerProvider *sdktrace.TracerProvider
)

// Init initializes the global logger and tracer based on environment variables
func Init() error {
	level := getEnv("LOG_LEVEL", "INFO")
	format := getEnv("LOG_FORMAT", "json")
	detailed := getEnv("LOG_DETAILED", "false") == "true"
	tracingEnabled = getEnv("LOG_TRACING_ENABLED", "true") == "true"

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	// Create encoder
	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		parseLogLevel(level),
	)

	// Create logger with caller skip
	opts := []zap.Option{zap.AddCallerSkip(1)}
	if detailed {
		opts = append(opts, zap.AddCaller())
	}

	logger := zap.New(core, opts...)
	globalLogger = logger.Sugar()

	// Initialize OpenTelemetry tracer if enabled
	if tracingEnabled {
		if err := initTracer(); err != nil {
			globalLogger.Warnw("Failed to initialize tracer", "error", err)
			tracingEnabled = false
		}
	}

	return nil
}

// Shutdown gracefully shuts down the tracer provider
func Shutdown(ctx context.Context) error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(ctx)
	}
	return nil
}

// StartSpan starts a new OpenTelemetry span
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !tracingEnabled || tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, spanName, opts...)
}

// Debug logs a debug message
func Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	globalLogger.With(traceFields(ctx)...).Debugw(msg, keysAndValues...)
}

// Info logs an info message
func Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	globalLogger.With(traceFields(ctx)...).Infow(msg, keysAndValues...)
}

// Warn logs a warning message
func Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	globalLogger.With(traceFields(ctx)...).Warnw(msg, keysAndValues...)
}

// Error logs an error message
func Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	globalLogger.With(traceFields(ctx)...).Errorw(msg, keysAndValues...)
}

// ErrorWithErr logs an error with error object and records it in span
func ErrorWithErr(ctx context.Context, msg string, err error, keysAndValues ...interface{}) {
	if tracingEnabled {
		if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}
	args := append([]interface{}{"error", err}, keysAndValues...)
	globalLogger.With(traceFields(ctx)...).Errorw(msg, args...)
}

// Helper functions

func initTracer() error {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return err
	}

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

	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	tracer = otel.Tracer("llm-trading-bot")
	return nil
}

func traceFields(ctx context.Context) []interface{} {
	if !tracingEnabled {
		return nil
	}
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}
	return []interface{}{
		"trace_id", span.SpanContext().TraceID().String(),
		"span_id", span.SpanContext().SpanID().String(),
	}
}

func parseLogLevel(level string) zapcore.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return zapcore.DebugLevel
	case "INFO":
		return zapcore.InfoLevel
	case "WARN":
		return zapcore.WarnLevel
	case "ERROR":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
