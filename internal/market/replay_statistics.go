package market

import "math"

// ReplayStatistics counts each fill that closes any quantity as one outcome,
// including partial closes and the closing portion of a reversal. Opening fills
// and unrealized P&L do not affect win rate or the average win/loss ratio.
type ReplayStatistics struct {
	InitialCapital float64
	Equity         float64
	RealizedPnL    float64
	UnrealizedPnL  float64
	ClosingTrades  int
	Wins           int
	Losses         int
	Breakeven      int
	GrossProfit    float64
	GrossLoss      float64 // Positive magnitude of realized losses.
}

// Statistics is a read-only snapshot marked at the supplied replay price.
func (a ReplayAccount) Statistics(initialCapital, price float64) ReplayStatistics {
	stats := ReplayStatistics{
		InitialCapital: initialCapital,
		Equity:         a.Equity(initialCapital, price),
		RealizedPnL:    a.RealizedPnL,
		UnrealizedPnL:  (price - a.AveragePrice) * a.Quantity,
	}
	var position ReplayAccount
	for _, trade := range a.Trades {
		next, pnl, closed := position.positionAfter(trade)
		position = next
		if !closed {
			continue
		}
		stats.ClosingTrades++
		switch {
		case pnl > 0:
			stats.Wins++
			stats.GrossProfit += pnl
		case pnl < 0:
			stats.Losses++
			stats.GrossLoss -= pnl
		default:
			stats.Breakeven++
		}
	}
	return stats
}

// WinRate includes breakeven closes in its denominator. No closes is undefined.
func (s ReplayStatistics) WinRate() (float64, bool) {
	if s.ClosingTrades == 0 {
		return 0, false
	}
	return float64(s.Wins) / float64(s.ClosingTrades), true
}

// PayoffRatio is average realized profit / average realized loss, not gross
// profit / gross loss. Winning without losses yields +Inf; all-breakeven or no
// closes is undefined, and losses without wins yields zero.
func (s ReplayStatistics) PayoffRatio() (float64, bool) {
	if s.Losses == 0 {
		if s.Wins > 0 {
			return math.Inf(1), true
		}
		return 0, false
	}
	if s.Wins == 0 {
		return 0, true
	}
	return (s.GrossProfit / float64(s.Wins)) / (s.GrossLoss / float64(s.Losses)), true
}

// ReturnRate includes unrealized P&L at the replay cursor.
func (s ReplayStatistics) ReturnRate() (float64, bool) {
	if s.InitialCapital <= 0 || !replayFinite(s.InitialCapital) {
		return 0, false
	}
	return (s.RealizedPnL + s.UnrealizedPnL) / s.InitialCapital, true
}
