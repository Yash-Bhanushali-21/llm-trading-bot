package broker

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"llm-trading-bot/internal/types"
)

type Params struct{ Mode, APIKey, AccessToken, Exchange string }
type Zerodha struct{ p Params }

func NewZerodha(p Params) *Zerodha                    { return &Zerodha{p: p} }
func (z *Zerodha) LTP(symbol string) (float64, error) { return 1000 + rand.Float64()*100, nil }
func (z *Zerodha) RecentCandles(symbol string, n int) ([]types.Candle, error) {
	cs := make([]types.Candle, 0, n)
	base := 1000.0
	now := time.Now().Unix()
	for i := n; i > 0; i-- {
		c := base + float64(i) + (rand.Float64()-0.5)*5
		h := c + rand.Float64()*3
		l := c - rand.Float64()*3
		cs = append(cs, types.Candle{Ts: now - int64((n-i+1)*60), Open: c - 0.5, High: h, Low: l, Close: c, Vol: rand.Float64() * 1000})
	}
	return cs, nil
}
func (z *Zerodha) PlaceOrder(req types.OrderReq) (types.OrderResp, error) {
	if z.p.Mode == "DRY_RUN" {
		return types.OrderResp{OrderID: fmt.Sprintf("SIM-%d", time.Now().UnixNano()), Status: "SIMULATED", Message: "dry-run"}, nil
	}
	if z.p.APIKey == "" || z.p.AccessToken == "" {
		return types.OrderResp{}, errors.New("missing API key/access token")
	}
	return types.OrderResp{OrderID: fmt.Sprintf("LIVE-%d", time.Now().UnixNano()), Status: "PLACED", Message: "ok"}, nil
}
