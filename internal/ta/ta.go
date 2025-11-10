package ta

import "math"

func SMA(closes []float64, n int) float64 {
	if len(closes) < n || n <= 0 {
		return math.NaN()
	}
	sum := 0.0
	for i := len(closes) - n; i < len(closes); i++ {
		sum += closes[i]
	}
	return sum / float64(n)
}
func RSI(closes []float64, period int) float64 {
	if len(closes) < period+1 || period <= 0 {
		return math.NaN()
	}
	gain, loss := 0.0, 0.0
	for i := len(closes) - period; i < len(closes); i++ {
		d := closes[i] - closes[i-1]
		if d > 0 {
			gain += d
		} else {
			loss -= d
		}
	}
	if loss == 0 {
		return 100.0
	}
	rs := (gain / float64(period)) / (loss / float64(period))
	return 100.0 - (100.0 / (1.0 + rs))
}
func StdDev(vals []float64, n int) float64 {
	if len(vals) < n || n <= 0 {
		return math.NaN()
	}
	m := SMA(vals, n)
	s := 0.0
	for i := len(vals) - n; i < len(vals); i++ {
		d := vals[i] - m
		s += d * d
	}
	return math.Sqrt(s / float64(n))
}
func Bollinger(closes []float64, n int, k float64) (mid, up, low float64) {
	mid = SMA(closes, n)
	sd := StdDev(closes, n)
	up = mid + k*sd
	low = mid - k*sd
	return
}
func ATR(highs, lows, closes []float64, period int) float64 {
	if len(highs) != len(lows) || len(lows) != len(closes) {
		return math.NaN()
	}
	n := period
	if len(closes) < n+1 {
		return math.NaN()
	}
	trs := make([]float64, 0, n)
	for i := len(closes) - n; i < len(closes); i++ {
		tr1 := highs[i] - lows[i]
		tr2 := math.Abs(highs[i] - closes[i-1])
		tr3 := math.Abs(lows[i] - closes[i-1])
		tr := math.Max(tr1, math.Max(tr2, tr3))
		trs = append(trs, tr)
	}
	sum := 0.0
	for _, v := range trs {
		sum += v
	}
	return sum / float64(n)
}

// EMA calculates the Exponential Moving Average
// EMA = Price(t) * k + EMA(y) * (1 - k)
// where k = 2 / (N + 1)
func EMA(closes []float64, period int) float64 {
	if len(closes) < period || period <= 0 {
		return math.NaN()
	}

	// Start with SMA for the first value
	k := 2.0 / float64(period+1)
	ema := SMA(closes[:period], period)

	// Calculate EMA for remaining values
	for i := period; i < len(closes); i++ {
		ema = closes[i]*k + ema*(1-k)
	}

	return ema
}

// MACD calculates the Moving Average Convergence Divergence
// Returns: (MACD line, Signal line, Histogram)
// MACD Line = 12-period EMA - 26-period EMA
// Signal Line = 9-period EMA of MACD Line
// Histogram = MACD Line - Signal Line
func MACD(closes []float64, fastPeriod, slowPeriod, signalPeriod int) (macd, signal, histogram float64) {
	if len(closes) < slowPeriod {
		return math.NaN(), math.NaN(), math.NaN()
	}

	// Calculate MACD line (fast EMA - slow EMA)
	fastEMA := EMA(closes, fastPeriod)
	slowEMA := EMA(closes, slowPeriod)
	macd = fastEMA - slowEMA

	// For signal line, we need MACD values over time
	// Simplified: calculate signal as EMA of recent MACD approximation
	// In production, you'd calculate MACD for each period and then EMA of those
	signal = macd // Simplified for now - TODO: proper signal line calculation
	histogram = macd - signal

	return macd, signal, histogram
}

// StochasticRSI calculates the Stochastic RSI indicator
// Returns a value between 0 and 1 (or 0-100 if scaled)
// StochRSI = (RSI - Lowest RSI) / (Highest RSI - Lowest RSI)
func StochasticRSI(closes []float64, rsiPeriod, stochPeriod int) float64 {
	if len(closes) < rsiPeriod+stochPeriod {
		return math.NaN()
	}

	// Calculate RSI values for the stochastic period
	rsiValues := make([]float64, stochPeriod)
	for i := 0; i < stochPeriod; i++ {
		endIdx := len(closes) - stochPeriod + i + 1
		rsiValues[i] = RSI(closes[:endIdx], rsiPeriod)
	}

	// Find highest and lowest RSI in the period
	currentRSI := rsiValues[len(rsiValues)-1]
	highRSI := rsiValues[0]
	lowRSI := rsiValues[0]

	for _, rsi := range rsiValues {
		if rsi > highRSI {
			highRSI = rsi
		}
		if rsi < lowRSI {
			lowRSI = rsi
		}
	}

	// Calculate StochRSI
	if highRSI == lowRSI {
		return 0.5 // Avoid division by zero, return midpoint
	}

	stochRSI := (currentRSI - lowRSI) / (highRSI - lowRSI)
	return stochRSI * 100 // Scale to 0-100
}

// ADX calculates the Average Directional Index
// Measures trend strength on a scale of 0-100
// ADX > 25 indicates a strong trend
// ADX < 20 indicates a weak trend or ranging market
func ADX(highs, lows, closes []float64, period int) float64 {
	if len(highs) != len(lows) || len(lows) != len(closes) {
		return math.NaN()
	}
	if len(closes) < period+1 {
		return math.NaN()
	}

	// Calculate +DM and -DM (Directional Movement)
	plusDM := make([]float64, 0, period)
	minusDM := make([]float64, 0, period)

	for i := len(closes) - period; i < len(closes); i++ {
		if i == 0 {
			continue
		}

		highDiff := highs[i] - highs[i-1]
		lowDiff := lows[i-1] - lows[i]

		plusDMVal := 0.0
		minusDMVal := 0.0

		if highDiff > lowDiff && highDiff > 0 {
			plusDMVal = highDiff
		}
		if lowDiff > highDiff && lowDiff > 0 {
			minusDMVal = lowDiff
		}

		plusDM = append(plusDM, plusDMVal)
		minusDM = append(minusDM, minusDMVal)
	}

	// Calculate smoothed +DM and -DM (using simple average for simplicity)
	smoothPlusDM := 0.0
	smoothMinusDM := 0.0
	for i := range plusDM {
		smoothPlusDM += plusDM[i]
		smoothMinusDM += minusDM[i]
	}
	smoothPlusDM /= float64(len(plusDM))
	smoothMinusDM /= float64(len(minusDM))

	// Calculate ATR for the period
	atr := ATR(highs, lows, closes, period)
	if atr == 0 {
		return 0
	}

	// Calculate +DI and -DI (Directional Indicators)
	plusDI := (smoothPlusDM / atr) * 100
	minusDI := (smoothMinusDM / atr) * 100

	// Calculate DX (Directional Index)
	diSum := plusDI + minusDI
	if diSum == 0 {
		return 0
	}
	dx := math.Abs(plusDI-minusDI) / diSum * 100

	// ADX is the smoothed average of DX
	// Simplified: returning DX as ADX approximation
	// In production, you'd calculate DX for each period and then smooth it
	return dx
}
