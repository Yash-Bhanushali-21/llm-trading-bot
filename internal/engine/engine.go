package engine

import (
	"context"
	"errors"
	"time"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

type Engine struct {
	cfg      *store.Config
	broker   interfaces.Broker
	llm      types.Decider
	dayStart time.Time

	positions *positionManager
	risk      *riskManager
	stop      *stopManager
	executor  *orderExecutor
}

func newEngine(cfg *store.Config, brk interfaces.Broker, d types.Decider) *Engine {
	return &Engine{
		cfg:      cfg,
		broker:   brk,
		llm:      d,
		dayStart: midnightIST(),

		positions: newPositionManager(),
		risk:      newRiskManager(),
		stop: newStopManager(
			cfg.Stop.Mode,
			cfg.Stop.Pct,
			cfg.Stop.ATRMult,
			cfg.Stop.MinTick,
			cfg.Stop.Trailing,
		),
		executor: newOrderExecutor(brk),
	}
}

// Step executes one complete trading cycle for a symbol.
func (e *Engine) Step(ctx context.Context, symbol string) (*types.StepResult, error) {

	// 1. Fetch market data
	candles, err := e.fetchCandles(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// 2. Calculate indicators
	indicators := calculateIndicators(candles, struct {
		SMAWindows []int
		RSIPeriod  int
		BBWindow   int
		BBStdDev   float64
		ATRPeriod  int
	}{
		SMAWindows: e.cfg.Indicators.SMAWindows,
		RSIPeriod:  e.cfg.Indicators.RSIPeriod,
		BBWindow:   e.cfg.Indicators.BBWindow,
		BBStdDev:   e.cfg.Indicators.BBStdDev,
		ATRPeriod:  e.cfg.Indicators.ATRPeriod,
	})

	e.logIndicators(ctx, symbol, indicators)

	latest := candles[len(candles)-1]
	price := latest.Close

	// 3. Check stop-loss trigger
	if result := e.handleStopLoss(ctx, symbol, price, latest.Ts); result != nil {
		return result, nil
	}

	// 4. Get LLM decision
	decision, err := e.llm.Decide(ctx, symbol, latest, indicators, map[string]any{
		"price": price,
		"risk":  e.cfg.Risk,
	})
	if err != nil {
		logger.ErrorWithErr(ctx, "LLM decision failed", err, "symbol", symbol)
		return nil, err
	}

	// Log decision
	e.executor.logDecision(ctx, symbol, decision, price, indicators)

	// 5. Determine quantity
	qty := pickQuantity(symbol, decision, struct {
		PerSymbol   map[string]int
		DefaultBuy  int
		DefaultSell int
	}{
		PerSymbol:   e.cfg.Qty.PerSymbol,
		DefaultBuy:  e.cfg.Qty.DefaultBuy,
		DefaultSell: e.cfg.Qty.DefaultSell,
	})


	// 6. Execute trading action
	orders, reason := e.executeDecision(ctx, symbol, decision, qty, price, indicators.ATR)

	// 7. Update trailing stop if enabled
	e.updateTrailingStop(ctx, symbol, price, indicators.ATR)


	return &types.StepResult{
		Symbol:   symbol,
		Decision: decision,
		Price:    price,
		Time:     latest.Ts,
		Orders:   orders,
		Reason:   reason,
	}, nil
}

// fetchCandles retrieves recent candle data from the broker.
func (e *Engine) fetchCandles(ctx context.Context, symbol string) ([]types.Candle, error) {
	candles, err := e.broker.RecentCandles(ctx, symbol, 250)
	if err != nil {
		logger.ErrorWithErr(ctx, "Failed to fetch candles", err, "symbol", symbol)
		return nil, err
	}


	if len(candles) < 50 {
		err := errors.New("not enough candles")
		logger.Error(ctx, "Insufficient candle data", "symbol", symbol, "received", len(candles), "required", 50)
		return nil, err
	}

	return candles, nil
}

// logIndicators logs calculated indicator values for debugging.
func (e *Engine) logIndicators(ctx context.Context, symbol string, inds types.Indicators) {
	// Indicators logged via middleware
}

// handleStopLoss checks and executes stop-loss if triggered.
func (e *Engine) handleStopLoss(ctx context.Context, symbol string, price float64, timestamp int64) *types.StepResult {
	pos := e.positions.get(symbol)
	if pos == nil || pos.qty <= 0 {
		return nil
	}

	if !e.stop.checkStopLoss(ctx, symbol, price, pos.stop, pos) {
		return nil
	}

	// Stop-loss triggered - place SELL order
	resp, err := e.executor.placeSellOrder(ctx, symbol, pos.qty, price, "STOP_LOSS", 1.0, "SL")
	if err != nil {
		logger.ErrorWithErr(ctx, "Failed to execute stop-loss order", err, "symbol", symbol, "qty", pos.qty, "price", price)
		return nil
	}

	// Close position
	e.positions.close(symbol)

	return &types.StepResult{
		Symbol: symbol,
		Price:  price,
		Time:   timestamp,
		Orders: []types.OrderResp{resp},
		Reason: "STOP_LOSS_TRIGGERED",
	}
}

// executeDecision executes the trading decision (BUY/SELL/HOLD).
func (e *Engine) executeDecision(ctx context.Context, symbol string, decision types.Decision, qty int, price, atr float64) ([]types.OrderResp, string) {
	orders := []types.OrderResp{}
	reason := decision.Reason

	switch decision.Action {
	case "BUY":
		if qty <= 0 {
			return orders, reason
		}


		// Risk check
		riskExceeded, _ := e.risk.validateTrade(ctx, symbol, price, qty, e.cfg.Risk.PerTradeRiskPct)
		if riskExceeded {
			reason += " | blocked: risk cap"
			return orders, reason
		}

		// Place order
		resp, err := e.executor.placeBuyOrder(ctx, symbol, qty, price, decision.Reason, decision.Confidence)
		if err != nil {
			reason += " | order_err:" + err.Error()
			return orders, reason
		}

		orders = append(orders, resp)

		// Calculate stop-loss
		stopPrice := e.stop.calculateStopPrice(price, atr)

		// Update position
		e.positions.addBuy(ctx, symbol, qty, price, atr, stopPrice)

	case "SELL":
		if qty <= 0 {
			return orders, reason
		}


		// Place order
		resp, err := e.executor.placeSellOrder(ctx, symbol, qty, price, decision.Reason, decision.Confidence, "LLM")
		if err != nil {
			reason += " | order_err:" + err.Error()
			return orders, reason
		}

		orders = append(orders, resp)

		// Update position
		e.positions.reduceSell(ctx, symbol, qty, price)

	case "HOLD":
	}

	return orders, reason
}

// updateTrailingStop updates the trailing stop-loss if enabled.
func (e *Engine) updateTrailingStop(ctx context.Context, symbol string, price, atr float64) {
	if !e.stop.isTrailingEnabled() {
		return
	}

	pos := e.positions.get(symbol)
	if pos == nil || pos.qty <= 0 {
		return
	}

	newStop := e.stop.calculateStopPrice(price, atr)
	e.positions.updateTrailingStop(ctx, symbol, newStop, atr)
}
