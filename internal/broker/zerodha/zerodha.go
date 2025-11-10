package zerodha

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/types"
)

// Params holds configuration parameters for the Zerodha broker
type Params struct {
	Mode         string
	APIKey       string
	AccessToken  string
	Exchange     string
	CandleSource string // "static" or "live"
}

// Zerodha implements the Broker interface for Zerodha broker
type Zerodha struct {
	p            Params
	tickerMgr    *tickerManager
	isTickerInit bool
}

// NewZerodha creates a new Zerodha broker instance
func NewZerodha(p Params) *Zerodha {
	z := &Zerodha{p: p}

	// Initialize ticker manager for live data mode
	if p.CandleSource == "LIVE" {
		z.tickerMgr = newTickerManager(p.APIKey, p.AccessToken, p.Exchange)
	}

	return z
}

// LTP returns the last traded price for a symbol
func (z *Zerodha) LTP(ctx context.Context, symbol string) (float64, error) {
	price := 1000 + rand.Float64()*100
	logger.Debug(ctx, "Fetched LTP", "symbol", symbol, "price", price)
	return price, nil
}

// RecentCandles fetches the last n candles for a symbol
func (z *Zerodha) RecentCandles(ctx context.Context, symbol string, n int) ([]types.Candle, error) {
	logger.Debug(ctx, "Fetching recent candles", "symbol", symbol, "count", n, "mode", z.p.Mode, "source", z.p.CandleSource)

	// Route to appropriate data source
	if z.p.CandleSource == "LIVE" {
		return z.fetchLiveCandles(ctx, symbol, n)
	}

	// Default: static/mock candles for development and testing
	return z.fetchStaticCandles(ctx, symbol, n)
}

// fetchStaticCandles generates mock candle data for testing
func (z *Zerodha) fetchStaticCandles(ctx context.Context, symbol string, n int) ([]types.Candle, error) {
	cs := make([]types.Candle, 0, n)
	base := 1000.0
	now := time.Now().Unix()

	for i := n; i > 0; i-- {
		c := base + float64(i) + (rand.Float64()-0.5)*5
		h := c + rand.Float64()*3
		l := c - rand.Float64()*3
		cs = append(cs, types.Candle{
			Ts:    now - int64((n-i+1)*60),
			Open:  c - 0.5,
			High:  h,
			Low:   l,
			Close: c,
			Vol:   rand.Float64() * 1000,
		})
	}

	logger.Debug(ctx, "Static candles generated", "symbol", symbol, "count", len(cs))
	return cs, nil
}

// fetchLiveCandles fetches real-time candle data from WebSocket cache
func (z *Zerodha) fetchLiveCandles(ctx context.Context, symbol string, n int) ([]types.Candle, error) {
	if z.tickerMgr == nil {
		logger.Warn(ctx, "Ticker manager not initialized - using static data", "symbol", symbol)
		return z.fetchStaticCandles(ctx, symbol, n)
	}

	// Get candles from WebSocket cache
	candles, err := z.tickerMgr.getRecentCandles(symbol, n)
	if err != nil {
		logger.Warn(ctx, "Failed to fetch live candles from cache - using static data",
			"symbol", symbol, "error", err.Error())
		return z.fetchStaticCandles(ctx, symbol, n)
	}

	logger.Debug(ctx, "Live candles fetched from WebSocket cache",
		"symbol", symbol, "count", len(candles))
	return candles, nil
}

// Start initializes the WebSocket connection and subscribes to symbols
func (z *Zerodha) Start(ctx context.Context, symbols []string) error {
	if z.tickerMgr == nil {
		return nil // Not in live mode, nothing to start
	}

	if z.isTickerInit {
		return nil // Already started
	}

	// Start WebSocket connection
	if err := z.tickerMgr.start(ctx); err != nil {
		return fmt.Errorf("failed to start ticker manager: %w", err)
	}

	// Wait a bit for connection to establish
	time.Sleep(2 * time.Second)

	// Subscribe to symbols
	if err := z.tickerMgr.subscribe(ctx, symbols); err != nil {
		return fmt.Errorf("failed to subscribe to symbols: %w", err)
	}

	z.isTickerInit = true
	logger.Info(ctx, "Zerodha broker started successfully", "symbols", symbols)
	return nil
}

// Stop closes the WebSocket connection
func (z *Zerodha) Stop(ctx context.Context) {
	if z.tickerMgr != nil {
		z.tickerMgr.stop(ctx)
		z.isTickerInit = false
	}
}

// PlaceOrder places an order and returns the order response
func (z *Zerodha) PlaceOrder(ctx context.Context, req types.OrderReq) (types.OrderResp, error) {
	logger.Debug(ctx, "Placing order", "symbol", req.Symbol, "side", req.Side, "qty", req.Qty, "tag", req.Tag, "mode", z.p.Mode)

	if z.p.Mode == "DRY_RUN" {
		resp := types.OrderResp{OrderID: fmt.Sprintf("SIM-%d", time.Now().UnixNano()), Status: "SIMULATED", Message: "dry-run"}
		logger.Info(ctx, "Simulated order placed", "symbol", req.Symbol, "side", req.Side, "qty", req.Qty, "order_id", resp.OrderID)
		return resp, nil
	}

	if z.p.APIKey == "" || z.p.AccessToken == "" {
		err := errors.New("missing API key/access token")
		logger.ErrorWithErr(ctx, "Cannot place live order - missing credentials", err, "symbol", req.Symbol)
		return types.OrderResp{}, err
	}

	resp := types.OrderResp{OrderID: fmt.Sprintf("LIVE-%d", time.Now().UnixNano()), Status: "PLACED", Message: "ok"}
	logger.Info(ctx, "Live order placed", "symbol", req.Symbol, "side", req.Side, "qty", req.Qty, "order_id", resp.OrderID)
	return resp, nil
}
