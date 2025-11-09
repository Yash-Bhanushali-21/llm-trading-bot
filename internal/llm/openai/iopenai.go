package openai

import (
	"context"

	"llm-trading-bot/internal/types"
)

// Decider defines the interface for making trading decisions using LLM
type Decider interface {
	// Decide analyzes market data and returns a trading decision
	Decide(ctx context.Context, symbol string, latest types.Candle, inds types.Indicators, contextData map[string]any) (types.Decision, error)
}
