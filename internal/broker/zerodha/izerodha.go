package zerodha

import (
	"context"

	"llm-trading-bot/internal/types"
)

// Broker defines the interface for interacting with a stock broker
type Broker interface {
	// LTP returns the last traded price for a symbol
	LTP(ctx context.Context, symbol string) (float64, error)

	// RecentCandles fetches the last n candles for a symbol
	RecentCandles(ctx context.Context, symbol string, n int) ([]types.Candle, error)

	// PlaceOrder places an order and returns the order response
	PlaceOrder(ctx context.Context, req types.OrderReq) (types.OrderResp, error)

	// Start initializes the broker (e.g., WebSocket connections for live data)
	Start(ctx context.Context, symbols []string) error

	// Stop gracefully shuts down the broker connections
	Stop(ctx context.Context)
}
