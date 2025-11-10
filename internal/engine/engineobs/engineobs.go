package engineobs

import (
	"context"
	"time"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/types"
)

type observableEngine struct {
	engine interfaces.Engine
}

var _ interfaces.Engine = (*observableEngine)(nil)

func Wrap(eng interfaces.Engine) interfaces.Engine {
	return &observableEngine{
		engine: eng,
	}
}

func (oe *observableEngine) Step(ctx context.Context, symbol string) (*types.StepResult, error) {
	ctx, span := trace.StartSpan(ctx, "engine.Step")
	defer span.End()

	start := time.Now()

	logger.InfoSkip(ctx, 1, "Starting trading cycle",
		"symbol", symbol,
	)

	result, err := oe.engine.Step(ctx, symbol)
	if err != nil {
		logger.ErrorWithErrSkip(ctx, 1, "Trading cycle failed", err,
			"symbol", symbol,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return nil, err
	}

	logger.InfoSkip(ctx, 1, "Trading cycle completed",
		"symbol", symbol,
		"action", result.Decision.Action,
		"confidence", result.Decision.Confidence,
		"reason", result.Decision.Reason,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}
