package engine

import (
	"context"
	"strings"
	"time"

	"llm-trading-bot/internal/logger"
)

type stopManager struct {
	mode        string  // "PCT", "ATR", "VOLATILITY", "TIME"
	pct         float64 // Stop-loss percentage (for PCT mode)
	atrMult     float64 // ATR multiplier (for ATR mode)
	minTick     float64 // Minimum price tick size
	trailing    bool    // Enable trailing stop
	maxHoldTime int     // Maximum hold time in seconds (for TIME mode)

	stopLevels map[string]float64
}

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

//
//
//
func (sm *stopManager) calculateStopPrice(entry, atr float64) float64 {
	var stop float64

	switch sm.mode {
	case "PCT":
		stop = entry * (1.0 - sm.pct/100.0)
	case "VOLATILITY":
		volatility := (atr / entry) * 100
		multiplier := sm.atrMult * (1.0 + volatility/50.0) // Adjust based on volatility
		stop = entry - (multiplier * atr)
	default:
		stop = entry - (sm.atrMult * atr)
	}

	return roundToTick(stop, sm.minTick)
}

func (sm *stopManager) calculateStopWithLevel(entry float64, level string) float64 {
	stopPct, ok := sm.stopLevels[level]
	if !ok {
		stopPct = sm.stopLevels["medium"] // Default to medium
	}

	stop := entry * (1.0 - stopPct/100.0)
	return roundToTick(stop, sm.minTick)
}

//
//
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

func (sm *stopManager) isTrailingEnabled() bool {
	return sm.trailing
}

//
//
func (sm *stopManager) checkTimeBasedStop(ctx context.Context, symbol string, pos *position) bool {
	if pos == nil || pos.qty <= 0 {
		return false
	}

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

func (sm *stopManager) setMaxHoldTime(seconds int) {
	sm.maxHoldTime = seconds
}

func (sm *stopManager) getStopLevel(level string) float64 {
	if pct, ok := sm.stopLevels[level]; ok {
		return pct
	}
	return sm.stopLevels["medium"] // Default
}

func (sm *stopManager) setStopLevel(level string, pct float64) {
	sm.stopLevels[level] = pct
}
