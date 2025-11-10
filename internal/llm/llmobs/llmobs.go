package llmobs

import (
	"context"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/types"
)

// observableDecider wraps a Decider with observability (logging & tracing)
type observableDecider struct {
	decider interfaces.Decider
}

// Compile-time interface check
var _ interfaces.Decider = (*observableDecider)(nil)

// Wrap wraps a decider with observability middleware
func Wrap(decider interfaces.Decider) interfaces.Decider {
	return &observableDecider{
		decider: decider,
	}
}

// Decide makes a trading decision with observability
func (od *observableDecider) Decide(
	ctx context.Context,
	symbol string,
	latest types.Candle,
	indicators types.Indicators,
	contextData map[string]any,
) (types.Decision, error) {
	ctx, span := trace.StartSpan(ctx, "llm.Decide")
	defer span.End()

	// Use DebugSkip(1) to report the actual caller, not this middleware wrapper
	logger.DebugSkip(ctx, 1, "Requesting trading decision",
		"symbol", symbol,
		"price", latest.Close,
		"rsi", indicators.RSI,
	)

	// Call underlying decider
	decision, err := od.decider.Decide(ctx, symbol, latest, indicators, contextData)
	if err != nil {
		// Use ErrorWithErrSkip(1) to report the actual caller
		logger.ErrorWithErrSkip(ctx, 1, "Failed to get trading decision", err,
			"symbol", symbol,
			"price", latest.Close,
		)
		return types.Decision{}, err
	}

	// Log decision result - use InfoSkip(1) to report the actual caller
	logger.InfoSkip(ctx, 1, "Trading decision received",
		"symbol", symbol,
		"action", decision.Action,
		"reason", decision.Reason,
		"confidence", decision.Confidence,
	)

	return decision, nil
}
