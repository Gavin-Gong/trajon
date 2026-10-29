package app

import (
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/journal"
)

func (a *application) overviewHeatmap(c *ui.Context, result *journal.Performance, now time.Time) {
	s, text := &a.overview, a.text()
	from, _ := overviewRange(s.period, now)
	year := now.Year()
	if s.period == 3 {
		year = s.heatmapYear
		if year == 0 && len(result.Days) > 0 {
			year = result.Days[len(result.Days)-1].Date.Year()
		}
		if year == 0 {
			year = now.Year()
		}
		if len(result.Days) > 0 {
			year = max(result.Days[0].Date.Year(), min(year, result.Days[len(result.Days)-1].Date.Year()))
		}
	} else if !from.IsZero() {
		year = from.Year()
	}
	monthView := s.period < 2
	days := make(map[string]journal.PerformanceDay, len(result.Days))
	limit := 0.0
	for _, day := range result.Days {
		days[day.Date.Format("2006-01-02")] = day
		if day.Date.Year() == year && (!monthView || day.Date.Month() == from.Month()) {
			limit = max(limit, math.Abs(day.PnL))
		}
	}
	overviewCard(c, "heatmap", "").Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			components.Text(c, text.OverviewDailyPnL()).FontSize(13).Bold()
			ui.Spacer(c)
			if s.period == 3 {
				minYear, maxYear := year, year
				if len(result.Days) > 0 {
					minYear, maxYear = result.Days[0].Date.Year(), result.Days[len(result.Days)-1].Date.Year()
				}
				recordIconButton(c, "previous-year", lucide.ChevronLeft(), text.OverviewPreviousYear()).Size(24, 24).Disabled(year <= minYear).
					OnClick(func() { s.heatmapYear, s.selectedDay = year-1, "" })
				components.Text(c, strconv.Itoa(year)).FontSize(12).TextColor(c.Theme().TextMuted)
				recordIconButton(c, "next-year", lucide.ChevronRight(), text.OverviewNextYear()).Size(24, 24).Disabled(year >= maxYear).
					OnClick(func() { s.heatmapYear, s.selectedDay = year+1, "" })
			} else if monthView {
				components.Text(c, text.OverviewMonth(int(from.Month()), strconv.Itoa(year))).FontSize(12).TextColor(c.Theme().TextMuted)
			} else {
				components.Text(c, strconv.Itoa(year)).FontSize(12).TextColor(c.Theme().TextMuted)
			}
		})
		weekdays := []string{text.OverviewMonday(), text.OverviewTuesday(), text.OverviewWednesday(), text.OverviewThursday(), text.OverviewFriday(), text.OverviewSaturday(), text.OverviewSunday()}
		if monthView {
			ui.Grid(c.Key("month-days")).FillWidth().Columns(7).Gap(6).Children(func() {
				for i, label := range weekdays {
					ui.Box(c.Key(i)).Height(20).Center().Children(func() { components.Text(c, label).FontSize(10).TextColor(c.Theme().TextMuted) })
				}
				offset := (int(from.Weekday()) + 6) % 7
				for i := 0; i < offset; i++ {
					ui.Box(c.Key(-i - 1)).Height(46)
				}
				for date := from; date.Month() == from.Month(); date = date.AddDate(0, 0, 1) {
					a.overviewHeatCell(c, date, days[date.Format("2006-01-02")], limit, false)
				}
			})
		} else {
			components.ScrollView(c.Key("year-scroll"), components.ScrollViewOptions{Axes: components.ScrollHorizontal}).FillWidth().Height(150).MinWidth(0).Children(func() {
				ui.Row(c).Gap(3).Children(func() {
					ui.Column(c.Key("weekdays")).Width(24).Gap(3).Children(func() {
						ui.Box(c).Height(16)
						for i, label := range weekdays {
							ui.Box(c.Key(i)).Height(14).Children(func() {
								if i%2 == 0 {
									components.Text(c, label).FontSize(9).TextColor(c.Theme().TextMuted)
								}
							})
						}
					})
					first := time.Date(year, 1, 1, 0, 0, 0, 0, now.Location())
					start := first.AddDate(0, 0, -(int(first.Weekday())+6)%7)
					for week := start; week.Before(first.AddDate(1, 0, 0)); week = week.AddDate(0, 0, 7) {
						ui.Column(c.Key(week.Format("2006-01-02"))).Width(14).Gap(3).Children(func() {
							ui.Box(c).Height(16).Children(func() {
								if week.Day() <= 7 && week.Year() == year {
									components.Text(c, strconv.Itoa(int(week.Month()))).FontSize(9).TextColor(c.Theme().TextMuted)
								}
							})
							for i := 0; i < 7; i++ {
								date := week.AddDate(0, 0, i)
								if date.Year() != year {
									ui.Box(c.Key(i)).Size(14, 14)
									continue
								}
								a.overviewHeatCell(c, date, days[date.Format("2006-01-02")], limit, true)
							}
						})
					}
				})
			})
		}
		a.overviewHeatLegend(c, limit)
		if s.selectedDay != "" {
			ui.Column(c.Key("day-trades")).FillWidth().Gap(8).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					components.Text(c, s.selectedDay).FontSize(12).Bold()
					ui.Spacer(c)
					recordIconButton(c, "close-day", lucide.X(), text.Cancel()).Size(24, 24).OnClick(func() { s.selectedDay = "" })
				})
				count := 0
				components.ScrollView(c.Key("trades-"+s.selectedDay), components.ScrollViewOptions{}).FillWidth().MaxHeight(260).Children(func() {
					ui.Column(c).FillWidth().Gap(4).Children(func() {
						for i := len(result.Trades) - 1; i >= 0; i-- {
							trade := result.Trades[i]
							if trade.ClosedAt.Local().Format("2006-01-02") == s.selectedDay {
								a.overviewTradeRow(c, trade)
								count++
							}
						}
						if count == 0 {
							components.Text(c, text.OverviewNoTrading()).FontSize(12).TextColor(c.Theme().TextMuted)
						}
					})
				})
			})
		}
	})
}

func overviewHeatColor(theme *ui.Theme, day journal.PerformanceDay, limit float64) ui.Color {
	if day.Count == 0 {
		return theme.Text.Alpha(.035)
	}
	if day.PnL == 0 || math.IsNaN(day.PnL) || math.IsInf(day.PnL, 0) || math.IsInf(limit, 0) {
		return theme.TextMuted.Alpha(.23)
	}
	intensity := .2
	if limit > 0 {
		intensity += .55 * math.Sqrt(min(1, math.Abs(day.PnL)/limit))
	}
	return theme.Surface.Mix(replaySummaryPnLColor(theme, day.PnL), float32(intensity))
}

func (a *application) overviewHeatCell(c *ui.Context, date time.Time, day journal.PerformanceDay, limit float64, compact bool) {
	text := a.text()
	key := date.Format("2006-01-02")
	label := text.OverviewDayDetail(key, overviewMoney(day.PnL), day.Count, day.Wins, day.Losses, day.Breakeven)
	if day.Count == 0 {
		label = key + " · " + text.OverviewNoTrading()
	}
	cell := components.Button(c.Key(key), "", components.ButtonGhost).MinWidth(0).Height(46).Padding(6).Radius(6).
		Background(overviewHeatColor(c.Theme(), day, limit)).Label(label).Tooltip(label).Selected(a.overview.selectedDay == key).
		OnClick(func() {
			if a.overview.selectedDay == key {
				a.overview.selectedDay = ""
			} else {
				a.overview.selectedDay = key
			}
		})
	if compact {
		cell.Size(14, 14).Padding(0).Radius(3)
	}
	if a.overview.selectedDay == key {
		cell.Border(1, c.Theme().Text)
	} else if cell.Hovered() || cell.Focused() {
		cell.Border(1, c.Theme().TextMuted)
	}
	if !compact {
		cell.Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(3).Children(func() {
				components.Text(c, strconv.Itoa(date.Day())).FontSize(10).SingleLine()
				if day.Count > 0 {
					components.Text(c, overviewMoney(day.PnL)).FontSize(11).Bold().SingleLine().MinWidth(0).FillWidth()
				}
			})
		})
	}
}

func (a *application) overviewHeatLegend(c *ui.Context, limit float64) {
	text := a.text()
	ui.Row(c.Key("legend")).FillWidth().Wrap().Gap(12).AlignItems(ui.Center).Children(func() {
		for _, item := range []struct {
			label string
			day   journal.PerformanceDay
		}{
			{text.OverviewNoTrading(), journal.PerformanceDay{}},
			{text.OverviewBreakeven(), journal.PerformanceDay{PerformanceSummary: journal.PerformanceSummary{Count: 1}}},
		} {
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(10, 10).Radius(3).Background(overviewHeatColor(c.Theme(), item.day, limit))
				components.Text(c, item.label).FontSize(10).TextColor(c.Theme().TextMuted)
			})
		}
		ui.Spacer(c)
		if limit > 0 {
			components.Text(c, text.OverviewMoney(overviewMoney(-limit))).FontSize(10).TextColor(c.Theme().TextMuted)
			ui.Row(c).Gap(3).Children(func() {
				for _, fraction := range []float64{-1, -.5, -.15, 0, .15, .5, 1} {
					day := journal.PerformanceDay{PerformanceSummary: journal.PerformanceSummary{Count: 1, PnL: limit * fraction}}
					ui.Box(c).Size(10, 10).Radius(3).Background(overviewHeatColor(c.Theme(), day, limit))
				}
			})
			components.Text(c, text.OverviewMoney(overviewMoney(limit))).FontSize(10).TextColor(c.Theme().TextMuted)
		}
	})
}
