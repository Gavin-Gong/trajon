package app

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
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
	components.Dialog(c.Key("replay-summary-dialog"), &state.replaySummaryOpen, components.DialogOptions{Label: a.text().ReplaySummary()}, func(panel ui.Element) {
		// Focus the dialog, not its destructive action, on entry.
		panel.Width(560).Padding(20).Gap(18).FontFeatures("tnum").Focusable().AutoFocus()
		text, theme := a.text(), c.Theme()
		ui.Row(c).FillWidth().AlignItems(ui.Center).Shrink(0).Children(func() {
			components.Text(c, text.ReplaySummary()).FontSize(18).Bold().SingleLine()
			ui.Spacer(c)
			if currency := a.currentInstrument.Currency; currency != "" {
				components.Text(c, currency).FontSize(13).TextColor(theme.TextMuted).SingleLine()
			}
		})

		stats := state.replaySummary
		components.ScrollView(c.Key("replay-summary-metrics"), components.ScrollViewOptions{}).
			FillWidth().MaxHeight(390).MinHeight(0).Shrink(1).Children(func() {
			ui.Column(c).FillWidth().Gap(20).Children(func() {
				ui.Column(c).FillWidth().Padding(16).Gap(14).Radius(12).Background(theme.Surface).Children(func() {
					ui.Grid(c).FillWidth().Columns(2).Gap(16).AlignItems(ui.Center).Children(func() {
						ui.Column(c).MinWidth(0).Gap(6).Children(func() {
							replaySummaryLabel(c, text.Equity(), lucide.Wallet)
							components.Text(c, fmt.Sprintf("%.2f", stats.Equity)).FontSize(30).Bold().SingleLine()
						})
						ui.Column(c).MinWidth(0).AlignItems(ui.End).Gap(6).Children(func() {
							components.Text(c, text.ReturnRate()).FontSize(13).TextColor(theme.TextMuted).SingleLine()
							returnRate, hasReturn := stats.ReturnRate()
							color := replaySummaryPnLColor(theme, stats.RealizedPnL+stats.UnrealizedPnL)
							ui.Row(c).Gap(6).AlignItems(ui.Center).TextColor(color).Children(func() {
								icon := lucide.Minus
								if hasReturn && returnRate > 0 {
									icon = lucide.TrendingUp
								} else if hasReturn && returnRate < 0 {
									icon = lucide.TrendingDown
								}
								components.Icon(c, icon).Size(20, 20)
								components.Text(c, replaySummaryPercent(returnRate, hasReturn)).FontSize(24).Bold().SingleLine()
							})
						})
					})
					ui.Row(c).FillWidth().Gap(8).Children(func() {
						components.Text(c, a.initialCapitalLabel()).FontSize(13).TextColor(theme.TextMuted).SingleLine()
						components.Text(c, fmt.Sprintf("%.2f", stats.InitialCapital)).FontSize(13).SingleLine()
					})
				})
				ui.Grid(c).FillWidth().Columns(2).Gap(20).Children(func() {
					replaySummaryMetric(c, text.RealizedPnL(), fmt.Sprintf("%+.2f", stats.RealizedPnL), lucide.Banknote, replaySummaryPnLColor(theme, stats.RealizedPnL))
					replaySummaryMetric(c, text.UnrealizedPnL(), fmt.Sprintf("%+.2f", stats.UnrealizedPnL), lucide.Activity, replaySummaryPnLColor(theme, stats.UnrealizedPnL))
				})
				ui.Box(c).FillWidth().Height(1).Background(theme.Border)
				ui.Row(c).FillWidth().Gap(24).AlignItems(ui.Center).Children(func() {
					replaySummaryOutcomes(c, stats, text.WinRate())
					ui.Column(c).Grow(1).MinWidth(0).Gap(16).Children(func() {
						ui.Grid(c).FillWidth().Columns(2).Gap(16).Children(func() {
							replaySummaryMetric(c, text.ClosingTrades(), fmt.Sprint(stats.ClosingTrades), lucide.ArrowRightLeft, theme.Text)
							ratio, hasRatio := stats.PayoffRatio()
							replaySummaryMetric(c, text.PayoffRatio(), replaySummaryRatio(ratio, hasRatio), lucide.Scale, theme.Text)
						})
						ui.Grid(c).FillWidth().Columns(3).Gap(12).Role(ui.RoleGroup).Label(text.WinLossBreakeven()).Children(func() {
							replaySummaryOutcome(c, text.WinningCloses(), stats.Wins, replaySummaryPnLColor(theme, 1))
							replaySummaryOutcome(c, text.LosingCloses(), stats.Losses, replaySummaryPnLColor(theme, -1))
							replaySummaryOutcome(c, text.BreakevenCloses(), stats.Breakeven, theme.TextMuted)
						})
					})
				})
			})
		})
		ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Shrink(0).Children(func() {
			components.Button(c.Key("replay-summary-discard"), a.text().Discard(), components.ButtonSecondary).
				OnClick(a.discardReplay)
			ui.Box(c.Key("replay-summary-save")).Disabled(true).Children(func() {
				components.Button(c, a.text().Save(), components.ButtonPrimary)
			})
		})
	})
	if wasOpen && !state.replaySummaryOpen {
		a.closeReplaySummary()
	}
}

func replaySummaryLabel(c *ui.Context, label string, icon func() *ui.SVG) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).TextColor(c.Theme().TextMuted).Children(func() {
		components.Icon(c, icon).Size(14, 14).Shrink(0)
		components.Text(c, label).FontSize(13).SingleLine()
	})
}

func replaySummaryMetric(c *ui.Context, label, value string, icon func() *ui.SVG, color ui.Color) {
	ui.Column(c).MinWidth(0).Gap(6).Children(func() {
		replaySummaryLabel(c, label, icon)
		components.Text(c, value).FontSize(22).FontWeight(600).TextColor(color).SingleLine()
	})
}

func replaySummaryOutcome(c *ui.Context, label string, count int, color ui.Color) {
	ui.Column(c).MinWidth(0).Gap(5).Children(func() {
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(color).Shrink(0)
			components.Text(c, label).FontSize(13).TextColor(c.Theme().TextMuted).SingleLine()
		})
		components.Text(c, fmt.Sprint(count)).FontSize(18).FontWeight(600).SingleLine()
	})
}

func replaySummaryOutcomes(c *ui.Context, stats market.ReplayStatistics, label string) {
	theme := c.Theme()
	values := make([]float64, 0, 3)
	colors := make([]ui.Color, 0, 3)
	palette := []ui.Color{replaySummaryPnLColor(theme, 1), replaySummaryPnLColor(theme, -1), theme.TextMuted}
	for i, count := range []int{stats.Wins, stats.Losses, stats.Breakeven} {
		if count > 0 {
			values = append(values, float64(count))
			colors = append(colors, palette[i])
		}
	}
	pad := 0.0
	if len(values) > 1 {
		pad = .035
	}
	slices := chart.Pie(values, pad)
	track := theme.Border
	ui.Box(c).Size(112, 112).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		outer := min(r.W, r.H) / 2
		inner := outer - 9
		if len(slices) == 0 {
			p.FillPath(chart.Sector(cx, cy, inner, outer, chart.Slice{End: 2 * math.Pi}), track)
		}
		for i, slice := range slices {
			p.FillPath(chart.Sector(cx, cy, inner, outer, slice), colors[i])
		}
	}).Children(func() {
		ui.Column(c).Fill().Center().Gap(4).Children(func() {
			winRate, defined := stats.WinRate()
			components.Text(c, replaySummaryPercent(winRate, defined)).FontSize(18).FontWeight(600).SingleLine()
			components.Text(c, label).FontSize(13).TextColor(theme.TextMuted).SingleLine()
		})
	})
}

func replaySummaryPnLColor(theme *ui.Theme, pnl float64) ui.Color {
	if pnl > 0 {
		if theme.Dark {
			return ui.Hex("#44c58b")
		}
		return ui.Hex("#168253")
	}
	if pnl < 0 {
		if theme.Dark {
			return ui.Hex("#f07880")
		}
		return ui.Hex("#c43d4b")
	}
	return theme.Text
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
