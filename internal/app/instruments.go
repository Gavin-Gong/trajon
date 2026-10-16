package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/market"
	appui "trajon/internal/ui"
)

type instrumentTab uint8

const (
	instrumentTabAll instrumentTab = iota
	instrumentTabSpot
	instrumentTabFutures
	instrumentTabStocks
)

var builtinInstrumentCatalog = market.InstrumentCatalog()

func (a *application) catalogInstruments() []instrument {
	return a.dataSources.DefaultInstruments()
}

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
	for i, item := range a.catalogInstruments() {
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
	} else if a.instrumentPickerTab == instrumentTabStocks {
		marketType = market.MarketStock
	}
	items := market.FilterInstruments(a.catalogInstruments(), a.instrumentPickerQuery, marketType)
	if strings.TrimSpace(a.instrumentPickerQuery) != "" {
		indexes := make(map[string]int, len(items))
		for index, item := range items {
			indexes[item.Identity()] = index
		}
		// Search sources already own query matching (including normalized
		// codes and aliases). Only apply the selected market tab here.
		for _, item := range market.FilterInstruments(a.instrumentSearch.items, "", marketType) {
			if index, exists := indexes[item.Identity()]; exists {
				items[index] = item
			} else {
				indexes[item.Identity()] = len(items)
				items = append(items, item)
			}
		}
	}
	return items
}

func (a *application) instrumentPickerDialog(c *ui.Context) {
	a.drainInstrumentSearchUpdates()
	components.Dialog(c.Key("instrument-picker-dialog"), &a.instrumentPickerOpen, components.DialogOptions{
		Label:              a.text().ChooseInstrument(),
		DisableEscapeClose: a.instrumentPickerQuery != "" && a.instrumentSearchHandle.Focused(c),
	}, func(panel ui.Element) {
		panel.Width(600).Height(520).Padding(18).Gap(12)

		t := c.Theme()
		ui.Row(c).FillWidth().Height(28).AlignItems(ui.Center).Children(func() {
			components.Text(c, a.text().ChooseInstrument()).FontSize(18).Bold().SingleLine()
			ui.Spacer(c)
			close := components.Button(c.Key("instrument-picker-close"), "", components.ButtonGhost).
				Size(28, 28).Padding(0).Radius(14).Center().Label(a.text().CloseInstrumentPicker())
			close.Children(func() {
				components.Icon(c, lucide.X).Size(15, 15)
			})
			if close.Clicked() {
				a.instrumentPickerOpen = false
			}
		})

		a.instrumentSearchField(c)

		a.instrumentHistoryChoices(c)

		ui.Row(c).FillWidth().Gap(4).Children(func() {
			a.instrumentTabButton(c, a.text().All(), instrumentTabAll)
			a.instrumentTabButton(c, a.text().Spot(), instrumentTabSpot)
			a.instrumentTabButton(c, a.text().Futures(), instrumentTabFutures)
			a.instrumentTabButton(c, a.text().Stocks(), instrumentTabStocks)
		})
		if a.instrumentPickerTab == instrumentTabStocks {
			components.Text(c, a.text().StockSymbolHint()).FontSize(11).TextColor(t.TextMuted).Wrap()
			if a.longbridge.client != nil && !a.longbridge.status.Authorized {
				ui.Row(c.Key("stocks-authorization")).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					components.Text(c, a.text().Longbridge()+" · "+a.text().LongbridgeNotAuthorized()).FontSize(12).TextColor(t.TextMuted)
					ui.Spacer(c)
					components.Button(c.Key("stocks-connect"), a.text().ConnectDataSource(), components.ButtonSecondary).
						OnClick(a.openDataSourceSettingsFromPicker)
				})
			}
		}

		ui.Row(c).FillWidth().Padding(8, 10, 4, 10).Gap(8).TextColor(t.TextMuted).Children(func() {
			components.Text(c, a.text().Name()).FontSize(11).SingleLine()
			ui.Spacer(c)
			components.Text(c, a.text().Symbol()).FontSize(11).SingleLine().Width(100)
			components.Text(c, a.text().Market()).FontSize(11).Width(164).SingleLine()
			ui.Box(c).Size(15, 1)
		})

		items := a.filteredInstruments()
		if a.instrumentSearch.loading && len(items) > 0 {
			components.Text(c, a.text().Searching()).FontSize(11).TextColor(t.TextMuted)
		} else if a.instrumentSearch.err != nil {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				label := a.text().PartialSearchFailed()
				if len(items) == 0 {
					label = a.text().SearchFailed()
				}
				components.Text(c, label).FontSize(11).TextColor(t.TextMuted).
					Tooltip(appui.MarketError(a.text(), a.instrumentSearch.err))
				ui.Spacer(c)
				components.Button(c.Key("instrument-search-retry"), a.text().Retry(), components.ButtonGhost).
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
		a.instrumentPickerList.Label = func(i int) string {
			return a.instrumentName(items[i]) + " " + items[i].Symbol + " " + a.instrumentTypeLabel(items[i])
		}
		a.instrumentPickerList.OnSubmit = func(i int) {
			if i >= 0 && i < len(items) {
				a.selectInstrument(items[i])
			}
		}

		if len(items) == 0 {
			ui.Box(c).Fill().Center().Children(func() {
				label := a.text().NoInstrument()
				if a.instrumentSearch.loading {
					label = a.text().Searching()
				} else if a.instrumentSearch.err != nil {
					label = a.text().SearchUnavailable()
				}
				components.Text(c, label).TextColor(t.TextMuted)
			})
			return
		}

		components.List(c.Key("instrument-picker-list"), &a.instrumentPickerList, len(items), func(i int) {
			item := items[i]
			row := components.Button(c.Key("instrument-row-"+item.Identity()), "", components.ButtonGhost).
				FillWidth().Height(42).MinWidth(0).Padding(8, 10).Radius(t.Radius).Label(a.instrumentName(item) + " " + item.Symbol + " " + a.instrumentTypeLabel(item))
			if i == a.instrumentPickerCursor {
				row.Background(t.Text.Alpha(.06))
			}
			if item.Identity() == a.currentInstrument.Identity() {
				row.Background(t.Text.Alpha(.12))
			}
			row.Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					components.Text(c, a.instrumentName(item)).SingleLine().Grow(1).MinWidth(0)
					ui.Spacer(c)
					components.Text(c, item.Symbol).TextColor(t.TextMuted).SingleLine().Width(100)
					components.Text(c, a.instrumentTypeLabel(item)).TextColor(t.TextMuted).Width(164).SingleLine()
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
	button := components.Button(c.Key(fmt.Sprintf("instrument-tab-%d", tab)), label, components.ButtonGhost).
		Grow(1).Height(30).Padding(0, 10).Radius(t.Radius).Selected(a.instrumentPickerTab == tab)
	if a.instrumentPickerTab == tab {
		button.Background(t.Text.Alpha(.1)).TextColor(t.Text)
	}
	if button.Clicked() {
		a.instrumentPickerTab = tab
		a.instrumentPickerCursor = 0
	}
}
