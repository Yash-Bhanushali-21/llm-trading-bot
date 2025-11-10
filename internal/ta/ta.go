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

func EMA(closes []float64, period int) float64 {
	if len(closes) < period || period <= 0 {
		return math.NaN()
	}

	k := 2.0 / float64(period+1)
	ema := SMA(closes[:period], period)

	for i := period; i < len(closes); i++ {
		ema = closes[i]*k + ema*(1-k)
	}

	return ema
}

func MACD(closes []float64, fastPeriod, slowPeriod, signalPeriod int) (macd, signal, histogram float64) {
	if len(closes) < slowPeriod {
		return math.NaN(), math.NaN(), math.NaN()
	}

	fastEMA := EMA(closes, fastPeriod)
	slowEMA := EMA(closes, slowPeriod)
	macd = fastEMA - slowEMA

	signal = macd // Simplified for now - TODO: proper signal line calculation
	histogram = macd - signal

	return macd, signal, histogram
}

func StochasticRSI(closes []float64, rsiPeriod, stochPeriod int) float64 {
	if len(closes) < rsiPeriod+stochPeriod {
		return math.NaN()
	}

	rsiValues := make([]float64, stochPeriod)
	for i := 0; i < stochPeriod; i++ {
		endIdx := len(closes) - stochPeriod + i + 1
		rsiValues[i] = RSI(closes[:endIdx], rsiPeriod)
	}

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

	if highRSI == lowRSI {
		return 0.5 // Avoid division by zero, return midpoint
	}

	stochRSI := (currentRSI - lowRSI) / (highRSI - lowRSI)
	return stochRSI * 100 // Scale to 0-100
}

func ADX(highs, lows, closes []float64, period int) float64 {
	if len(highs) != len(lows) || len(lows) != len(closes) {
		return math.NaN()
	}
	if len(closes) < period+1 {
		return math.NaN()
	}

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

	smoothPlusDM := 0.0
	smoothMinusDM := 0.0
	for i := range plusDM {
		smoothPlusDM += plusDM[i]
		smoothMinusDM += minusDM[i]
	}
	smoothPlusDM /= float64(len(plusDM))
	smoothMinusDM /= float64(len(minusDM))

	atr := ATR(highs, lows, closes, period)
	if atr == 0 {
		return 0
	}

	plusDI := (smoothPlusDM / atr) * 100
	minusDI := (smoothMinusDM / atr) * 100

	diSum := plusDI + minusDI
	if diSum == 0 {
		return 0
	}
	dx := math.Abs(plusDI-minusDI) / diSum * 100

	return dx
}
