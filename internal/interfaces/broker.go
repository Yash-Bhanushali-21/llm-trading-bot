package interfaces

import (
	"context"

	"llm-trading-bot/internal/types"
)

type Broker interface {
	LTP(ctx context.Context, symbol string) (float64, error)
	RecentCandles(ctx context.Context, symbol string, n int) ([]types.Candle, error)
	PlaceOrder(ctx context.Context, req types.OrderReq) (types.OrderResp, error)
	Start(ctx context.Context, symbols []string) error
	Stop(ctx context.Context)
}
