package engine

import (
	"context"
	"strings"
	"time"

	"llm-trading-bot/internal/logger"
)

// stopManager handles stop-loss calculations and checks.
type stopManager struct {
	mode        string  // "PCT", "ATR", "VOLATILITY", "TIME"
	pct         float64 // Stop-loss percentage (for PCT mode)
	atrMult     float64 // ATR multiplier (for ATR mode)
	minTick     float64 // Minimum price tick size
	trailing    bool    // Enable trailing stop
	maxHoldTime int     // Maximum hold time in seconds (for TIME mode)

	// Stop level presets (tight, medium, wide)
	stopLevels map[string]float64
}

// newStopManager creates a new stop manager with configuration.
func newStopManager(mode string, pct, atrMult, minTick float64, trailing bool) *stopManager {
	return &stopManager{
		mode:        strings.ToUpper(mode),
		pct:         pct,
		atrMult:     atrMult,
		minTick:     minTick,
		trailing:    trailing,
		maxHoldTime: 3600, // Default: 1 hour
		stopLevels: map[string]float64{
			"tight":  0.5,  // 0.5% stop loss
			"medium": 1.0,  // 1.0% stop loss
			"wide":   2.0,  // 2.0% stop loss
		},
	}
}

// calculateStopPrice computes the stop-loss price for a position.
//
// Multiple modes:
//   - PCT: entry * (1 - pct/100)
//   - ATR: entry - (atrMult * atr)
//   - VOLATILITY: entry - (atr * volatilityMultiplier)
//
// Parameters:
//   - entry: Entry price
//   - atr: Average True Range value
//
// Returns:
//   - stop: Calculated stop-loss price (rounded to tick size)
func (sm *stopManager) calculateStopPrice(entry, atr float64) float64 {
	var stop float64

	switch sm.mode {
	case "PCT":
		stop = entry * (1.0 - sm.pct/100.0)
	case "VOLATILITY":
		// Volatility-adjusted: wider stop in high volatility, tighter in low volatility
		// ATR as percentage of price determines volatility
		volatility := (atr / entry) * 100
		multiplier := sm.atrMult * (1.0 + volatility/50.0) // Adjust based on volatility
		stop = entry - (multiplier * atr)
	default:
		// ATR mode (default)
		stop = entry - (sm.atrMult * atr)
	}

	return roundToTick(stop, sm.minTick)
}

// calculateStopWithLevel calculates stop-loss using predefined level (tight/medium/wide)
func (sm *stopManager) calculateStopWithLevel(entry float64, level string) float64 {
	stopPct, ok := sm.stopLevels[level]
	if !ok {
		stopPct = sm.stopLevels["medium"] // Default to medium
	}

	stop := entry * (1.0 - stopPct/100.0)
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

// checkTimeBasedStop verifies if position should be closed due to time limit.
// Useful for preventing overnight holds or limiting position duration.
//
// Parameters:
//   - ctx: Context for logging
//   - symbol: Trading symbol
//   - pos: Current position details
//
// Returns:
//   - triggered: true if time limit exceeded
func (sm *stopManager) checkTimeBasedStop(ctx context.Context, symbol string, pos *position) bool {
	if pos == nil || pos.qty <= 0 {
		return false
	}

	// Check if position has exceeded max hold time
	holdDuration := time.Since(pos.entryTime)
	maxDuration := time.Duration(sm.maxHoldTime) * time.Second

	if holdDuration > maxDuration {
		logger.Warn(ctx, "Time-based stop triggered",
			"symbol", symbol,
			"event", "TIME_STOP_TRIGGERED",
			"hold_duration_seconds", holdDuration.Seconds(),
			"max_hold_seconds", sm.maxHoldTime,
			"position_qty", pos.qty,
			"position_avg", pos.avg,
			"entry_time", pos.entryTime,
		)
		return true
	}

	return false
}

// setMaxHoldTime sets the maximum hold time for positions in seconds.
func (sm *stopManager) setMaxHoldTime(seconds int) {
	sm.maxHoldTime = seconds
}

// getStopLevel returns the stop percentage for a given level preset.
func (sm *stopManager) getStopLevel(level string) float64 {
	if pct, ok := sm.stopLevels[level]; ok {
		return pct
	}
	return sm.stopLevels["medium"] // Default
}

// setStopLevel sets a custom stop percentage for a level preset.
func (sm *stopManager) setStopLevel(level string, pct float64) {
	sm.stopLevels[level] = pct
}
