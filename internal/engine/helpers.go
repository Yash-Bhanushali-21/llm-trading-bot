package engine

import (
	"math"
	"time"

	"llm-trading-bot/internal/ta"
	"llm-trading-bot/internal/types"
)

func roundToTick(price, tick float64) float64 {
	if tick <= 0 {
		return price
	}
	return math.Round(price/tick) * tick
}

func midnightIST() time.Time {
	now := time.Now().UTC()
	ist := time.FixedZone("IST", 19800) // IST is UTC+5:30 (19800 seconds)
	znow := now.In(ist)
	return time.Date(znow.Year(), znow.Month(), znow.Day(), 0, 0, 0, 0, ist)
}

//
//
func calculateIndicators(candles []types.Candle, cfg struct {
	SMAWindows []int
	RSIPeriod  int
	BBWindow   int
	BBStdDev   float64
	ATRPeriod  int
}) types.Indicators {
	closes := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))

	for i, c := range candles {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
	}

	indicators := types.Indicators{SMA: map[int]float64{}}

	for _, window := range cfg.SMAWindows {
		indicators.SMA[window] = ta.SMA(closes, window)
	}

	indicators.RSI = ta.RSI(closes, cfg.RSIPeriod)

	middle, upper, lower := ta.Bollinger(closes, cfg.BBWindow, cfg.BBStdDev)
	indicators.BB.Middle = middle
	indicators.BB.Upper = upper
	indicators.BB.Lower = lower

	indicators.ATR = ta.ATR(highs, lows, closes, cfg.ATRPeriod)

	return indicators
}

//
func pickQuantity(symbol string, decision types.Decision, cfg struct {
	PerSymbol  map[string]int
	DefaultBuy int
	DefaultSell int
}) int {
	if decision.Qty > 0 {
		return decision.Qty
	}

	if qty, ok := cfg.PerSymbol[symbol]; ok {
		return qty
	}

	if decision.Action == "SELL" {
		return cfg.DefaultSell
	}
	return cfg.DefaultBuy
}
