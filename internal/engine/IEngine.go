package engine

import (
	"context"

	"llm-trading-bot/internal/broker/zerodha"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

// IEngine defines the interface for the trading engine.
// Implementations must provide methods to create a new engine and execute trading steps.
type IEngine interface {
	// Step executes one complete trading cycle for a given symbol:
	// 1. Fetches recent candles
	// 2. Calculates technical indicators
	// 3. Checks stop-loss triggers
	// 4. Gets LLM decision
	// 5. Executes orders
	// 6. Updates positions
	// 7. Manages trailing stops
	//
	// Returns StepResult containing decision, orders, and execution details.
	Step(ctx context.Context, symbol string) (*types.StepResult, error)
}

// New creates a new Engine instance with the provided configuration,
// broker, and LLM decider.
//
// Parameters:
//   - cfg: Trading configuration (risk, indicators, stop-loss settings)
//   - brk: Broker implementation for order execution and market data
//   - d: LLM decider for trading decisions
//
// Returns:
//   - IEngine: Initialized trading engine ready to execute steps
func New(cfg *store.Config, brk zerodha.Broker, d types.Decider) IEngine {
	return newEngine(cfg, brk, d)
}
