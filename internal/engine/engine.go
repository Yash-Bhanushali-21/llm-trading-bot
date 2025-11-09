package engine

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"llm-trading-bot/internal/broker/zerodha"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/ta"
	"llm-trading-bot/internal/tradelog"
	"llm-trading-bot/internal/types"
)

type position struct {
	qty     int
	avg     float64
	stop    float64
	lastATR float64
}
type Engine struct {
	cfg      *store.Config
	brk      zerodha.Broker
	llm      types.Decider
	pnl      float64
	dayStart time.Time
	pos      map[string]*position
}

func New(cfg *store.Config, brk zerodha.Broker, d types.Decider) *Engine {
	return &Engine{cfg: cfg, brk: brk, llm: d, dayStart: midnightIST(), pos: map[string]*position{}}
}
func (e *Engine) Step(ctx context.Context, symbol string) (*types.StepResult, error) {
	logger.Debug(ctx, "Starting trading step", "symbol", symbol)

	// Fetch candles
	candles, err := e.brk.RecentCandles(ctx, symbol, 250)
	if err != nil {
		logger.ErrorWithErr(ctx, "Failed to fetch candles", err, "symbol", symbol)
		return nil, err
	}
	logger.Debug(ctx, "Candles fetched successfully", "symbol", symbol, "count", len(candles))

	if len(candles) < 50 {
		err := errors.New("not enough candles")
		logger.Error(ctx, "Insufficient candle data", "symbol", symbol, "received", len(candles), "required", 50)
		return nil, err
	}

	// Calculate indicators
	inds := e.calcIndicators(candles)
	logger.Debug(ctx, "Indicators calculated",
		"symbol", symbol,
		"rsi", inds.RSI,
		"sma20", inds.SMA[20],
		"sma50", inds.SMA[50],
		"sma200", inds.SMA[200],
		"bb_upper", inds.BB.Upper,
		"bb_middle", inds.BB.Middle,
		"bb_lower", inds.BB.Lower,
		"atr", inds.ATR,
	)

	latest := candles[len(candles)-1]
	price := latest.Close
	logger.Debug(ctx, "Current market state", "symbol", symbol, "price", price, "timestamp", latest.Ts)

	// Check stop-loss trigger
	if p := e.pos[symbol]; p != nil && p.qty > 0 {
		if price <= p.stop {
			logger.Warn(ctx, "Stop loss triggered",
				"symbol", symbol,
				"event", "STOP_LOSS_TRIGGERED",
				"current_price", price,
				"stop_price", p.stop,
				"position_qty", p.qty,
				"position_avg", p.avg,
				"unrealized_loss", (price-p.avg)*float64(p.qty),
			)

			resp, err := e.brk.PlaceOrder(ctx, types.OrderReq{Symbol: symbol, Side: "SELL", Qty: p.qty, Tag: "SL"})
			if err == nil {
				logger.Info(ctx, "Trade executed", "symbol", symbol, "side", "SELL", "qty", p.qty, "price", price, "order_id", resp.OrderID, "tag", "SL", "reason", "STOP_LOSS")
				_ = tradelog.Append(tradelog.Entry{Symbol: symbol, Side: "SELL", Qty: p.qty, Price: price, OrderID: resp.OrderID, Reason: "STOP_LOSS", Confidence: 1.0})
				delete(e.pos, symbol)
				return &types.StepResult{Symbol: symbol, Price: price, Time: latest.Ts, Orders: []types.OrderResp{resp}, Reason: "STOP_LOSS_TRIGGERED"}, nil
			} else {
				logger.ErrorWithErr(ctx, "Failed to execute stop-loss order", err, "symbol", symbol, "qty", p.qty, "price", price)
			}
		}
	}

	// Get LLM decision
	decision, err := e.llm.Decide(ctx, symbol, latest, inds, map[string]any{"price": price, "risk": e.cfg.Risk})
	if err != nil {
		logger.ErrorWithErr(ctx, "LLM decision failed", err, "symbol", symbol)
		return nil, err
	}

	// Log the decision
	logger.Info(ctx, "Trading decision", "symbol", symbol, "action", decision.Action, "confidence", decision.Confidence, "reason", decision.Reason)
	_ = tradelog.AppendDecision(tradelog.DecisionEntry{Symbol: symbol, Action: decision.Action, Confidence: decision.Confidence, Reason: decision.Reason, Price: price, Indicators: map[string]float64{"RSI": inds.RSI, "SMA20": inds.SMA[20], "SMA50": inds.SMA[50], "SMA200": inds.SMA[200], "BB_MID": inds.BB.Middle, "BB_UP": inds.BB.Upper, "BB_LOW": inds.BB.Lower, "ATR": inds.ATR}})

	qty := e.pickQty(symbol, decision)
	logger.Debug(ctx, "Position sizing determined", "symbol", symbol, "action", decision.Action, "qty", qty)

	orders := []types.OrderResp{}
	reason := decision.Reason

	// Execute BUY decision
	if decision.Action == "BUY" && qty > 0 {
		logger.Debug(ctx, "Processing BUY decision", "symbol", symbol, "qty", qty, "price", price)

		// Risk check
		riskExceeded := e.exceedsRisk(e.cfg.Risk.PerTradeRiskPct, price, qty)
		if riskExceeded {
			exposure := price * float64(qty)
			logger.Warn(ctx, "Trade blocked by risk cap",
				"symbol", symbol,
				"event", "TRADE_BLOCKED_RISK_CAP",
				"qty", qty,
				"price", price,
				"exposure", exposure,
				"risk_limit_pct", e.cfg.Risk.PerTradeRiskPct,
			)
			reason += " | blocked: risk cap"
		} else {
			resp, err := e.brk.PlaceOrder(ctx, types.OrderReq{Symbol: symbol, Side: "BUY", Qty: qty, Tag: "LLM"})
			if err == nil {
				orders = append(orders, resp)
				logger.Info(ctx, "Trade executed", "symbol", symbol, "side", "BUY", "qty", qty, "price", price, "order_id", resp.OrderID, "tag", "LLM", "confidence", decision.Confidence)
				_ = tradelog.Append(tradelog.Entry{Symbol: symbol, Side: "BUY", Qty: qty, Price: price, OrderID: resp.OrderID, Reason: decision.Reason, Confidence: decision.Confidence})

				// Update position
				p := e.pos[symbol]
				if p == nil {
					p = &position{}
					e.pos[symbol] = p
					logger.Debug(ctx, "New position created", "symbol", symbol)
				}
				oldQty := p.qty
				oldAvg := p.avg
				total := p.avg*float64(p.qty) + price*float64(qty)
				p.qty += qty
				p.avg = total / float64(p.qty)
				p.lastATR = inds.ATR
				st := e.computeStop(p.avg, p.lastATR)
				if p.stop == 0 || st > p.stop {
					p.stop = st
				}
				logger.Info(ctx, "Position updated after BUY",
					"symbol", symbol,
					"old_qty", oldQty,
					"old_avg", oldAvg,
					"new_qty", p.qty,
					"new_avg", p.avg,
					"stop_price", p.stop,
					"atr", p.lastATR,
				)
			} else {
				logger.ErrorWithErr(ctx, "Failed to place BUY order", err, "symbol", symbol, "qty", qty, "price", price)
				reason += " | order_err:" + err.Error()
			}
		}
	} else if decision.Action == "SELL" && qty > 0 {
		// Execute SELL decision
		logger.Debug(ctx, "Processing SELL decision", "symbol", symbol, "qty", qty, "price", price)
		resp, err := e.brk.PlaceOrder(ctx, types.OrderReq{Symbol: symbol, Side: "SELL", Qty: qty, Tag: "LLM"})
		if err == nil {
			orders = append(orders, resp)
			logger.Info(ctx, "Trade executed", "symbol", symbol, "side", "SELL", "qty", qty, "price", price, "order_id", resp.OrderID, "tag", "LLM", "confidence", decision.Confidence)
			_ = tradelog.Append(tradelog.Entry{Symbol: symbol, Side: "SELL", Qty: qty, Price: price, OrderID: resp.OrderID, Reason: decision.Reason, Confidence: decision.Confidence})

			// Update position
			if p := e.pos[symbol]; p != nil {
				oldQty := p.qty
				p.qty -= qty
				realizedPnL := (price - p.avg) * float64(qty)
				logger.Info(ctx, "Position updated after SELL",
					"symbol", symbol,
					"old_qty", oldQty,
					"new_qty", p.qty,
					"avg_price", p.avg,
					"sell_price", price,
					"realized_pnl", realizedPnL,
				)
				if p.qty <= 0 {
					logger.Info(ctx, "Position closed", "symbol", symbol, "realized_pnl", realizedPnL)
					delete(e.pos, symbol)
				}
			}
		} else {
			logger.ErrorWithErr(ctx, "Failed to place SELL order", err, "symbol", symbol, "qty", qty, "price", price)
			reason += " | order_err:" + err.Error()
		}
	} else if decision.Action == "HOLD" {
		logger.Debug(ctx, "HOLD decision - no action taken", "symbol", symbol, "reason", decision.Reason)
	}

	// Update trailing stop if enabled
	if e.cfg.Stop.Trailing {
		if p := e.pos[symbol]; p != nil && p.qty > 0 {
			oldStop := p.stop
			p.lastATR = inds.ATR
			candStop := e.computeStop(price, p.lastATR)
			if candStop > p.stop {
				p.stop = candStop
				logger.Debug(ctx, "Trailing stop updated",
					"symbol", symbol,
					"old_stop", oldStop,
					"new_stop", p.stop,
					"current_price", price,
					"atr", p.lastATR,
				)
			}
		}
	}

	logger.Debug(ctx, "Trading step completed", "symbol", symbol, "action", decision.Action, "orders", len(orders))
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
