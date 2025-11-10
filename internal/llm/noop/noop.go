package noop

import (
	"context"

	"llm-trading-bot/internal/types"
)

type NoopDecider struct{}

func NewNoopDecider() *NoopDecider {
	return &NoopDecider{}
}

func (d *NoopDecider) Decide(ctx context.Context, symbol string, latest types.Candle, inds types.Indicators, ctxmap map[string]any) (types.Decision, error) {
	return types.Decision{
		Action:     "HOLD",
		Reason:     "noop_decider_fallback",
		Confidence: 0.0,
	}, nil
}
