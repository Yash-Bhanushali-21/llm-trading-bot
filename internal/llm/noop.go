package llm

import "llm-trading-bot/internal/types"

// NoopDecider is a fallback decider used when no LLM (like OpenAI) is configured.
type NoopDecider struct{}

// NewNoopDecider returns a new instance that always decides HOLD.
func NewNoopDecider() *NoopDecider {
	return &NoopDecider{}
}

// Decide implements the Decider interface. It always returns HOLD with 0 confidence.
func (d *NoopDecider) Decide(symbol string, latest types.Candle, inds types.Indicators, ctxmap map[string]any) (types.Decision, error) {
	return types.Decision{
		Action:     "HOLD",
		Reason:     "noop_decider_fallback",
		Confidence: 0.0,
	}, nil
}
