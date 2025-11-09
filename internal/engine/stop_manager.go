package engine

import (
	"context"
	"strings"

	"llm-trading-bot/internal/logger"
)

// stopManager handles stop-loss calculations and checks.
type stopManager struct {
	mode     string  // "PCT" or "ATR"
	pct      float64 // Stop-loss percentage (for PCT mode)
	atrMult  float64 // ATR multiplier (for ATR mode)
	minTick  float64 // Minimum price tick size
	trailing bool    // Enable trailing stop
}

// newStopManager creates a new stop manager with configuration.
func newStopManager(mode string, pct, atrMult, minTick float64, trailing bool) *stopManager {
	return &stopManager{
		mode:     strings.ToUpper(mode),
		pct:      pct,
		atrMult:  atrMult,
		minTick:  minTick,
		trailing: trailing,
	}
}

// calculateStopPrice computes the stop-loss price for a position.
//
// Two modes:
//   - PCT: entry * (1 - pct/100)
//   - ATR: entry - (atrMult * atr)
//
// Parameters:
//   - entry: Entry price
//   - atr: Average True Range value
//
// Returns:
//   - stop: Calculated stop-loss price (rounded to tick size)
func (sm *stopManager) calculateStopPrice(entry, atr float64) float64 {
	var stop float64

	if sm.mode == "PCT" {
		stop = entry * (1.0 - sm.pct/100.0)
	} else {
		// ATR mode (default)
		stop = entry - (sm.atrMult * atr)
	}

	return roundToTick(stop, sm.minTick)
}

// checkStopLoss verifies if current price has hit the stop-loss.
//
// Parameters:
//   - ctx: Context for logging
//   - symbol: Trading symbol
//   - currentPrice: Current market price
//   - stopPrice: Stop-loss price
//   - position: Current position details (for logging)
//
// Returns:
//   - triggered: true if stop-loss was hit
func (sm *stopManager) checkStopLoss(ctx context.Context, symbol string, currentPrice, stopPrice float64, pos *position) bool {
	if pos == nil || pos.qty <= 0 {
		return false
	}

	if currentPrice <= stopPrice {
		unrealizedLoss := (currentPrice - pos.avg) * float64(pos.qty)

		logger.Warn(ctx, "Stop loss triggered",
			"symbol", symbol,
			"event", "STOP_LOSS_TRIGGERED",
			"current_price", currentPrice,
			"stop_price", stopPrice,
			"position_qty", pos.qty,
			"position_avg", pos.avg,
			"unrealized_loss", unrealizedLoss,
		)

		return true
	}

	return false
}

// isTrailingEnabled returns whether trailing stop is enabled.
func (sm *stopManager) isTrailingEnabled() bool {
	return sm.trailing
}
