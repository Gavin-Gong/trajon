package market

func AddEMA20(candles []Candle) {
	if len(candles) == 0 {
		return
	}
	ema := candles[0].Close
	alpha := 2.0 / 21.0
	for i := range candles {
		if i > 0 {
			ema += alpha * (candles[i].Close - ema)
		}
		candles[i].EMA20 = ema
	}
}
