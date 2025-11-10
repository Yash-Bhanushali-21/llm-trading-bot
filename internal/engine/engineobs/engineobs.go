package engineobs

import (
	"context"
	"time"

	"llm-trading-bot/internal/engine"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/types"
)

// observableEngine wraps an IEngine with observability (logging & tracing)
type observableEngine struct {
	engine engine.IEngine
}

// Compile-time interface check
var _ engine.IEngine = (*observableEngine)(nil)

// Wrap wraps an engine with observability middleware
func Wrap(eng engine.IEngine) engine.IEngine {
	return &observableEngine{
		engine: eng,
	}
}

// Step executes one trading cycle with observability
func (oe *observableEngine) Step(ctx context.Context, symbol string) (*types.StepResult, error) {
	ctx, span := trace.StartSpan(ctx, "engine.Step")
	defer span.End()

	start := time.Now()

	// Use InfoSkip(1) to report the actual caller, not this middleware wrapper
	logger.InfoSkip(ctx, 1, "Starting trading cycle",
		"symbol", symbol,
	)

	// Call underlying engine
	result, err := oe.engine.Step(ctx, symbol)
	if err != nil {
		// Use ErrorWithErrSkip(1) to report the actual caller
		logger.ErrorWithErrSkip(ctx, 1, "Trading cycle failed", err,
			"symbol", symbol,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return nil, err
	}

	// Log successful cycle with result details
	logger.InfoSkip(ctx, 1, "Trading cycle completed",
		"symbol", symbol,
		"action", result.Decision.Action,
		"confidence", result.Decision.Confidence,
		"reason", result.Decision.Reason,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}
