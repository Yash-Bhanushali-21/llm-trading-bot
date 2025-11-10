package zerodha

import (
	"context"

	"llm-trading-bot/internal/types"
)

// TickerManager defines the interface for WebSocket ticker management
type TickerManager interface {
	// Start initializes and starts the WebSocket connection
	Start(ctx context.Context) error

	// Stop closes the WebSocket connection gracefully
	Stop(ctx context.Context)

	// Subscribe subscribes to symbols for live data streaming
	Subscribe(ctx context.Context, symbols []string) error

	// GetRecentCandles retrieves recent candles from cache
	GetRecentCandles(symbol string, n int) ([]types.Candle, error)
}

// newTickerManager creates a new WebSocket ticker manager instance
func newTickerManager(apiKey, accessToken, exchange string) TickerManager {
	return &tickerManager{
		apiKey:      apiKey,
		accessToken: accessToken,
		exchange:    exchange,
		cache:       newCandleCache(),
		mapper:      newInstrumentMapper(),
	}
}
