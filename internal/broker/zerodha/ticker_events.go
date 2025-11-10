package zerodha

import (
	"context"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/types"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"github.com/zerodha/gokiteconnect/v4/models"
)

// setupEventHandlers configures all WebSocket event callbacks
func (tm *tickerManager) setupEventHandlers() {
	tm.ticker.OnConnect(tm.onConnect)
	tm.ticker.OnError(tm.onError)
	tm.ticker.OnClose(tm.onClose)
	tm.ticker.OnReconnect(tm.onReconnect)
	tm.ticker.OnNoReconnect(tm.onNoReconnect)
	tm.ticker.OnTick(tm.onTick)
	tm.ticker.OnOrderUpdate(tm.onOrderUpdate)
}

// Event handlers

func (tm *tickerManager) onConnect() {
	// Connection established
}

func (tm *tickerManager) onError(err error) {
	logger.ErrorWithErr(context.Background(), "WebSocket error occurred", err)
}

func (tm *tickerManager) onClose(code int, reason string) {
	logger.Warn(context.Background(), "WebSocket connection closed",
		"code", code,
		"reason", reason,
	)
}

func (tm *tickerManager) onReconnect(attempt int, delay time.Duration) {
	// Reconnecting to WebSocket
}

func (tm *tickerManager) onNoReconnect(attempt int) {
	logger.Warn(context.Background(), "WebSocket reconnection failed - giving up",
		"attempts", attempt,
	)
}

func (tm *tickerManager) onTick(tick models.Tick) {
	// Get symbol from token
	symbol, exists := tm.tokenToSymbol[tick.InstrumentToken]
	if !exists {
		return
	}

	// Convert tick to candle and add to cache
	candle := types.Candle{
		Ts:    tick.Timestamp.Time.Unix(),
		Open:  tick.OHLC.Open,
		High:  tick.OHLC.High,
		Low:   tick.OHLC.Low,
		Close: tick.LastPrice,
		Vol:   float64(tick.VolumeTraded),
	}

	tm.addCandle(symbol, candle)
}

func (tm *tickerManager) onOrderUpdate(order kiteconnect.Order) {
	// Order update received via WebSocket
}
