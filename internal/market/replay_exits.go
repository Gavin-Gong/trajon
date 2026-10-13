package market

import (
	"errors"
	"math"
	"time"
)

type ReplayExitKind uint8

const (
	ReplayTakeProfit ReplayExitKind = iota + 1
	ReplayStopLoss
)

var ErrInvalidReplayExit = errors.New("invalid replay exit")

type ReplayExit struct {
	Price       float64
	ActiveAfter time.Time
}

// SetExit brackets the current net position relative to its average price.
// Each new or modified exit starts with the next candle, never retroactively
// against the high/low of the candle already revealed when it was set.
func (a *ReplayAccount) SetExit(kind ReplayExitKind, price float64, stamp time.Time) error {
	if a.Quantity == 0 {
		return ErrNoReplayPosition
	}
	if !replayFinite(price) || price <= 0 || price == a.AveragePrice || stamp.IsZero() {
		return ErrInvalidReplayExit
	}
	profitable := (price > a.AveragePrice) == (a.Quantity > 0)
	if (kind == ReplayTakeProfit && !profitable) || (kind == ReplayStopLoss && profitable) {
		return ErrInvalidReplayExit
	}
	exit := ReplayExit{Price: price, ActiveAfter: stamp}
	switch kind {
	case ReplayTakeProfit:
		a.TakeProfit = exit
	case ReplayStopLoss:
		a.StopLoss = exit
	default:
		return ErrInvalidReplayExit
	}
	return nil
}

func (a *ReplayAccount) CancelExit(kind ReplayExitKind) {
	switch kind {
	case ReplayTakeProfit:
		a.TakeProfit = ReplayExit{}
	case ReplayStopLoss:
		a.StopLoss = ReplayExit{}
	}
}

// CheckExits closes the entire current position when a newly revealed candle
// touches a bracket. Stop-loss wins if both are touched; a gap executes at Open.
func (a *ReplayAccount) CheckExits(settings ReplayTradingSettings, candle Candle) (ReplayTrade, error) {
	if a.Quantity == 0 {
		return ReplayTrade{}, nil
	}
	active := func(exit ReplayExit) bool {
		return exit.Price > 0 && candle.Time.After(exit.ActiveAfter)
	}
	if !active(a.StopLoss) && !active(a.TakeProfit) {
		return ReplayTrade{}, nil
	}
	if !replayFinite(candle.Open) || !replayFinite(candle.High) || !replayFinite(candle.Low) ||
		!replayFinite(candle.Close) || candle.Low <= 0 || candle.High < candle.Low ||
		candle.High < max(candle.Open, candle.Close) || candle.Low > min(candle.Open, candle.Close) {
		return ReplayTrade{}, ErrInvalidReplayTrade
	}

	kind, price := ReplayExitKind(0), 0.0
	long := a.Quantity > 0
	if active(a.StopLoss) {
		level := a.StopLoss.Price
		if long && candle.Low <= level {
			kind, price = ReplayStopLoss, math.Min(level, candle.Open)
		} else if !long && candle.High >= level {
			kind, price = ReplayStopLoss, math.Max(level, candle.Open)
		}
	}
	if kind == 0 && active(a.TakeProfit) {
		level := a.TakeProfit.Price
		if long && candle.High >= level {
			kind, price = ReplayTakeProfit, math.Max(level, candle.Open)
		} else if !long && candle.Low <= level {
			kind, price = ReplayTakeProfit, math.Min(level, candle.Open)
		}
	}
	if kind == 0 {
		return ReplayTrade{}, nil
	}
	trade, err := a.closingTrade(price, candle.Time)
	if err != nil {
		return ReplayTrade{}, err
	}
	trade.ExitKind = kind
	if err := a.Execute(settings, trade); err != nil {
		return ReplayTrade{}, err
	}
	return trade, nil
}
