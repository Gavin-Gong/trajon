package market

import (
	"errors"
	"math"
	"time"
)

type ReplayTradingSettings struct {
	InitialCapital float64
	Leverage       float64
}

type ReplaySide uint8

const (
	ReplayBuy ReplaySide = iota + 1
	ReplaySell
)

var (
	ErrInvalidReplayTrade        = errors.New("invalid replay trade")
	ErrNoReplayPosition          = errors.New("no replay position")
	ErrInvalidReplaySettings     = errors.New("invalid replay trading settings")
	ErrInsufficientReplayCapital = errors.New("insufficient replay capital")
)

type ReplayTrade struct {
	Side     ReplaySide
	Quantity float64
	Price    float64
	Time     time.Time
	ExitKind ReplayExitKind
}

// ReplayAccount tracks a single instrument's net position. Positive quantities
// are long; negative quantities are short. Trades execute without fees or slippage.
// The session supplies the initial capital and leverage used for margin checks.
type ReplayAccount struct {
	Quantity     float64
	AveragePrice float64
	RealizedPnL  float64
	Trades       []ReplayTrade
	TakeProfit   ReplayExit
	StopLoss     ReplayExit
}

// Close executes an opposing order for the entire position, independently of
// the UI's order quantity.
func (a *ReplayAccount) Close(settings ReplayTradingSettings, price float64, stamp time.Time) (ReplayTrade, error) {
	trade, err := a.closingTrade(price, stamp)
	if err != nil {
		return ReplayTrade{}, err
	}
	if err := a.Execute(settings, trade); err != nil {
		return ReplayTrade{}, err
	}
	return trade, nil
}

func (a ReplayAccount) closingTrade(price float64, stamp time.Time) (ReplayTrade, error) {
	if a.Quantity == 0 {
		return ReplayTrade{}, ErrNoReplayPosition
	}
	side := ReplaySell
	if a.Quantity < 0 {
		side = ReplayBuy
	}
	return ReplayTrade{Side: side, Quantity: math.Abs(a.Quantity), Price: price, Time: stamp}, nil
}

func (a ReplayAccount) Equity(initialCapital, price float64) float64 {
	return initialCapital + a.RealizedPnL + (price-a.AveragePrice)*a.Quantity
}

// Execute closes the opposite position first, then opens any remaining quantity.
// Closing orders are allowed even when losses have exhausted available capital.
// Failed orders leave both the position and trade history unchanged.
func (a *ReplayAccount) Execute(settings ReplayTradingSettings, trade ReplayTrade) error {
	if !replayFinite(settings.InitialCapital) || settings.InitialCapital <= 0 ||
		!replayFinite(settings.Leverage) || settings.Leverage < 1 {
		return ErrInvalidReplaySettings
	}
	if (trade.Side != ReplayBuy && trade.Side != ReplaySell) ||
		!replayFinite(trade.Quantity) || trade.Quantity <= 0 ||
		!replayFinite(trade.Price) || trade.Price <= 0 || trade.Time.IsZero() {
		return ErrInvalidReplayTrade
	}

	next, _, _ := a.positionAfter(trade)
	equity := next.Equity(settings.InitialCapital, trade.Price)
	margin := math.Abs(next.Quantity) * (trade.Price / settings.Leverage)
	if !replayFinite(next.Quantity) || !replayFinite(next.AveragePrice) ||
		!replayFinite(next.RealizedPnL) || !replayFinite(equity) || !replayFinite(margin) {
		return ErrInvalidReplayTrade
	}
	sameDirection := a.Quantity == 0 || (a.Quantity > 0) == (trade.Side == ReplayBuy)
	opensPosition := sameDirection || trade.Quantity > math.Abs(a.Quantity)
	if opensPosition && margin > equity {
		return ErrInsufficientReplayCapital
	}

	if next.Quantity == 0 || (a.Quantity != 0 && (a.Quantity > 0) != (next.Quantity > 0)) {
		a.TakeProfit, a.StopLoss = ReplayExit{}, ReplayExit{}
	}
	a.Quantity, a.AveragePrice, a.RealizedPnL = next.Quantity, next.AveragePrice, next.RealizedPnL
	a.Trades = append(a.Trades, trade)
	return nil
}

// positionAfter applies a validated fill without modifying the account or its
// trade slice. The returned P&L and flag describe only its closing portion.
func (a ReplayAccount) positionAfter(trade ReplayTrade) (next ReplayAccount, pnl float64, closed bool) {
	delta := trade.Quantity
	if trade.Side == ReplaySell {
		delta = -delta
	}
	next = a
	next.Quantity += delta
	if a.Quantity == 0 || (a.Quantity > 0) == (delta > 0) {
		total := math.Abs(next.Quantity)
		next.AveragePrice = a.AveragePrice*(math.Abs(a.Quantity)/total) + trade.Price*(trade.Quantity/total)
		return next, 0, false
	}

	pnl = min(math.Abs(a.Quantity), trade.Quantity) * (trade.Price - a.AveragePrice) * math.Copysign(1, a.Quantity)
	next.RealizedPnL += pnl
	switch {
	case next.Quantity == 0:
		next.AveragePrice = 0
	case trade.Quantity > math.Abs(a.Quantity):
		next.AveragePrice = trade.Price
	}
	return next, pnl, true
}

func replayFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
