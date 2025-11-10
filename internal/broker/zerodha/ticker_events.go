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

// Event handler implementations

func (tm *tickerManager) onConnect() {
	logger.Info(context.Background(), "WebSocket connected successfully")
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
	logger.Info(context.Background(), "WebSocket reconnecting",
		"attempt", attempt,
		"delay", delay,
	)
}

func (tm *tickerManager) onNoReconnect(attempt int) {
	logger.Warn(context.Background(), "WebSocket reconnection failed - giving up",
		"attempts", attempt,
	)
}

func (tm *tickerManager) onTick(tick models.Tick) {
	symbol := tm.mapper.getSymbol(tick.InstrumentToken)
	if symbol == "" {
		return
	}

	// Convert tick to candle format
	candle := tm.convertTickToCandle(tick)

	// Add to candle cache
	tm.cache.addCandle(symbol, candle)
}

func (tm *tickerManager) onOrderUpdate(order kiteconnect.Order) {
	// Order updates can be logged if needed
	logger.Debug(context.Background(), "Order update received",
		"order_id", order.OrderID,
		"status", order.Status,
		"symbol", order.TradingSymbol,
	)
}

// convertTickToCandle converts a WebSocket tick to candle format
func (tm *tickerManager) convertTickToCandle(tick models.Tick) types.Candle {
	// TODO: Aggregate ticks into proper 1-minute candles
	// For now, treat each tick as a candle point
	return types.Candle{
		Ts:    tick.Timestamp.Time.Unix(),
		Open:  tick.OHLC.Open,
		High:  tick.OHLC.High,
		Low:   tick.OHLC.Low,
		Close: tick.LastPrice,
		Vol:   float64(tick.VolumeTraded),
	}
}
