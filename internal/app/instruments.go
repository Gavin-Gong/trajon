package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/market"
)

type instrumentTab uint8

const (
	instrumentTabAll instrumentTab = iota
	instrumentTabSpot
	instrumentTabFutures
)

var builtinInstrumentCatalog = market.InstrumentCatalog()

func defaultInstrument() instrument {
	return market.DefaultInstrument()
}

func (a *application) openInstrumentPicker() {
	a.stopInstrumentSearch()
	a.instrumentSearch.items = nil
	a.instrumentSearch.err = nil
	a.instrumentPickerOpen = true
	a.instrumentPickerQuery = ""
	a.instrumentPickerTab = instrumentTabAll
	a.instrumentPickerCursor = 0
	for i, item := range builtinInstrumentCatalog {
		if item.Identity() == a.currentInstrument.Identity() {
			a.instrumentPickerCursor = i
			break
		}
	}
	a.instrumentPickerList = components.ListState{}
}

func (a *application) filteredInstruments() []instrument {
	marketType := market.MarketType("")
	if a.instrumentPickerTab == instrumentTabSpot {
		marketType = market.MarketSpot
	} else if a.instrumentPickerTab == instrumentTabFutures {
		marketType = market.MarketFutures
	}
	items := append([]instrument(nil), builtinInstrumentCatalog...)
	if strings.TrimSpace(a.instrumentPickerQuery) != "" {
		seen := make(map[string]bool, len(items))
		for _, item := range items {
			seen[item.Identity()] = true
		}
		for _, item := range a.instrumentSearch.items {
			if !seen[item.Identity()] {
				items = append(items, item)
				seen[item.Identity()] = true
			}
		}
	}
	return market.FilterInstruments(items, a.instrumentPickerQuery, marketType)
}

func (a *application) instrumentPickerDialog(c *ui.Context) {
	a.drainInstrumentSearchUpdates()
	components.Dialog(c, &a.instrumentPickerOpen, components.DialogOptions{Label: "选择标的"}, func(panel ui.Element) {
		panel.Width(460).Height(520).Padding(18).Gap(12)

		t := c.Theme()
		ui.Row(c).FillWidth().Height(28).AlignItems(ui.Center).Children(func() {
			components.Text(c, "选择标的").FontSize(18).Bold().SingleLine()
			ui.Spacer(c)
			close := components.Button(c.Key("instrument-picker-close"), "", components.ButtonGhost).
				Size(28, 28).Padding(0).Radius(14).Center().Label("关闭标的选择")
			close.Children(func() {
				components.Icon(c, lucide.X).Size(15, 15)
			})
			if close.Clicked() {
				a.instrumentPickerOpen = false
			}
		})

		ui.SearchField(c.Key("instrument-picker-search"), &a.instrumentPickerQuery).
			FillWidth().Label("搜索标的").Placeholder("搜索 symbol").AutoFocus().
			OnChange(a.searchInstruments)

		a.instrumentHistoryChoices(c)

		ui.Row(c).FillWidth().Gap(4).Children(func() {
			a.instrumentTabButton(c, "全部", instrumentTabAll)
			a.instrumentTabButton(c, "现货", instrumentTabSpot)
			a.instrumentTabButton(c, "合约", instrumentTabFutures)
		})

		ui.Row(c).FillWidth().Padding(8, 10, 4, 10).Gap(8).TextColor(t.TextMuted).Children(func() {
			components.Text(c, "名称").FontSize(11).SingleLine()
			ui.Spacer(c)
			components.Text(c, "symbol").FontSize(11).SingleLine()
			components.Text(c, "市场").FontSize(11).Width(94).SingleLine()
			ui.Box(c).Size(15, 1)
		})

		items := a.filteredInstruments()
		if a.instrumentSearch.loading && len(items) > 0 {
			components.Text(c, "搜索中…").FontSize(11).TextColor(t.TextMuted)
		} else if a.instrumentSearch.err != nil {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				label := "部分市场搜索失败"
				if len(items) == 0 {
					label = "搜索失败"
				}
				components.Text(c, label).FontSize(11).TextColor(t.TextMuted).
					Tooltip(a.instrumentSearch.err.Error())
				ui.Spacer(c)
				components.Button(c.Key("instrument-search-retry"), "重试", components.ButtonGhost).
					Height(24).OnClick(a.searchInstruments)
			})
		}

		if a.instrumentPickerCursor >= len(items) {
			a.instrumentPickerCursor = len(items) - 1
		}
		if len(items) == 0 {
			a.instrumentPickerCursor = -1
		}
		a.instrumentPickerList.Selected = &a.instrumentPickerCursor
		a.instrumentPickerList.Key = func(i int) any { return items[i].Identity() }
		a.instrumentPickerList.Label = func(i int) string { return items[i].Name + " " + items[i].Symbol + " " + items[i].TypeLabel() }
		a.instrumentPickerList.OnSubmit = func(i int) {
			if i >= 0 && i < len(items) {
				a.selectInstrument(items[i])
			}
		}

		if len(items) == 0 {
			ui.Box(c).Fill().Center().Children(func() {
				label := "未找到标的"
				if a.instrumentSearch.loading {
					label = "搜索中…"
				} else if a.instrumentSearch.err != nil {
					label = "无法完成搜索"
				}
				components.Text(c, label).TextColor(t.TextMuted)
			})
			return
		}

		components.List(c.Key("instrument-picker-list"), &a.instrumentPickerList, len(items), func(i int) {
			item := items[i]
			row := components.Button(c.Key("instrument-row-"+item.Identity()), "", components.ButtonGhost).
				FillWidth().Height(42).MinWidth(0).Padding(8, 10).Radius(t.Radius).Label(item.Name + " " + item.Symbol + " " + item.TypeLabel())
			if i == a.instrumentPickerCursor {
				row.Background(t.Text.Alpha(.06))
			}
			if item.Identity() == a.currentInstrument.Identity() {
				row.Background(t.Text.Alpha(.12))
			}
			row.Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					components.Text(c, item.Name).SingleLine()
					ui.Spacer(c)
					components.Text(c, item.Symbol).TextColor(t.TextMuted).SingleLine()
					components.Text(c, item.TypeLabel()).TextColor(t.TextMuted).Width(94).SingleLine()
					if item.Identity() == a.currentInstrument.Identity() {
						components.Icon(c, lucide.Check).Size(15, 15).TextColor(t.Accent)
					} else {
						ui.Box(c).Size(15, 15)
					}
				})
			})
			row.OnClick(func() {
				a.selectInstrument(item)
			})
		}).Grow(1).MinHeight(0)
	})
	if !a.instrumentPickerOpen && a.instrumentSearch.cancel != nil {
		a.stopInstrumentSearch()
	}
}

func (a *application) instrumentTabButton(c *ui.Context, label string, tab instrumentTab) {
	t := c.Theme()
	button := components.Button(c.Key("instrument-tab-"+label), label, components.ButtonGhost).
		Grow(1).Height(30).Padding(0, 10).Radius(t.Radius).Selected(a.instrumentPickerTab == tab)
	if a.instrumentPickerTab == tab {
		button.Background(t.Text.Alpha(.1)).TextColor(t.Text)
	}
	if button.Clicked() {
		a.instrumentPickerTab = tab
		a.instrumentPickerCursor = 0
	}
}
