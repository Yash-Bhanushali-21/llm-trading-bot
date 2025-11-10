package zerodha

import (
	"context"
	"fmt"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/types"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
)

const (
	// defaultCandleBufferSize is the default number of candles to store per symbol
	defaultCandleBufferSize = 200

	// connectionWaitTime is how long to wait for WebSocket connection to establish
	connectionWaitTime = 2 * time.Second
)

// tickerManager manages WebSocket connections for live market data streaming
type tickerManager struct {
	kc          *kiteconnect.Client
	ticker      *kiteticker.Ticker
	apiKey      string
	accessToken string
	exchange    string

	// Modular components
	cache  *candleCache
	mapper *instrumentMapper
}

// Start initializes and starts the WebSocket connection
func (tm *tickerManager) Start(ctx context.Context) error {
	// Create Kite Connect client
	tm.kc = kiteconnect.New(tm.apiKey)
	tm.kc.SetAccessToken(tm.accessToken)

	// Create ticker instance
	tm.ticker = kiteticker.New(tm.apiKey, tm.accessToken)

	// Setup event handlers
	tm.setupEventHandlers()

	// Start the ticker in a goroutine
	go func() {
		logger.Info(ctx, "Starting Zerodha WebSocket ticker")
		tm.ticker.Serve()
	}()

	return nil
}

// Stop closes the WebSocket connection gracefully
func (tm *tickerManager) Stop(ctx context.Context) {
	if tm.ticker != nil {
		logger.Info(ctx, "Stopping Zerodha WebSocket ticker")
		tm.ticker.Stop()
	}
}

// Subscribe subscribes to symbols for live data streaming
func (tm *tickerManager) Subscribe(ctx context.Context, symbols []string) error {
	// Get instrument tokens for symbols
	tokens, err := tm.resolveInstrumentTokens(ctx, symbols)
	if err != nil {
		return fmt.Errorf("failed to resolve instrument tokens: %w", err)
	}

	// Subscribe to tokens
	if err := tm.ticker.Subscribe(tokens); err != nil {
		return fmt.Errorf("failed to subscribe to symbols: %w", err)
	}

	// Set mode to FULL to get OHLC data
	if err := tm.ticker.SetMode(kiteticker.ModeFull, tokens); err != nil {
		return fmt.Errorf("failed to set ticker mode: %w", err)
	}

	logger.Info(ctx, "Subscribed to symbols for live data",
		"symbols", symbols,
		"count", len(symbols),
	)

	return nil
}

// GetRecentCandles retrieves recent candles from cache
func (tm *tickerManager) GetRecentCandles(symbol string, n int) ([]types.Candle, error) {
	return tm.cache.getRecent(symbol, n)
}

// resolveInstrumentTokens resolves symbols to instrument tokens
func (tm *tickerManager) resolveInstrumentTokens(ctx context.Context, symbols []string) ([]uint32, error) {
	// TODO: Implement actual instrument token lookup from Kite API
	// For now, using placeholder tokens for development
	tokens := make([]uint32, 0, len(symbols))

	for _, symbol := range symbols {
		// Placeholder token mapping (replace with actual API call)
		token := tm.getPlaceholderToken(symbol)

		// Add mapping for bidirectional lookup
		tm.mapper.addMapping(symbol, token)

		// Initialize candle buffer for this symbol
		tm.cache.initBuffer(symbol, defaultCandleBufferSize)

		tokens = append(tokens, token)
	}

	logger.Debug(ctx, "Resolved instrument tokens",
		"symbols", symbols,
		"tokens", tokens,
	)

	return tokens, nil
}

// getPlaceholderToken returns a placeholder token for a symbol
// TODO: Replace with actual Kite API instrument token lookup
func (tm *tickerManager) getPlaceholderToken(symbol string) uint32 {
	// Common placeholder tokens for testing
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

	// Default fallback token
	return 256265
}
