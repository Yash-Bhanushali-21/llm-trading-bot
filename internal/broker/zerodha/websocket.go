package zerodha

import (
	"context"
	"fmt"
	"sync"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/types"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"github.com/zerodha/gokiteconnect/v4/models"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
)

// tickerManager manages WebSocket connections for live market data streaming
type tickerManager struct {
	kc          *kiteconnect.Client
	ticker      *kiteticker.Ticker
	apiKey      string
	accessToken string
	exchange    string

	// Candle cache: symbol -> circular buffer of recent candles
	candleCache   map[string]*candleBuffer
	candleCacheMu sync.RWMutex

	// Subscription management
	symbols      []string
	symbolTokens map[string]uint32
}

// candleBuffer stores recent candles in a circular buffer
type candleBuffer struct {
	candles []types.Candle
	maxSize int
}

// newTickerManager creates a new WebSocket ticker manager
func newTickerManager(apiKey, accessToken, exchange string) *tickerManager {
	return &tickerManager{
		apiKey:       apiKey,
		accessToken:  accessToken,
		exchange:     exchange,
		candleCache:  make(map[string]*candleBuffer),
		symbolTokens: make(map[string]uint32),
	}
}

// start initializes and starts the WebSocket connection
func (tm *tickerManager) start(ctx context.Context) error {
	// Create Kite Connect client
	tm.kc = kiteconnect.New(tm.apiKey)
	tm.kc.SetAccessToken(tm.accessToken)

	// Create ticker instance
	tm.ticker = kiteticker.New(tm.apiKey, tm.accessToken)

	// Set up event handlers
	tm.ticker.OnConnect(tm.onConnect)
	tm.ticker.OnError(tm.onError)
	tm.ticker.OnClose(tm.onClose)
	tm.ticker.OnReconnect(tm.onReconnect)
	tm.ticker.OnNoReconnect(tm.onNoReconnect)
	tm.ticker.OnTick(tm.onTick)
	tm.ticker.OnOrderUpdate(tm.onOrderUpdate)

	// Start the ticker in a goroutine
	go func() {
		logger.Info(ctx, "Starting Zerodha WebSocket ticker")
		tm.ticker.Serve()
	}()

	return nil
}

// stop closes the WebSocket connection
func (tm *tickerManager) stop(ctx context.Context) {
	if tm.ticker != nil {
		logger.Info(ctx, "Stopping Zerodha WebSocket ticker")
		tm.ticker.Stop()
	}
}

// subscribe subscribes to symbols for live data streaming
func (tm *tickerManager) subscribe(ctx context.Context, symbols []string) error {
	tm.symbols = symbols

	// Get instrument tokens for symbols
	// TODO: Implement instrument token lookup from Kite API
	// For now, using placeholder tokens
	tokens := make([]uint32, 0, len(symbols))
	for _, symbol := range symbols {
		// Placeholder: In production, fetch actual instrument tokens
		token := uint32(256265) // Example: RELIANCE token
		tm.symbolTokens[symbol] = token
		tokens = append(tokens, token)

		// Initialize candle buffer for this symbol
		tm.candleCacheMu.Lock()
		tm.candleCache[symbol] = &candleBuffer{
			candles: make([]types.Candle, 0),
			maxSize: 200, // Store last 200 candles
		}
		tm.candleCacheMu.Unlock()
	}

	// Subscribe to tokens
	if err := tm.ticker.Subscribe(tokens); err != nil {
		return fmt.Errorf("failed to subscribe to symbols: %w", err)
	}

	// Set mode to FULL to get OHLC data
	if err := tm.ticker.SetMode(kiteticker.ModeFull, tokens); err != nil {
		return fmt.Errorf("failed to set ticker mode: %w", err)
	}

	logger.Info(ctx, "Subscribed to symbols for live data", "symbols", symbols, "count", len(symbols))
	return nil
}

// getRecentCandles retrieves recent candles from cache
func (tm *tickerManager) getRecentCandles(symbol string, n int) ([]types.Candle, error) {
	tm.candleCacheMu.RLock()
	defer tm.candleCacheMu.RUnlock()

	buffer, exists := tm.candleCache[symbol]
	if !exists {
		return nil, fmt.Errorf("no candle data for symbol %s", symbol)
	}

	candles := buffer.candles
	if len(candles) == 0 {
		return nil, fmt.Errorf("no candles available for %s", symbol)
	}

	// Return last n candles
	if len(candles) < n {
		return candles, nil
	}
	return candles[len(candles)-n:], nil
}

// Event handlers

func (tm *tickerManager) onConnect() {
	logger.Info(context.Background(), "WebSocket connected")
}

func (tm *tickerManager) onError(err error) {
	logger.ErrorWithErr(context.Background(), "WebSocket error", err)
}

func (tm *tickerManager) onClose(code int, reason string) {
	logger.Warn(context.Background(), "WebSocket closed", "code", code, "reason", reason)
}

func (tm *tickerManager) onReconnect(attempt int, delay time.Duration) {
	logger.Info(context.Background(), "WebSocket reconnecting", "attempt", attempt, "delay", delay)
}

func (tm *tickerManager) onNoReconnect(attempt int) {
	logger.Warn(context.Background(), "WebSocket reconnection failed", "attempt", attempt)
}

func (tm *tickerManager) onTick(tick models.Tick) {
	symbol := tm.getSymbolByToken(tick.InstrumentToken)
	if symbol == "" {
		return
	}

	// Convert tick to candle format
	// TODO: Aggregate ticks into 1-minute candles
	// For now, treat each tick as a candle point
	candle := types.Candle{
		Ts:    tick.Timestamp.Time.Unix(),
		Open:  tick.OHLC.Open,
		High:  tick.OHLC.High,
		Low:   tick.OHLC.Low,
		Close: tick.LastPrice,
		Vol:   float64(tick.VolumeTraded),
	}

	// Add to candle cache
	tm.addCandle(symbol, candle)
}

func (tm *tickerManager) onOrderUpdate(order kiteconnect.Order) {
	// Order updates can be logged if needed
	logger.Debug(context.Background(), "Order update received", "order_id", order.OrderID, "status", order.Status)
}

// Helper methods

func (tm *tickerManager) getSymbolByToken(token uint32) string {
	for symbol, t := range tm.symbolTokens {
		if t == token {
			return symbol
		}
	}
	return ""
}

func (tm *tickerManager) addCandle(symbol string, candle types.Candle) {
	tm.candleCacheMu.Lock()
	defer tm.candleCacheMu.Unlock()

	buffer, exists := tm.candleCache[symbol]
	if !exists {
		return
	}

	// Add candle to buffer
	buffer.candles = append(buffer.candles, candle)

	// Maintain circular buffer size
	if len(buffer.candles) > buffer.maxSize {
		buffer.candles = buffer.candles[1:]
	}
}
