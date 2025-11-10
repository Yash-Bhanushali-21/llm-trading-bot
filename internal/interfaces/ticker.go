package interfaces

import (
	"context"

	"llm-trading-bot/internal/types"
)

type TickerManager interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context)
	Subscribe(ctx context.Context, symbols []string) error
	GetRecentCandles(symbol string, n int) ([]types.Candle, error)
}
