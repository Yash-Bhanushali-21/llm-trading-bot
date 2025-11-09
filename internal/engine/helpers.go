package engine

import (
	"math"
	"time"

	"llm-trading-bot/internal/ta"
	"llm-trading-bot/internal/types"
)

// roundToTick rounds a price to the nearest tick size.
// If tick is 0 or negative, returns the original price.
func roundToTick(price, tick float64) float64 {
	if tick <= 0 {
		return price
	}
	return math.Round(price/tick) * tick
}

// midnightIST returns midnight in Indian Standard Time (IST) for the current day.
// Used for tracking day boundaries for trading sessions.
func midnightIST() time.Time {
	now := time.Now().UTC()
	ist := time.FixedZone("IST", 19800) // IST is UTC+5:30 (19800 seconds)
	znow := now.In(ist)
	return time.Date(znow.Year(), znow.Month(), znow.Day(), 0, 0, 0, 0, ist)
}

// calculateIndicators computes all configured technical indicators from candle data.
//
// Parameters:
//   - candles: Historical price data
//   - cfg: Indicator configuration (periods, windows, etc.)
//
// Returns:
//   - types.Indicators: Calculated indicator values (RSI, SMA, Bollinger Bands, ATR)
func calculateIndicators(candles []types.Candle, cfg struct {
	SMAWindows []int
	RSIPeriod  int
	BBWindow   int
	BBStdDev   float64
	ATRPeriod  int
}) types.Indicators {
	// Extract price arrays
	closes := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))

	for i, c := range candles {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
	}

	// Calculate indicators
	indicators := types.Indicators{SMA: map[int]float64{}}

	// Calculate SMA for all configured windows
	for _, window := range cfg.SMAWindows {
		indicators.SMA[window] = ta.SMA(closes, window)
	}

	// Calculate RSI
	indicators.RSI = ta.RSI(closes, cfg.RSIPeriod)

	// Calculate Bollinger Bands
	middle, upper, lower := ta.Bollinger(closes, cfg.BBWindow, cfg.BBStdDev)
	indicators.BB.Middle = middle
	indicators.BB.Upper = upper
	indicators.BB.Lower = lower

	// Calculate ATR
	indicators.ATR = ta.ATR(highs, lows, closes, cfg.ATRPeriod)

	return indicators
}

// pickQuantity determines the quantity to trade based on decision and configuration.
//
// Priority order:
//  1. Quantity from LLM decision (if > 0)
//  2. Per-symbol configuration
//  3. Default buy/sell quantity
func pickQuantity(symbol string, decision types.Decision, cfg struct {
	PerSymbol  map[string]int
	DefaultBuy int
	DefaultSell int
}) int {
	// If LLM provided specific quantity, use it
	if decision.Qty > 0 {
		return decision.Qty
	}

	// Check for symbol-specific quantity
	if qty, ok := cfg.PerSymbol[symbol]; ok {
		return qty
	}

	// Use default based on action
	if decision.Action == "SELL" {
		return cfg.DefaultSell
	}
	return cfg.DefaultBuy
}
