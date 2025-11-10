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

func Init() error {
	level := getEnv("LOG_LEVEL", "INFO")
	format := getEnv("LOG_FORMAT", "json")
	detailed := getEnv("LOG_DETAILED", "false") == "true"

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		parseLogLevel(level),
	)

	opts := []zap.Option{zap.AddCallerSkip(1)}
	if detailed {
		opts = append(opts, zap.AddCaller())
	}

	logger := zap.New(core, opts...)
	globalLogger = logger.Sugar()

	return nil
}

func Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	globalLogger.With(traceFields(ctx)...).Debugw(msg, keysAndValues...)
}

func Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	globalLogger.With(traceFields(ctx)...).Infow(msg, keysAndValues...)
}

func Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	globalLogger.With(traceFields(ctx)...).Warnw(msg, keysAndValues...)
}

func Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	globalLogger.With(traceFields(ctx)...).Errorw(msg, keysAndValues...)
}

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


func DebugSkip(ctx context.Context, skip int, msg string, keysAndValues ...interface{}) {
	globalLogger.WithOptions(zap.AddCallerSkip(skip)).With(traceFields(ctx)...).Debugw(msg, keysAndValues...)
}

func InfoSkip(ctx context.Context, skip int, msg string, keysAndValues ...interface{}) {
	globalLogger.WithOptions(zap.AddCallerSkip(skip)).With(traceFields(ctx)...).Infow(msg, keysAndValues...)
}

func WarnSkip(ctx context.Context, skip int, msg string, keysAndValues ...interface{}) {
	globalLogger.WithOptions(zap.AddCallerSkip(skip)).With(traceFields(ctx)...).Warnw(msg, keysAndValues...)
}

func ErrorSkip(ctx context.Context, skip int, msg string, keysAndValues ...interface{}) {
	globalLogger.WithOptions(zap.AddCallerSkip(skip)).With(traceFields(ctx)...).Errorw(msg, keysAndValues...)
}

func ErrorWithErrSkip(ctx context.Context, skip int, msg string, err error, keysAndValues ...interface{}) {
	if trace.Enabled() {
		if span := ottrace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}
	args := append([]interface{}{"error", err}, keysAndValues...)
	globalLogger.WithOptions(zap.AddCallerSkip(skip)).With(traceFields(ctx)...).Errorw(msg, args...)
}


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
