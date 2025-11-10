package zerodha

import (
	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/types"
)

func newTickerManager(apiKey, accessToken, exchange string) interfaces.TickerManager {
	return &tickerManager{
		apiKey:        apiKey,
		accessToken:   accessToken,
		exchange:      exchange,
		candles:       make(map[string][]types.Candle),
		tokenToSymbol: make(map[uint32]string),
	}
}
