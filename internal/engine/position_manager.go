package engine

import (
	"context"
	"time"

	"llm-trading-bot/internal/logger"
)

type position struct {
	qty       int       // Current quantity held
	avg       float64   // Average entry price
	stop      float64   // Stop-loss price
	lastATR   float64   // Last ATR value for stop calculation
	entryTime time.Time // Time when position was opened (for time-based stops)
}

type positionManager struct {
	positions map[string]*position
}

func newPositionManager() *positionManager {
	return &positionManager{
		positions: make(map[string]*position),
	}
}

func (pm *positionManager) get(symbol string) *position {
	return pm.positions[symbol]
}

func (pm *positionManager) has(symbol string) bool {
	return pm.positions[symbol] != nil
}

//
func (pm *positionManager) addBuy(ctx context.Context, symbol string, qty int, price, atr, stopPrice float64) {
	p := pm.positions[symbol]
	if p == nil {
		p = &position{
			qty:       qty,
			avg:       price,
			stop:      stopPrice,
			lastATR:   atr,
			entryTime: time.Now(), // Set entry time for time-based stops
		}
		pm.positions[symbol] = p
	} else {
		totalCost := p.avg*float64(p.qty) + price*float64(qty)
		p.qty += qty
		p.avg = totalCost / float64(p.qty)
		p.lastATR = atr

		if stopPrice > p.stop {
			p.stop = stopPrice
		}
	}
}

//
//
func (pm *positionManager) reduceSell(ctx context.Context, symbol string, qty int, price float64) float64 {
	p := pm.positions[symbol]
	if p == nil {
		logger.Warn(ctx, "Attempted to sell with no position", "symbol", symbol, "qty", qty)
		return 0
	}

	p.qty -= qty

	realizedPnL := (price - p.avg) * float64(qty)


	if p.qty <= 0 {
		delete(pm.positions, symbol)
	}

	return realizedPnL
}

func (pm *positionManager) close(symbol string) {
	delete(pm.positions, symbol)
}

//
//
func (pm *positionManager) updateTrailingStop(ctx context.Context, symbol string, newStop, atr float64) bool {
	p := pm.positions[symbol]
	if p == nil || p.qty <= 0 {
		return false
	}

	p.lastATR = atr

	if newStop > p.stop {
		p.stop = newStop
		return true
	}

	return false
}
