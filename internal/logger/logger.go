package logger

import (
	"context"
	"os"
	"strings"

	"llm-trading-bot/internal/trace"

	"go.opentelemetry.io/otel/codes"
	ottrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *zap.SugaredLogger

// Init initializes the global logger
func Init() error {
	level := getEnv("LOG_LEVEL", "INFO")
	format := getEnv("LOG_FORMAT", "json")
	detailed := getEnv("LOG_DETAILED", "false") == "true"

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

	return nil
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
	if trace.Enabled() {
		if span := ottrace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}
	args := append([]interface{}{"error", err}, keysAndValues...)
	globalLogger.With(traceFields(ctx)...).Errorw(msg, args...)
}

// Helper functions

func traceFields(ctx context.Context) []interface{} {
	if traceID, spanID, ok := trace.GetTraceFields(ctx); ok {
		return []interface{}{"trace_id", traceID, "span_id", spanID}
	}
	return nil
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
