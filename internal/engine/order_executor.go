package engine

import (
	"context"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/tradelog"
	"llm-trading-bot/internal/types"
)

type orderExecutor struct {
	broker interfaces.Broker
}

func newOrderExecutor(broker interfaces.Broker) *orderExecutor {
	return &orderExecutor{
		broker: broker,
	}
}

//
//
func (oe *orderExecutor) placeBuyOrder(ctx context.Context, symbol string, qty int, price float64, reason string, confidence float64) (types.OrderResp, error) {
	req := types.OrderReq{
		Symbol: symbol,
		Side:   "BUY",
		Qty:    qty,
		Tag:    "LLM",
	}

	resp, err := oe.broker.PlaceOrder(ctx, req)
	if err != nil {
		logger.ErrorWithErr(ctx, "Failed to place BUY order", err,
			"symbol", symbol,
			"qty", qty,
			"price", price,
		)
		return types.OrderResp{}, err
	}


	_ = tradelog.Append(tradelog.Entry{
		Symbol:     symbol,
		Side:       "BUY",
		Qty:        qty,
		Price:      price,
		OrderID:    resp.OrderID,
		Reason:     reason,
		Confidence: confidence,
	})

	return resp, nil
}

//
//
func (oe *orderExecutor) placeSellOrder(ctx context.Context, symbol string, qty int, price float64, reason string, confidence float64, tag string) (types.OrderResp, error) {
	req := types.OrderReq{
		Symbol: symbol,
		Side:   "SELL",
		Qty:    qty,
		Tag:    tag,
	}

	resp, err := oe.broker.PlaceOrder(ctx, req)
	if err != nil {
		logger.ErrorWithErr(ctx, "Failed to place SELL order", err,
			"symbol", symbol,
			"qty", qty,
			"price", price,
		)
		return types.OrderResp{}, err
	}


	_ = tradelog.Append(tradelog.Entry{
		Symbol:     symbol,
		Side:       "SELL",
		Qty:        qty,
		Price:      price,
		OrderID:    resp.OrderID,
		Reason:     reason,
		Confidence: confidence,
	})

	return resp, nil
}

func (oe *orderExecutor) logDecision(ctx context.Context, symbol string, decision types.Decision, price float64, indicators types.Indicators) {

	_ = tradelog.AppendDecision(tradelog.DecisionEntry{
		Symbol:     symbol,
		Action:     decision.Action,
		Confidence: decision.Confidence,
		Reason:     decision.Reason,
		Price:      price,
		Indicators: map[string]float64{
			"RSI":    indicators.RSI,
			"SMA20":  indicators.SMA[20],
			"SMA50":  indicators.SMA[50],
			"SMA200": indicators.SMA[200],
			"BB_MID": indicators.BB.Middle,
			"BB_UP":  indicators.BB.Upper,
			"BB_LOW": indicators.BB.Lower,
			"ATR":    indicators.ATR,
		},
	})
}
