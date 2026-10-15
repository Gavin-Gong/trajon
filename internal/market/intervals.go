package market

// CandleIntervals lists the intervals supported by all bundled candle sources,
// in ascending order. Uppercase D/W/M distinguish calendar units from minutes.
func CandleIntervals() []string {
	return []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "1D", "1W", "1M"}
}
