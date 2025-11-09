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
}
