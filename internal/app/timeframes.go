package app

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/market"
)

func defaultTimeframeFavorites() []string {
	return []string{"15m", "1h", "4h", "1D"}
}

func (a *application) favoriteTimeframes() []string {
	if a.timeframeFavorites == nil {
		return defaultTimeframeFavorites()
	}
	return a.timeframeFavorites
}

func (a *application) toggleTimeframeFavorite(period string) {
	if !slices.Contains(market.CandleIntervals(), period) {
		return
	}
	favorites := slices.Clone(a.favoriteTimeframes())
	if index := slices.Index(favorites, period); index >= 0 {
		favorites = slices.Delete(favorites, index, index+1)
	} else {
		favorites = append(favorites, period)
	}
	a.timeframeFavorites = normalizeTimeframeFavorites(favorites)
	a.saveTimeframeFavorites()
}

func (a *application) saveTimeframeFavorites() {
	if a.timeframeStore != nil {
		a.timeframeSaveError = nil
		a.timeframeStore.Save(a.favoriteTimeframes())
	}
}

func (a *application) publishTimeframeSaveResult(favorites []string, err error) {
	if a.window != nil {
		a.window.Update(func() { a.applyTimeframeSaveResult(favorites, err) })
	}
}

func (a *application) applyTimeframeSaveResult(favorites []string, err error) {
	if slices.Equal(favorites, a.favoriteTimeframes()) {
		a.timeframeSaveError = err
	}
}

func (a *application) closeTimeframePicker() {
	a.timeframePickerOpen = false
	a.timeframeButton.Focus()
}

func (a *application) selectTimeframe(period string) {
	if a.chart.replayLocked() || !slices.Contains(market.CandleIntervals(), period) {
		return
	}
	a.closeTimeframePicker()
	if a.chart.timeframe == period {
		return
	}
	previous := a.chart.timeframe
	a.chart.timeframe = period
	a.chart.hover = -1
	item := a.displayInstrument()
	if a.marketLoader != nil && a.marketLoadStarted && item.SourceID != "" {
		// Retain the committed interval across rapid choices, so a failure rolls
		// back to the series still being displayed rather than a canceled choice.
		if a.pendingPreviousTimeframe == "" {
			a.pendingPreviousTimeframe = previous
		}
		a.loadMarketData(item, period, false)
	}
}

func (a *application) timeframeControl(c *ui.Context) {
	t := c.Theme()
	favorites := a.favoriteTimeframes()
	showCurrent := a.chart.timeframe != "" && !slices.Contains(favorites, a.chart.timeframe)
	triggerWidth := float32(28)
	if showCurrent {
		triggerWidth = 58
	}
	ui.Row(c.Key("timeframes")).Gap(2).AlignItems(ui.Center).MinWidth(triggerWidth).Shrink(1).Children(func() {
		if len(favorites) > 0 {
			// Keep the arrow and drawing tools reachable even when every interval
			// is starred in a narrow window. Only the quick choices scroll.
			components.ScrollView(c.Key("timeframe-favorites"), components.ScrollViewOptions{Axes: components.ScrollHorizontal}).
				Width(float32(len(favorites)*34 - 2)).Height(28).MinWidth(0).Shrink(1).Gap(2).AlignItems(ui.Center).
				Children(func() {
					for _, period := range favorites {
						selected := a.chart.timeframe == period
						button := components.Button(c.Key("timeframe-"+period), period, components.ButtonGhost).
							Size(32, 26).Shrink(0).Padding(0).Radius(t.Radius).Selected(selected).
							Disabled(a.chart.replayLocked()).OnClick(func() { a.selectTimeframe(period) })
						if selected {
							button.Background(t.Text.Alpha(.12)).TextColor(t.Text)
						}
					}
				})
		}
		if a.chart.replayLocked() {
			a.timeframePickerOpen = false
		}
		trigger := components.Button(c.Key("timeframe-picker-trigger"), "", components.ButtonGhost).
			Size(triggerWidth, 28).Shrink(0).Padding(0).Radius(14).Center().
			Label(a.text().Timeframes()).Description(a.chart.timeframe).Tooltip(a.text().Timeframes()).
			Selected(showCurrent || a.timeframePickerOpen).Disabled(a.chart.replayLocked()).
			Expanded(a.timeframePickerOpen).Bind(&a.timeframeButton)
		if showCurrent || a.timeframePickerOpen {
			trigger.Background(t.Text.Alpha(.12)).TextColor(t.Text)
		}
		trigger.Children(func() {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				if showCurrent {
					components.Text(c, a.chart.timeframe).FontSize(12).SingleLine()
				}
				components.Icon(c, lucide.ChevronDown).Size(14, 14)
			})
		})
		opened := false
		if trigger.Clicked() {
			a.timeframePickerOpen = !a.timeframePickerOpen
			opened = a.timeframePickerOpen
			if opened {
				a.timeframeCursor = max(0, slices.Index(market.CandleIntervals(), a.chart.timeframe))
				a.timeframePickerList.ScrollIntoView(a.timeframeCursor)
			}
		}
		popup := components.Popover(c.Key("timeframe-picker"), trigger, &a.timeframePickerOpen, func(panel ui.Element) {
			panel.Width(132).Padding(6).Gap(6).Label(a.text().Timeframes())
			a.timeframeList(c)
			if opened {
				a.timeframePickerList.Focus(c)
			}
			if a.timeframeSaveError != nil {
				ui.Row(c.Key("timeframe-save-error")).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
					components.Text(c, a.text().TimeframeSaveFailed()).FontSize(11).TextColor(t.TextMuted).Wrap().Grow(1).MinWidth(0).
						Tooltip(a.timeframeSaveError.Error())
					components.Button(c.Key("timeframe-save-retry"), a.text().Retry(), components.ButtonGhost).
						Height(26).Padding(2, 4).OnClick(a.saveTimeframeFavorites)
				})
			}
		})
		if popup.Dismissed() && a.timeframePickerList.FocusWithin(c) {
			a.closeTimeframePicker()
		}
	})
}

func (a *application) timeframeList(c *ui.Context) {
	periods := market.CandleIntervals()
	list := &a.timeframePickerList
	list.Selected = &a.timeframeCursor
	list.Key = func(i int) any { return periods[i] }
	list.Label = func(i int) string { return periods[i] }
	list.OnSubmit = func(i int) {
		if i >= 0 && i < len(periods) {
			a.selectTimeframe(periods[i])
		}
	}
	components.List(c.Key("timeframe-list"), list, len(periods), func(i int) {
		period := periods[i]
		t := c.Theme()
		selected := period == a.chart.timeframe
		row := ui.Row(c.Key("timeframe-row-"+period)).FillWidth().Height(32).Padding(0, 16, 0, 0).Gap(2).AlignItems(ui.Center).Radius(t.Radius)
		hovered := row.Hovered()
		highlighted := selected || hovered || i == a.timeframeCursor
		// The list already shades the keyboard cursor's entire row. Other
		// selection/hover backgrounds belong to this row, never its buttons.
		if i != a.timeframeCursor {
			if selected {
				row.Background(t.Text.Alpha(.1))
			} else if hovered {
				row.Background(t.Text.Alpha(.06))
			}
		}
		row.Children(func() {
			choice := components.Button(c.Key("timeframe-choice-"+period), "", components.ButtonGhost).
				Grow(1).MinWidth(0).Height(30).Padding(4, 8).Radius(t.Radius).Justify(ui.Start).
				Label(a.text().SelectTimeframe(period)).Selected(selected).Disabled(a.chart.replayLocked()).
				OnClick(func() { a.selectTimeframe(period) }).Children(func() {
				components.Text(c, period).SingleLine()
			})
			if highlighted {
				choice.Background(ui.Transparent).TextColor(t.Text)
			}
			starred := slices.Contains(a.favoriteTimeframes(), period)
			label := a.text().FavoriteTimeframe(period)
			if starred {
				label = a.text().UnfavoriteTimeframe(period)
			}
			star := klineActionIconButton(c.Key("timeframe-star-"+period), lucide.Star, label, starred).
				Background(ui.Transparent).OnClick(func() { a.toggleTimeframeFavorite(period) })
			if starred {
				star.TextColor(t.Accent)
			} else {
				star.TextColor(t.TextMuted)
			}
		})
	}).FillWidth().Height(320).Label(a.text().Timeframes())
}
