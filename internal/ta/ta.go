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
