package brokerobs

import (
	"context"
	"fmt"

	"llm-trading-bot/internal/broker/zerodha"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/types"
)

// observableBroker wraps a Broker with observability (logging & tracing)
type observableBroker struct {
	broker zerodha.Broker
}

// Compile-time interface check
var _ zerodha.Broker = (*observableBroker)(nil)

// Wrap wraps a broker with observability middleware
func Wrap(broker zerodha.Broker) zerodha.Broker {
	return &observableBroker{
		broker: broker,
	}
}

// LTP returns the last traded price with observability
func (ob *observableBroker) LTP(ctx context.Context, symbol string) (float64, error) {
	ctx, span := trace.StartSpan(ctx, "broker.LTP")
	defer span.End()

	logger.DebugSkip(ctx, 1, "Fetching LTP", "symbol", symbol)

	price, err := ob.broker.LTP(ctx, symbol)
	if err != nil {
		logger.ErrorWithErrSkip(ctx, 1, "Failed to fetch LTP", err, "symbol", symbol)
		return 0, err
	}

	logger.DebugSkip(ctx, 1, "LTP fetched successfully", "symbol", symbol, "price", price)
	return price, nil
}

// RecentCandles fetches candles with observability
func (ob *observableBroker) RecentCandles(ctx context.Context, symbol string, n int) ([]types.Candle, error) {
	ctx, span := trace.StartSpan(ctx, "broker.RecentCandles")
	defer span.End()

	logger.DebugSkip(ctx, 1, "Fetching recent candles", "symbol", symbol, "count", n)

	candles, err := ob.broker.RecentCandles(ctx, symbol, n)
	if err != nil {
		logger.ErrorWithErrSkip(ctx, 1, "Failed to fetch candles", err, "symbol", symbol, "count", n)
		return nil, err
	}

	logger.DebugSkip(ctx, 1, "Candles fetched successfully", "symbol", symbol, "count", len(candles))
	return candles, nil
}

// PlaceOrder places an order with observability
func (ob *observableBroker) PlaceOrder(ctx context.Context, req types.OrderReq) (types.OrderResp, error) {
	ctx, span := trace.StartSpan(ctx, "broker.PlaceOrder")
	defer span.End()

	logger.InfoSkip(ctx, 1, "Placing order",
		"symbol", req.Symbol,
		"side", req.Side,
		"qty", req.Qty,
		"tag", req.Tag,
	)

	resp, err := ob.broker.PlaceOrder(ctx, req)
	if err != nil {
		logger.ErrorWithErrSkip(ctx, 1, "Failed to place order", err,
			"symbol", req.Symbol,
			"side", req.Side,
			"qty", req.Qty,
		)
		return types.OrderResp{}, err
	}

	logger.InfoSkip(ctx, 1, "Order placed successfully",
		"symbol", req.Symbol,
		"order_id", resp.OrderID,
		"status", resp.Status,
	)
	return resp, nil
}

// Start initializes the broker with observability
func (ob *observableBroker) Start(ctx context.Context, symbols []string) error {
	ctx, span := trace.StartSpan(ctx, "broker.Start")
	defer span.End()

	logger.InfoSkip(ctx, 1, "Starting broker", "symbols", symbols, "count", len(symbols))

	err := ob.broker.Start(ctx, symbols)
	if err != nil {
		logger.ErrorWithErrSkip(ctx, 1, "Failed to start broker", err, "symbols", symbols)
		return fmt.Errorf("broker start failed: %w", err)
	}

	logger.InfoSkip(ctx, 1, "Broker started successfully", "symbols", symbols)
	return nil
}

// Stop shuts down the broker with observability
func (ob *observableBroker) Stop(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "broker.Stop")
	defer span.End()

	logger.InfoSkip(ctx, 1, "Stopping broker")
	ob.broker.Stop(ctx)
	logger.InfoSkip(ctx, 1, "Broker stopped successfully")
}
