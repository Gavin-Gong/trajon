package app

import (
	"errors"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/market"
)

var errReplayTradingUnavailable = errors.New("replay trading unavailable")

func (s *klineState) clearReplayTrading() {
	s.replayAccount = market.ReplayAccount{}
	s.replayLineDrag = replayLineDrag{}
	s.replayExitError = nil
	s.replayTradeMessage = ""
	s.replayTradeMessageUntil = time.Time{}
}

func (a *application) canTradeReplay() bool {
	state := &a.chart
	return state.replayHasStart && !state.replaySelecting && !state.replayComplete &&
		!a.marketLoading && state.replayEnd > 0 && state.replayEnd <= len(a.candles)
}

func (a *application) placeReplayTrade(side market.ReplaySide) (market.ReplayTrade, error) {
	if !a.canTradeReplay() {
		return market.ReplayTrade{}, errReplayTradingUnavailable
	}
	// The replay cutoff, not the viewport or hover, owns the execution price.
	candle := a.candles[a.chart.replayEnd-1]
	trade := market.ReplayTrade{
		Side:     side,
		Quantity: a.chart.replayOrderQuantity,
		Price:    candle.Close,
		Time:     candle.Time,
	}
	if err := a.chart.replayAccount.Execute(a.replaySettings, trade); err != nil {
		return market.ReplayTrade{}, err
	}
	a.chart.replayLineDrag = replayLineDrag{}
	return trade, nil
}

func (a *application) submitReplayTrade(c *ui.Context, side market.ReplaySide) {
	trade, err := a.placeReplayTrade(side)
	if err != nil {
		message := "当前无法下单"
		switch {
		case errors.Is(err, market.ErrInsufficientReplayCapital):
			message = "资金不足"
		case errors.Is(err, market.ErrInvalidReplayTrade):
			message = "下单数量或价格无效"
		case errors.Is(err, market.ErrInvalidReplaySettings):
			message = "回放资金设置无效"
		}
		a.showReplayTradeMessage(c, message)
		return
	}
	label := "买入"
	if side == market.ReplaySell {
		label = "卖出"
	}
	a.showReplayTradeMessage(c, replayFillMessage(label, trade))
}

func (a *application) showReplayTradeMessage(c *ui.Context, message string) {
	now := c.Now()
	if now.IsZero() {
		// Native HandleInput callbacks run between builds, without a frame clock.
		now = time.Now()
	}
	a.chart.replayTradeMessage = message
	a.chart.replayTradeMessageUntil = now.Add(3 * time.Second)
	c.Announce(message)
}

func (a *application) replayTradeFeedback(c *ui.Context) {
	state := &a.chart
	if !state.replayHasStart || state.replayTradeMessage == "" || !c.Now().Before(state.replayTradeMessageUntil) {
		return
	}
	c.After(state.replayTradeMessageUntil.Sub(c.Now()))
	top := state.actionBarY - 28
	if top < 0 {
		top = state.actionBarY + klineActionBarHeight + 4
	}
	// Trade feedback must never intercept repeated orders or playback controls.
	ui.Box(c.Key("replay-trade-feedback")).Absolute().Left(state.actionBarX).Top(top).
		Size(state.actionBarWidth, 24).Padding(4, 6).Radius(6).
		Background(c.Theme().Background.Alpha(.9)).PassThrough().Role(ui.RoleStatus).Children(func() {
		components.Text(c, state.replayTradeMessage).FontSize(12).SingleLine().FillWidth().TextAlign(ui.Center).PassThrough()
	})
}

func (a *application) replayTradingControls(c *ui.Context) {
	canTrade := a.canTradeReplay()
	ui.Box(c).Size(1, 16).Background(c.Theme().Border)
	ui.Box(c.Key("replay-order-quantity")).Size(klineReplayQuantityWidth, 28).FontSize(12).
		Disabled(!canTrade).Children(func() {
		components.NumberInput(c, &a.chart.replayOrderQuantity, 0, math.MaxFloat64, .01).
			Label("下单数量").FillWidth().Height(28)
	})
	klineActionIconButton(c.Key("replay-buy"), lucide.TrendingUp, "买入", false).
		Disabled(!canTrade || a.chart.replayOrderQuantity <= 0).
		OnClick(func() { a.submitReplayTrade(c, market.ReplayBuy) })
	klineActionIconButton(c.Key("replay-sell"), lucide.TrendingDown, "卖出", false).
		Disabled(!canTrade || a.chart.replayOrderQuantity <= 0).
		OnClick(func() { a.submitReplayTrade(c, market.ReplaySell) })
}
