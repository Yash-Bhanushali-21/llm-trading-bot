package engine

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"example.com/zerodha-llm-bot/internal/store"
	"example.com/zerodha-llm-bot/internal/ta"
	"example.com/zerodha-llm-bot/internal/tradelog"
	"example.com/zerodha-llm-bot/internal/types"
)

type Broker interface {
	LTP(symbol string) (float64, error)
	RecentCandles(symbol string, n int) ([]types.Candle, error)
	PlaceOrder(req types.OrderReq) (types.OrderResp, error)
}
type position struct {
	qty     int
	avg     float64
	stop    float64
	lastATR float64
}
type Engine struct {
	cfg      *store.Config
	brk      Broker
	llm      types.Decider
	pnl      float64
	dayStart time.Time
	pos      map[string]*position
}

func New(cfg *store.Config, brk Broker, d types.Decider) *Engine {
	return &Engine{cfg: cfg, brk: brk, llm: d, dayStart: midnightIST(), pos: map[string]*position{}}
}
func (e *Engine) Step(ctx context.Context, symbol string) (*types.StepResult, error) {
	candles, err := e.brk.RecentCandles(symbol, 250)
	if err != nil {
		return nil, err
	}
	if len(candles) < 50 {
		return nil, errors.New("not enough candles")
	}
	inds := e.calcIndicators(candles)
	latest := candles[len(candles)-1]
	price := latest.Close
	if p := e.pos[symbol]; p != nil && p.qty > 0 && price <= p.stop {
		resp, err := e.brk.PlaceOrder(types.OrderReq{Symbol: symbol, Side: "SELL", Qty: p.qty, Tag: "SL"})
		if err == nil {
			_ = tradelog.Append(tradelog.Entry{Symbol: symbol, Side: "SELL", Qty: p.qty, Price: price, OrderID: resp.OrderID, Reason: "STOP_LOSS", Confidence: 1.0})
			delete(e.pos, symbol)
			return &types.StepResult{Symbol: symbol, Price: price, Time: latest.Ts, Orders: []types.OrderResp{resp}, Reason: "STOP_LOSS_TRIGGERED"}, nil
		}
	}
	decision, err := e.llm.Decide(symbol, latest, inds, map[string]any{"price": price, "risk": e.cfg.Risk})
	if err != nil {
		return nil, err
	}
	_ = tradelog.AppendDecision(tradelog.DecisionEntry{Symbol: symbol, Action: decision.Action, Confidence: decision.Confidence, Reason: decision.Reason, Price: price, Indicators: map[string]float64{"RSI": inds.RSI, "SMA20": inds.SMA[20], "SMA50": inds.SMA[50], "SMA200": inds.SMA[200], "BB_MID": inds.BB.Middle, "BB_UP": inds.BB.Upper, "BB_LOW": inds.BB.Lower, "ATR": inds.ATR}})
	qty := e.pickQty(symbol, decision)
	orders := []types.OrderResp{}
	reason := decision.Reason
	if decision.Action == "BUY" && qty > 0 {
		if e.exceedsRisk(e.cfg.Risk.PerTradeRiskPct, price, qty) {
			reason += " | blocked: risk cap"
		} else {
			resp, err := e.brk.PlaceOrder(types.OrderReq{Symbol: symbol, Side: "BUY", Qty: qty, Tag: "LLM"})
			if err == nil {
				orders = append(orders, resp)
				_ = tradelog.Append(tradelog.Entry{Symbol: symbol, Side: "BUY", Qty: qty, Price: price, OrderID: resp.OrderID, Reason: decision.Reason, Confidence: decision.Confidence})
				p := e.pos[symbol]
				if p == nil {
					p = &position{}
					e.pos[symbol] = p
				}
				total := p.avg*float64(p.qty) + price*float64(qty)
				p.qty += qty
				p.avg = total / float64(p.qty)
				p.lastATR = inds.ATR
				st := e.computeStop(p.avg, p.lastATR)
				if p.stop == 0 || st > p.stop {
					p.stop = st
				}
			} else {
				reason += " | order_err:" + err.Error()
			}
		}
	} else if decision.Action == "SELL" && qty > 0 {
		resp, err := e.brk.PlaceOrder(types.OrderReq{Symbol: symbol, Side: "SELL", Qty: qty, Tag: "LLM"})
		if err == nil {
			orders = append(orders, resp)
			_ = tradelog.Append(tradelog.Entry{Symbol: symbol, Side: "SELL", Qty: qty, Price: price, OrderID: resp.OrderID, Reason: decision.Reason, Confidence: decision.Confidence})
			if p := e.pos[symbol]; p != nil {
				p.qty -= qty
				if p.qty <= 0 {
					delete(e.pos, symbol)
				}
			}
		} else {
			reason += " | order_err:" + err.Error()
		}
	}
	if e.cfg.Stop.Trailing {
		if p := e.pos[symbol]; p != nil && p.qty > 0 {
			p.lastATR = inds.ATR
			candStop := e.computeStop(price, p.lastATR)
			if candStop > p.stop {
				p.stop = candStop
			}
		}
	}
	return &types.StepResult{Symbol: symbol, Decision: decision, Price: price, Time: latest.Ts, Orders: orders, Reason: reason}, nil
}
func (e *Engine) pickQty(symbol string, d types.Decision) int {
	if d.Qty > 0 {
		return d.Qty
	}
	if v, ok := e.cfg.Qty.PerSymbol[symbol]; ok {
		return v
	}
	if d.Action == "SELL" {
		return e.cfg.Qty.DefaultSell
	}
	return e.cfg.Qty.DefaultBuy
}
func (e *Engine) exceedsRisk(perTradePct float64, price float64, qty int) bool {
	if perTradePct <= 0 {
		return false
	}
	acct := 100.0
	exp := price * float64(qty)
	return (exp / acct * 100.0) > perTradePct
}
func (e *Engine) calcIndicators(cs []types.Candle) types.Indicators {
	cl := make([]float64, len(cs))
	h := make([]float64, len(cs))
	l := make([]float64, len(cs))
	for i, c := range cs {
		cl[i] = c.Close
		h[i] = c.High
		l[i] = c.Low
	}
	inds := types.Indicators{SMA: map[int]float64{}}
	for _, w := range e.cfg.Indicators.SMAWindows {
		inds.SMA[w] = ta.SMA(cl, w)
	}
	inds.RSI = ta.RSI(cl, e.cfg.Indicators.RSIPeriod)
	m, u, lo := ta.Bollinger(cl, e.cfg.Indicators.BBWindow, e.cfg.Indicators.BBStdDev)
	inds.BB.Middle, inds.BB.Upper, inds.BB.Lower = m, u, lo
	inds.ATR = ta.ATR(h, l, cl, e.cfg.Indicators.ATRPeriod)
	return inds
}
func (e *Engine) computeStop(entry, atr float64) float64 {
	mode := strings.ToUpper(e.cfg.Stop.Mode)
	var stop float64
	if mode == "PCT" {
		stop = entry * (1.0 - e.cfg.Stop.Pct/100.0)
	} else {
		stop = entry - e.cfg.Stop.ATRMult*atr
	}
	return roundToTick(stop, e.cfg.Stop.MinTick)
}
func roundToTick(x, tick float64) float64 {
	if tick <= 0 {
		return x
	}
	return math.Round(x/tick) * tick
}
func midnightIST() time.Time {
	now := time.Now().UTC()
	ist := time.FixedZone("IST", 19800)
	znow := now.In(ist)
	return time.Date(znow.Year(), znow.Month(), znow.Day(), 0, 0, 0, 0, ist)
}
