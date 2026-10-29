package app

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/journal"
)

func (a *application) overviewPage(c *ui.Context) {
	if !a.records.ready {
		a.recordsLoadState(c)
		return
	}
	now := time.Now()
	a.ensureOverview(now)
	s, text := &a.overview, a.text()
	result := s.result
	if result == nil {
		result = &journal.Performance{}
	}
	components.ScrollView(c.Key("overview-scroll"), components.ScrollViewOptions{}).Fill().Children(func() {
		root := ui.Column(c.Key("overview-page")).FillWidth().Padding(24).Gap(20).FontFeatures("tnum")
		width := root.LayoutBounds().W
		if width <= 0 {
			width, _ = c.Size()
		}
		root.Children(func() {
			a.overviewFilters(c, result)
			if s.loading {
				ui.Column(c.Key("calculating")).FillWidth().Height(200).Center().Children(func() {
					components.Text(c, text.OverviewCalculating()).TextColor(c.Theme().TextMuted)
				})
				return
			}
			if len(result.Issues) > 0 {
				a.overviewIssues(c, result)
			}
			a.overviewMetrics(c, result, width)
			if result.Count == 0 {
				ui.Column(c.Key("empty")).FillWidth().Padding(28).Gap(14).Center().Children(func() {
					components.Text(c, text.OverviewNoTrades()).TextColor(c.Theme().TextMuted)
					components.Button(c.Key("new"), text.JournalNewRecord(), components.ButtonSecondary).OnClick(func() {
						a.activePage = "journal"
						a.newJournalRecord()
					})
				})
			}
			tracks := []ui.Track{ui.Fr(2), ui.Fr(1)}
			if width < 800 {
				tracks = []ui.Track{ui.Fr(1)}
			}
			ui.Grid(c.Key("plots")).FillWidth().ColumnTracks(tracks...).Gap(16).Children(func() {
				overviewCard(c, "trend", text.OverviewCumulativePnL()).Children(func() { a.overviewTrend(c, result, now) })
				overviewCard(c, "outcomes", text.OverviewOutcomes()).Children(func() { a.overviewOutcomes(c, result) })
			})
			a.overviewHeatmap(c, result, now)
			ui.Grid(c.Key("details")).FillWidth().ColumnTracks(tracks...).Gap(16).Children(func() {
				a.overviewRecent(c, result)
				a.overviewGroups(c, result)
			})
		})
	})
}

func overviewCard(c *ui.Context, key, title string) ui.Element {
	card := ui.Column(c.Key(key)).MinWidth(0).Padding(18).Gap(16).Radius(12).
		Background(c.Theme().Surface).Border(1, c.Theme().Border.Alpha(.65))
	if title != "" {
		card.Children(func() { components.Text(c, title).FontSize(13).Bold().SingleLine() })
	}
	return card
}

func (a *application) overviewFilters(c *ui.Context, result *journal.Performance) {
	s, text := &a.overview, a.text()
	ui.Row(c.Key("filters")).FillWidth().Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
		periods := []components.NativeSelectOption[int]{
			{Value: 0, Label: text.OverviewThisMonth()}, {Value: 1, Label: text.OverviewLastMonth()},
			{Value: 2, Label: text.OverviewThisYear()}, {Value: 3, Label: text.OverviewAllTime()},
		}
		components.NativeSelect(c.Key("period"), &s.period, periods).Trigger.Label(text.OverviewPeriod()).Height(30)
		instruments := []components.NativeSelectOption[string]{{Value: "", Label: text.OverviewAllInstruments()}}
		for _, name := range result.Instruments {
			instruments = append(instruments, components.NativeSelectOption[string]{Value: name, Label: name})
		}
		components.NativeSelect(c.Key("instrument"), &s.instrument, instruments).Trigger.Label(text.JournalInstrument()).MaxWidth(200).Height(30)
		ui.Spacer(c)
		components.Text(c, text.OverviewTradeCount(result.Count)).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine()
		recordIconButton(c, "policy", lucide.Info(), text.OverviewBeforeFees()).Size(24, 24)
	})
}

func (a *application) overviewMetrics(c *ui.Context, result *journal.Performance, width float32) {
	text := a.text()
	columns := 4
	if width < 800 {
		columns = 2
	}
	pnl, drawdown := "—", "—"
	if result.Count > 0 {
		pnl, drawdown = text.OverviewMoney(overviewMoney(result.PnL)), text.OverviewMoney(overviewMoney(-result.MaxDrawdown))
	}
	rate, rateOK := result.WinRate()
	ratio, ratioOK := result.PayoffRatio()
	ui.Grid(c.Key("metrics")).FillWidth().Columns(columns).Gap(16).Children(func() {
		for i, metric := range []struct {
			label, value, help string
			color              ui.Color
		}{
			{text.OverviewTotalPnL(), pnl, text.OverviewBeforeFees(), replaySummaryPnLColor(c.Theme(), result.PnL)},
			{text.WinRate(), replaySummaryPercent(rate, rateOK), text.OverviewPayoffHelp(), c.Theme().Text},
			{text.PayoffRatio(), replaySummaryRatio(ratio, ratioOK), text.OverviewPayoffHelp(), c.Theme().Text},
			{text.OverviewMaxDrawdown(), drawdown, text.OverviewDrawdownHelp(), c.Theme().Text},
		} {
			ui.Column(c.Key(i)).MinWidth(0).Padding(4, 0).Gap(8).Children(func() {
				components.Text(c, metric.label).FontSize(12).TextColor(c.Theme().TextMuted).Tooltip(metric.help).SingleLine()
				components.Text(c, metric.value).FontSize(27).FontWeight(600).TextColor(metric.color).SingleLine().MinWidth(0).Tooltip(metric.value)
			})
		}
	})
}

func overviewMoney(amount float64) string {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return "—"
	}
	if amount == 0 {
		return "0.00"
	}
	if math.Abs(amount) < .005 {
		return fmt.Sprintf("%+.4g", amount)
	}
	return fmt.Sprintf("%+.2f", amount)
}

func (a *application) overviewOutcomes(c *ui.Context, result *journal.Performance) {
	text, theme := a.text(), c.Theme()
	// Share the existing accessible/native presentation, not replay statistics:
	// journal outcomes count whole records, never partial closing fills.
	values := []int{result.Wins, result.Losses, result.Breakeven}
	palette := []ui.Color{replaySummaryPnLColor(theme, 1), replaySummaryPnLColor(theme, -1), theme.TextMuted}
	ui.Column(c).FillWidth().AlignItems(ui.Center).Gap(16).Children(func() {
		a.overviewOutcomeRing(c, result, palette)
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			for i, label := range []string{text.OverviewProfit(), text.OverviewLoss(), text.OverviewBreakeven()} {
				ui.Row(c.Key(i)).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(7, 7).Radius(4).Background(palette[i])
					components.Text(c, label).FontSize(12).TextColor(theme.TextMuted)
					ui.Spacer(c)
					components.Text(c, strconv.Itoa(values[i])).FontSize(13).Bold()
				})
			}
		})
	})
}

func (a *application) overviewRecent(c *ui.Context, result *journal.Performance) {
	text := a.text()
	overviewCard(c, "recent", "").Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			components.Text(c, text.OverviewRecentTrades()).FontSize(13).Bold()
			ui.Spacer(c)
			components.Button(c.Key("all"), text.OverviewViewAll(), components.ButtonGhost).Height(26).FontSize(12).OnClick(func() {
				a.activePage = "journal"
				a.recordsBrowser.filter, a.recordsBrowser.query = int(journal.ClosedRecords), a.overview.instrument
			})
		})
		if len(result.Trades) == 0 {
			components.Text(c, "—").TextColor(c.Theme().TextMuted)
		}
		for i := len(result.Trades) - 1; i >= max(0, len(result.Trades)-5); i-- {
			a.overviewTradeRow(c, result.Trades[i])
		}
	})
}

func (a *application) overviewTradeRow(c *ui.Context, trade journal.Trade) {
	text := a.text()
	name := trade.Instrument
	if name == "" {
		name = text.JournalUntitled()
	}
	components.Button(c.Key(trade.ID), "", components.ButtonGhost).FillWidth().MinWidth(0).Padding(8, 0).Radius(6).
		Label(text.JournalOpenRecord(name)).OnClick(func() { a.openOverviewRecord(trade.ID) }).Children(func() {
		ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				components.Text(c, name).FontSize(13).Bold().SingleLine().FillWidth().MinWidth(0)
				components.Text(c, a.recordDirection(trade.Direction)).FontSize(11).TextColor(c.Theme().TextMuted).SingleLine()
			})
			ui.Column(c).Gap(4).AlignItems(ui.End).Children(func() {
				components.Text(c, text.OverviewMoney(overviewMoney(trade.PnL))).FontSize(13).Bold().TextColor(replaySummaryPnLColor(c.Theme(), trade.PnL)).SingleLine()
				components.Text(c, trade.ClosedAt.Local().Format("01-02 15:04")).FontSize(11).TextColor(c.Theme().TextMuted).SingleLine().
					Tooltip(text.OverviewClosedAt() + ": " + trade.ClosedAt.Local().Format("2006-01-02 15:04") + "\n" + text.OverviewHoldTime() + ": " + trade.ClosedAt.Sub(trade.OpenedAt).Round(time.Minute).String())
			})
		})
	})
}

func (a *application) overviewGroups(c *ui.Context, result *journal.Performance) {
	s, text := &a.overview, a.text()
	overviewCard(c, "groups", "").Children(func() {
		options := []components.NativeSelectOption[bool]{{Value: false, Label: text.OverviewByInstrument()}, {Value: true, Label: text.OverviewByTag()}}
		components.NativeSelect(c.Key("group-kind"), &s.byTag, options).Trigger.Height(28)
		groups := result.ByInstrument
		if s.byTag {
			groups = result.ByTag
		}
		if len(groups) == 0 {
			components.Text(c, "—").TextColor(c.Theme().TextMuted)
		}
		for _, group := range groups[:min(len(groups), 5)] {
			name := group.Name
			if name == "" {
				name = text.JournalUntitled()
			}
			if s.byTag {
				name = "#" + name
			}
			components.Button(c.Key(struct {
				Tag  bool
				Name string
			}{s.byTag, group.Name}), "", components.ButtonGhost).FillWidth().MinWidth(0).Padding(6, 0).
				Label(name).OnClick(func() {
				if s.byTag {
					a.activePage = "journal"
					a.recordsBrowser.query, a.recordsBrowser.filter = group.Name, int(journal.ClosedRecords)
				} else {
					s.instrument = group.Name
				}
			}).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(5).Children(func() {
					ui.Row(c).FillWidth().Gap(8).Children(func() {
						components.Text(c, name).FontSize(12).SingleLine().Grow(1).MinWidth(0)
						components.Text(c, text.OverviewMoney(overviewMoney(group.PnL))).FontSize(12).Bold().SingleLine().TextColor(replaySummaryPnLColor(c.Theme(), group.PnL))
					})
					rate, ok := group.WinRate()
					ui.Row(c).FillWidth().Children(func() {
						components.Text(c, text.OverviewTradeCount(group.Count)).FontSize(11).TextColor(c.Theme().TextMuted)
						ui.Spacer(c)
						components.Text(c, replaySummaryPercent(rate, ok)).FontSize(11).TextColor(c.Theme().TextMuted).Tooltip(text.WinRate())
					})
				})
			})
		}
	})
}

func (a *application) overviewIssues(c *ui.Context, result *journal.Performance) {
	text := a.text()
	ui.Column(c.Key("issues")).FillWidth().Padding(12).Radius(8).Background(c.Theme().Text.Alpha(.035)).Gap(8).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			components.Text(c, text.OverviewIncomplete(len(result.Issues))).FontSize(12).TextColor(c.Theme().TextMuted).Wrap().Grow(1).MinWidth(0)
			components.Button(c.Key("toggle"), text.OverviewRepair(), components.ButtonGhost).Height(26).FontSize(12).OnClick(func() { a.overview.showIssues = !a.overview.showIssues })
		})
		if a.overview.showIssues {
			components.ScrollView(c.Key("issue-list"), components.ScrollViewOptions{}).FillWidth().MaxHeight(220).Children(func() {
				ui.Column(c).FillWidth().Gap(4).Children(func() {
					for _, issue := range result.Issues {
						name := issue.Instrument
						if name == "" {
							name = text.JournalUntitled()
						}
						components.Button(c.Key(issue.ID), "", components.ButtonGhost).FillWidth().MinWidth(0).Padding(8).Label(text.JournalOpenRecord(name)).
							OnClick(func() { a.openOverviewRecord(issue.ID) }).Children(func() {
							ui.Row(c).FillWidth().Gap(8).Children(func() {
								components.Text(c, name).FontSize(12).SingleLine().Grow(1).MinWidth(0)
								components.Text(c, text.OverviewMissingExecution()).FontSize(11).TextColor(c.Theme().TextMuted).SingleLine()
							})
						})
					}
				})
			})
		}
	})
}
