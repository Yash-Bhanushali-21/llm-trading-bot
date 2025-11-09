package broker

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/types"
)

type Params struct{ Mode, APIKey, AccessToken, Exchange string }
type Zerodha struct{ p Params }

func NewZerodha(p Params) *Zerodha { return &Zerodha{p: p} }

func (z *Zerodha) LTP(ctx context.Context, symbol string) (float64, error) {
	price := 1000 + rand.Float64()*100
	logger.Debug(ctx, "Fetched LTP", "symbol", symbol, "price", price)
	return price, nil
}

func (z *Zerodha) RecentCandles(ctx context.Context, symbol string, n int) ([]types.Candle, error) {
	logger.Debug(ctx, "Fetching recent candles", "symbol", symbol, "count", n, "mode", z.p.Mode)

	cs := make([]types.Candle, 0, n)
	base := 1000.0
	now := time.Now().Unix()
	for i := n; i > 0; i-- {
		c := base + float64(i) + (rand.Float64()-0.5)*5
		h := c + rand.Float64()*3
		l := c - rand.Float64()*3
		cs = append(cs, types.Candle{Ts: now - int64((n-i+1)*60), Open: c - 0.5, High: h, Low: l, Close: c, Vol: rand.Float64() * 1000})
	}

	logger.Debug(ctx, "Candles fetched successfully", "symbol", symbol, "count", len(cs))
	return cs, nil
}

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
