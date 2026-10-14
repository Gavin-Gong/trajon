package app

import (
	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

func (a *application) toggleReplay(c *ui.Context) {
	if a.chart.actionPlaying {
		a.chart.pauseReplay()
	} else {
		a.startReplay(c)
	}
}

func (a *application) replayActionShortcuts(c *ui.Context, bar ui.Element) {
	state := &a.chart
	if !state.replayHasStart || state.replaySelecting || state.replayComplete || state.replaySummaryOpen || state.textEditing {
		return
	}
	// Keep editor, selector, toolbar and position-button keyboard behavior native.
	// Action-bar buttons share the chart's commands, including Space after an order.
	if !state.chartHandle.Focused(c) && !bar.FocusWithin() {
		return
	}
	if state.replayQuantityHandle.FocusWithin(c) || state.replaySpeedHandle.FocusWithin(c) {
		return
	}
	state.chartHandle.OnShortcut(c, 0, ui.KeySpace, func() { a.toggleReplay(c) })
	if !state.actionPlaying && !a.marketLoading {
		state.chartHandle.OnShortcut(c, 0, ui.KeyRight, func() { a.advanceReplayWithFeedback(c) })
	}
	if a.canTradeReplay() && state.replayOrderQuantity > 0 {
		state.chartHandle.OnShortcut(c, 0, ui.KeyUp, func() { a.submitReplayTrade(c, market.ReplayBuy) })
		state.chartHandle.OnShortcut(c, 0, ui.KeyDown, func() { a.submitReplayTrade(c, market.ReplaySell) })
	}
}
