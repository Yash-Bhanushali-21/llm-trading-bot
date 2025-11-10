package engine

import (
	"context"
	"time"

	"llm-trading-bot/internal/logger"
)

// position represents an open trading position for a symbol.
type position struct {
	qty       int       // Current quantity held
	avg       float64   // Average entry price
	stop      float64   // Stop-loss price
	lastATR   float64   // Last ATR value for stop calculation
	entryTime time.Time // Time when position was opened (for time-based stops)
}

// positionManager handles all position tracking and updates.
type positionManager struct {
	positions map[string]*position
}

// newPositionManager creates a new position manager with empty positions map.
func newPositionManager() *positionManager {
	return &positionManager{
		positions: make(map[string]*position),
	}
}

// get retrieves the current position for a symbol.
// Returns nil if no position exists.
func (pm *positionManager) get(symbol string) *position {
	return pm.positions[symbol]
}

// has checks if a position exists for the symbol.
func (pm *positionManager) has(symbol string) bool {
	return pm.positions[symbol] != nil
}

// addBuy updates position after a BUY order execution.
// Calculates new average price and quantity.
//
// Parameters:
//   - ctx: Context for logging
//   - symbol: Trading symbol
//   - qty: Quantity bought
//   - price: Execution price
//   - atr: Current ATR value
//   - stopPrice: Calculated stop-loss price
func (pm *positionManager) addBuy(ctx context.Context, symbol string, qty int, price, atr, stopPrice float64) {
	p := pm.positions[symbol]
	if p == nil {
		// New position
		p = &position{
			qty:       qty,
			avg:       price,
			stop:      stopPrice,
			lastATR:   atr,
			entryTime: time.Now(), // Set entry time for time-based stops
		}
		pm.positions[symbol] = p
	} else {
		// Add to existing position
		// Calculate new average price
		totalCost := p.avg*float64(p.qty) + price*float64(qty)
		p.qty += qty
		p.avg = totalCost / float64(p.qty)
		p.lastATR = atr

		// Update stop if new stop is higher
		if stopPrice > p.stop {
			p.stop = stopPrice
		}
		// Position update logged via middleware
	}
}

// reduceSell updates position after a SELL order execution.
// Calculates realized P&L and removes position if fully closed.
//
// Parameters:
//   - ctx: Context for logging
//   - symbol: Trading symbol
//   - qty: Quantity sold
//   - price: Execution price
//
// Returns:
//   - realizedPnL: Profit or loss from the sale
func (pm *positionManager) reduceSell(ctx context.Context, symbol string, qty int, price float64) float64 {
	p := pm.positions[symbol]
	if p == nil {
		logger.Warn(ctx, "Attempted to sell with no position", "symbol", symbol, "qty", qty)
		return 0
	}

	p.qty -= qty

	// Calculate realized P&L
	realizedPnL := (price - p.avg) * float64(qty)

	// Position reduction logged via middleware

	// Close position if fully sold
	if p.qty <= 0 {
		delete(pm.positions, symbol)
	}

	return realizedPnL
}

// close removes a position (used for stop-loss triggers).
func (pm *positionManager) close(symbol string) {
	delete(pm.positions, symbol)
}

// updateTrailingStop updates the stop-loss price if the new stop is higher.
// Only updates if trailing stop is enabled and there's an active position.
//
// Parameters:
//   - ctx: Context for logging
//   - symbol: Trading symbol
//   - newStop: New stop-loss price
//   - atr: Current ATR value
//
// Returns:
//   - updated: true if stop was updated
func (pm *positionManager) updateTrailingStop(ctx context.Context, symbol string, newStop, atr float64) bool {
	p := pm.positions[symbol]
	if p == nil || p.qty <= 0 {
		return false
	}

	p.lastATR = atr

	// Only trail up, never down
	if newStop > p.stop {
		p.stop = newStop
		// Trailing stop update logged via middleware
		return true
	}

	return false
}
