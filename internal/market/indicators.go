package market

// ComputeEMA returns a close-price EMA aligned with candles, seeded by the
// first close. It does not mutate candles. Non-positive periods return nil.
func ComputeEMA(candles []Candle, period int) []float64 {
	if len(candles) == 0 || period <= 0 {
		return nil
	}
	values := make([]float64, len(candles))
	ema := candles[0].Close
	alpha := 2.0 / (float64(period) + 1)
	for i, candle := range candles {
		if i > 0 {
			ema += alpha * (candle.Close - ema)
		}
		values[i] = ema
	}
	return values
}

// MACDSeries contains candle-aligned DIF, DEA (signal), and histogram values.
// Histogram uses the convention 2 * (DIF - DEA).
type MACDSeries struct {
	DIF, DEA, Histogram []float64
}

// ComputeMACD uses first-close seeded EMAs, and a first-DIF seeded signal EMA.
// Invalid periods (including fast >= slow) return an empty series.
func ComputeMACD(candles []Candle, fast, slow, signal int) MACDSeries {
	if len(candles) == 0 || fast <= 0 || slow <= fast || signal <= 0 {
		return MACDSeries{}
	}
	short, long := ComputeEMA(candles, fast), ComputeEMA(candles, slow)
	result := MACDSeries{
		DIF: make([]float64, len(candles)), DEA: make([]float64, len(candles)),
		Histogram: make([]float64, len(candles)),
	}
	alpha, dea := 2.0/(float64(signal)+1), 0.0
	for i := range candles {
		dif := short[i] - long[i]
		if i == 0 {
			dea = dif
		} else {
			dea += alpha * (dif - dea)
		}
		result.DIF[i], result.DEA[i] = dif, dea
		result.Histogram[i] = 2 * (dif - dea)
	}
	return result
}

func AddEMA20(candles []Candle) {
	for i, value := range ComputeEMA(candles, 20) {
		candles[i].EMA20 = value
	}
}
