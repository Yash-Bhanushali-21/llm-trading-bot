package llmobs

import (
	"context"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/types"
)

type observableDecider struct {
	decider interfaces.Decider
}

var _ interfaces.Decider = (*observableDecider)(nil)

func Wrap(decider interfaces.Decider) interfaces.Decider {
	return &observableDecider{
		decider: decider,
	}
}

func (od *observableDecider) Decide(
	ctx context.Context,
	symbol string,
	latest types.Candle,
	indicators types.Indicators,
	contextData map[string]any,
) (types.Decision, error) {
	ctx, span := trace.StartSpan(ctx, "llm.Decide")
	defer span.End()

	logger.DebugSkip(ctx, 1, "Requesting trading decision",
		"symbol", symbol,
		"price", latest.Close,
		"rsi", indicators.RSI,
	)

	decision, err := od.decider.Decide(ctx, symbol, latest, indicators, contextData)
	if err != nil {
		logger.ErrorWithErrSkip(ctx, 1, "Failed to get trading decision", err,
			"symbol", symbol,
			"price", latest.Close,
		)
		return types.Decision{}, err
	}

	logger.InfoSkip(ctx, 1, "Trading decision received",
		"symbol", symbol,
		"action", decision.Action,
		"reason", decision.Reason,
		"confidence", decision.Confidence,
	)

	return decision, nil
}
