package app

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"trajon/internal/market"
)

func (a *application) openReplaySummary() {
	state := &a.chart
	if !state.replayHasStart || state.replaySummaryOpen || state.replayEnd <= 0 || state.replayEnd > len(a.candles) {
		return
	}
	state.pauseReplay()
	state.replayLineDrag = replayLineDrag{}
	// Mark at the last revealed close, never at the hover, viewport, or future.
	state.replaySummary = state.replayAccount.Statistics(a.replaySettings.InitialCapital, a.candles[state.replayEnd-1].Close)
	state.replaySummaryOpen = true
}

func (a *application) closeReplaySummary() {
	a.chart.replaySummaryOpen = false
	a.chart.replaySummary = market.ReplayStatistics{}
	if a.chart.replayHasStart {
		a.replayEndButton.Focus()
	}
}

func (a *application) discardReplay() {
	if !a.chart.replaySummaryOpen {
		return
	}
	a.chart.endReplay(len(a.candles))
	a.chart.chartHandle.Focus()
}

func (a *application) replaySummaryDialog(c *ui.Context) {
	state := &a.chart
	wasOpen := state.replaySummaryOpen
	components.Dialog(c.Key("replay-summary-dialog"), &state.replaySummaryOpen, components.DialogOptions{Label: "模拟交易统计"}, func(panel ui.Element) {
		// Focus the dialog, not its destructive action, on entry.
		panel.Width(400).Padding(18).Gap(16).Focusable().AutoFocus()
		components.Text(c, "模拟交易统计").FontSize(18).Bold().SingleLine()

		stats := state.replaySummary
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			replaySummaryRow(c, "初始资金", fmt.Sprintf("%.2f", stats.InitialCapital))
			replaySummaryRow(c, "账户权益", fmt.Sprintf("%.2f", stats.Equity))
			replaySummaryRow(c, "已实现盈亏", fmt.Sprintf("%+.2f", stats.RealizedPnL))
			replaySummaryRow(c, "浮动盈亏", fmt.Sprintf("%+.2f", stats.UnrealizedPnL))
			returnRate, hasReturn := stats.ReturnRate()
			replaySummaryRow(c, "收益率", replaySummaryPercent(returnRate, hasReturn))
			replaySummaryRow(c, "平仓次数", fmt.Sprint(stats.ClosingTrades))
			replaySummaryRow(c, "盈利 / 亏损 / 持平", fmt.Sprintf("%d / %d / %d", stats.Wins, stats.Losses, stats.Breakeven))
			winRate, hasWinRate := stats.WinRate()
			replaySummaryRow(c, "胜率", replaySummaryPercent(winRate, hasWinRate))
			ratio, hasRatio := stats.PayoffRatio()
			replaySummaryRow(c, "盈亏比", replaySummaryRatio(ratio, hasRatio))
		})
		ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
			components.Button(c.Key("replay-summary-discard"), "放弃", components.ButtonSecondary).
				OnClick(a.discardReplay)
			ui.Box(c.Key("replay-summary-save")).Disabled(true).Children(func() {
				components.Button(c, "保存", components.ButtonPrimary)
			})
		})
	})
	if wasOpen && !state.replaySummaryOpen {
		a.closeReplaySummary()
	}
}

func replaySummaryRow(c *ui.Context, label, value string) {
	ui.Row(c).FillWidth().Justify(ui.SpaceBetween).Gap(16).Children(func() {
		components.Text(c, label).TextColor(c.Theme().TextMuted).SingleLine()
		components.Text(c, value).SingleLine()
	})
}

func replaySummaryPercent(value float64, defined bool) string {
	if !defined {
		return "—"
	}
	return fmt.Sprintf("%.2f%%", value*100)
}

func replaySummaryRatio(value float64, defined bool) string {
	if !defined {
		return "—"
	}
	if math.IsInf(value, 1) {
		return "∞"
	}
	return fmt.Sprintf("%.2f", value)
}
