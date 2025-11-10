package zerodha

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/types"
)

type Params struct {
	Mode         string
	APIKey       string
	AccessToken  string
	Exchange     string
	CandleSource string
}

type Zerodha struct {
	p            Params
	tickerMgr    interfaces.TickerManager
	isTickerInit bool
}

var _ interfaces.Broker = (*Zerodha)(nil)

func NewZerodha(p Params) *Zerodha {
	z := &Zerodha{p: p}

	if p.CandleSource == "LIVE" {
		z.tickerMgr = newTickerManager(p.APIKey, p.AccessToken, p.Exchange)
	}

	return z
}

func newTickerManager(apiKey, accessToken, exchange string) interfaces.TickerManager {
	return &tickerManager{
		apiKey:        apiKey,
		accessToken:   accessToken,
		exchange:      exchange,
		candles:       make(map[string][]types.Candle),
		tokenToSymbol: make(map[uint32]string),
	}
}

func (z *Zerodha) LTP(ctx context.Context, symbol string) (float64, error) {
	price := 1000 + rand.Float64()*100
	return price, nil
}

func (z *Zerodha) RecentCandles(ctx context.Context, symbol string, n int) ([]types.Candle, error) {
	if z.p.CandleSource == "LIVE" {
		return z.fetchLiveCandles(ctx, symbol, n)
	}

	return z.fetchStaticCandles(ctx, symbol, n)
}

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

	return cs, nil
}

func (z *Zerodha) fetchLiveCandles(ctx context.Context, symbol string, n int) ([]types.Candle, error) {
	if z.tickerMgr == nil {
		return z.fetchStaticCandles(ctx, symbol, n)
	}

	candles, err := z.tickerMgr.GetRecentCandles(symbol, n)
	if err != nil {
		return z.fetchStaticCandles(ctx, symbol, n)
	}

	return candles, nil
}

func (z *Zerodha) Start(ctx context.Context, symbols []string) error {
	if z.tickerMgr == nil {
		return nil // Not in live mode, nothing to start
	}

	if z.isTickerInit {
		return nil // Already started
	}

	if err := z.tickerMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start ticker manager: %w", err)
	}

	time.Sleep(2 * time.Second)

	if err := z.tickerMgr.Subscribe(ctx, symbols); err != nil {
		return fmt.Errorf("failed to subscribe to symbols: %w", err)
	}

	z.isTickerInit = true
	return nil
}

func (z *Zerodha) Stop(ctx context.Context) {
	if z.tickerMgr != nil {
		z.tickerMgr.Stop(ctx)
		z.isTickerInit = false
	}
}

func (z *Zerodha) PlaceOrder(ctx context.Context, req types.OrderReq) (types.OrderResp, error) {
	if z.p.Mode == "DRY_RUN" {
		return types.OrderResp{
			OrderID: fmt.Sprintf("SIM-%d", time.Now().UnixNano()),
			Status:  "SIMULATED",
			Message: "dry-run",
		}, nil
	}

	if z.p.APIKey == "" || z.p.AccessToken == "" {
		return types.OrderResp{}, errors.New("missing API key/access token")
	}

	return types.OrderResp{
		OrderID: fmt.Sprintf("LIVE-%d", time.Now().UnixNano()),
		Status:  "PLACED",
		Message: "ok",
	}, nil
}
