package interfaces

import (
	"context"

	"llm-trading-bot/internal/types"
)

type Decider interface {
	Decide(ctx context.Context, symbol string, latest types.Candle, inds types.Indicators, contextData map[string]any) (types.Decision, error)
}
