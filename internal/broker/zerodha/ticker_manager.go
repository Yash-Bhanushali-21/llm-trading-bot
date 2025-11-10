package zerodha

import (
	"context"
	"fmt"
	"sync"
	"time"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/types"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
)

const (
	maxCandlesPerSymbol = 200

	connectionWaitTime = 2 * time.Second
)

type tickerManager struct {
	kc          *kiteconnect.Client
	ticker      *kiteticker.Ticker
	apiKey      string
	accessToken string
	exchange    string

	candles map[string][]types.Candle
	mu      sync.RWMutex

	tokenToSymbol map[uint32]string
}

var _ interfaces.TickerManager = (*tickerManager)(nil)

func (tm *tickerManager) Start(ctx context.Context) error {
	tm.kc = kiteconnect.New(tm.apiKey)
	tm.kc.SetAccessToken(tm.accessToken)

	tm.ticker = kiteticker.New(tm.apiKey, tm.accessToken)

	tm.setupEventHandlers()

	go func() {
		tm.ticker.Serve()
	}()

	return nil
}

func (tm *tickerManager) Stop(ctx context.Context) {
	if tm.ticker != nil {
		tm.ticker.Stop()
	}
}

func (tm *tickerManager) Subscribe(ctx context.Context, symbols []string) error {
	tokens := make([]uint32, 0, len(symbols))

	for _, symbol := range symbols {
		token := tm.getPlaceholderToken(symbol)

		tm.tokenToSymbol[token] = symbol

		tm.mu.Lock()
		tm.candles[symbol] = make([]types.Candle, 0, maxCandlesPerSymbol)
		tm.mu.Unlock()

		tokens = append(tokens, token)
	}

	if err := tm.ticker.Subscribe(tokens); err != nil {
		return fmt.Errorf("failed to subscribe to symbols: %w", err)
	}

	if err := tm.ticker.SetMode(kiteticker.ModeFull, tokens); err != nil {
		return fmt.Errorf("failed to set ticker mode: %w", err)
	}

	return nil
}

func (tm *tickerManager) GetRecentCandles(symbol string, n int) ([]types.Candle, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	symbolCandles, exists := tm.candles[symbol]
	if !exists {
		return nil, fmt.Errorf("no candle data for symbol %s", symbol)
	}

	if len(symbolCandles) == 0 {
		return nil, fmt.Errorf("no candles available for %s", symbol)
	}

	if len(symbolCandles) < n {
		return symbolCandles, nil
	}

	return symbolCandles[len(symbolCandles)-n:], nil
}

func (tm *tickerManager) addCandle(symbol string, candle types.Candle) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	symbolCandles := tm.candles[symbol]
	symbolCandles = append(symbolCandles, candle)

	if len(symbolCandles) > maxCandlesPerSymbol {
		symbolCandles = symbolCandles[1:]
	}

	tm.candles[symbol] = symbolCandles
}

func (tm *tickerManager) getPlaceholderToken(symbol string) uint32 {
	placeholderTokens := map[string]uint32{
		"RELIANCE":   256265,
		"TCS":        2953217,
		"HDFCBANK":   341249,
		"INFY":       408065,
		"HCLTECH":    1850625,
		"LT":         2939649,
		"SBIN":       779521,
		"ICICIBANK":  1270529,
		"AXISBANK":   1510401,
		"KOTAKBANK":  492033,
		"ITC":        424961,
		"TATAMOTORS": 884737,
		"TITAN":      897537,
		"JSWSTEEL":   3001089,
		"ULTRACEMCO": 2952193,
		"BAJFINANCE": 81153,
		"HDFCLIFE":   119553,
		"BHARTIARTL": 2714625,
		"ASIANPAINT": 60417,
		"MARUTI":     2815745,
	}

	if token, exists := placeholderTokens[symbol]; exists {
		return token
	}

	return 256265
}
