package app

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"github.com/oneo-me/onevra/source/shells/sidebar"
	"trajon/internal/market"
	"trajon/internal/market/defaults"
)

type application struct {
	activePage   string
	settingsOpen bool
	candles      []Candle
	chart        klineState

	replaySettings       replaySettings
	replaySettingsDraft  replaySettings
	replaySettingsOpen   bool
	replaySettingsButton ui.Handle

	currentInstrument      instrument
	instrumentPickerOpen   bool
	instrumentPickerQuery  string
	instrumentPickerTab    instrumentTab
	instrumentPickerCursor int
	instrumentPickerList   components.ListState

	window                   *mygo.Window
	marketUpdates            chan market.Snapshot
	dataSources              *marketDataRegistry
	marketLoader             *market.Loader
	marketLoadStarted        bool
	marketToolbarInTitle     bool
	marketLoading            bool
	marketError              error
	pendingCandleLoad        *candleRequest
	pendingPreviousTimeframe string

	olderLoading           bool
	olderError             error
	olderExhausted         bool
	pendingOlderLoad       *candleRequest
	pendingOlderErrorToast bool
}

type navigationItem struct {
	id    string
	label string
	icon  func() *ui.SVG
}

var navigationItems = []navigationItem{
	{id: "overview", label: "概览", icon: lucide.LayoutDashboard},
	{id: "workspace", label: "工作区", icon: lucide.FolderKanban},
	{id: "market", label: "回放", icon: lucide.MonitorPlay},
	{id: "recent", label: "最近访问", icon: lucide.Clock3},
}

func Run() {
	currentInstrument := defaultInstrument()
	app := &application{
		activePage:        "overview",
		currentInstrument: currentInstrument,
		chart:             newKlineState(0),
		replaySettings:    defaultReplaySettings(),
		marketUpdates:     make(chan market.Snapshot, 16),
		dataSources:       defaults.NewRegistry(nil),
	}
	app.marketLoader = market.NewLoader(app.dataSources, app.publishMarketSnapshot)

	mygo.App.WhenReady(func() {
		shell := sidebar.NewShell(sidebar.ShellOptions{
			Title:        "Trajon",
			Width:        1120,
			Height:       760,
			MinWidth:     760,
			MinHeight:    520,
			LeftWidth:    248,
			LeftMinWidth: 200,
			LeftMaxWidth: 340,
			Left:         app.left,
			Content:      app.content,
			ContentTitle: app.contentTitle,
			LeftToolbar:  app.leftToolbar,
		})
		app.window = shell.Window()
		app.window.OnClosed(func() {
			if app.marketLoader != nil {
				app.marketLoader.Close()
			}
			mygo.App.Quit()
		})
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func (a *application) left(c *ui.Context) {
	ui.Column(c).Fill().Padding(sidebar.HeaderHeight(c), 0, 0, 0).Children(func() {
		components.ScrollView(c, components.ScrollViewOptions{}).Fill().Children(func() {
			components.Sidebar(c).Children(func() {
				components.SidebarSection(c, "导航", func() {
					for _, item := range navigationItems {
						if components.SidebarItem(c, item.label, item.icon(), a.activePage == item.id).Clicked() {
							a.activePage = item.id
						}
					}
				})
			})
		})
	})
}

func (a *application) leftToolbar(c *ui.Context) {
	if sidebar.IconButton(c, lucide.Settings(), "设置").Clicked() {
		a.settingsOpen = true
	}
}

func (a *application) contentTitle(c *ui.Context) {
	a.marketToolbarInTitle = a.activePage == "market"
	if a.marketToolbarInTitle {
		a.klineToolbarInTitle(c)
		return
	}
	components.Text(c, a.pageTitle()).Bold().SingleLine()
}

func (a *application) content(c *ui.Context) {
	a.drainMarketUpdates()
	if a.activePage != "market" {
		a.advanceReplayFrame(c)
	}
	ui.Column(c).Fill().Padding(sidebar.HeaderHeight(c), 0, 0, 0).Children(func() {
		if a.activePage == "market" {
			// The chart is a viewport-sized surface, not scrollable page content.
			// Keeping it outside the scroll view gives it the finite content area
			// needed to react to window and sidebar resizes.
			a.marketPage(c)
			return
		}
		components.ScrollView(c, components.ScrollViewOptions{}).Fill().Children(func() {
			ui.Column(c).FillWidth().Padding(40).Gap(12).Children(func() {
				components.Text(c, a.pageTitle()).FontSize(28).Bold()
				components.Text(c, "页面内容将在这里显示。").TextColor(c.Theme().TextMuted).Wrap()
			})
		})
	})

	// The dialog is intentionally an empty surface for now.
	components.Dialog(c, &a.settingsOpen, components.DialogOptions{Label: "设置"}, func(panel ui.Element) {
		panel.Width(440).Height(280).Padding(0).Gap(0)
	})
	a.instrumentPickerDialog(c)
	a.replaySettingsDialog(c)
}

func (a *application) pageTitle() string {
	for _, item := range navigationItems {
		if item.id == a.activePage {
			return item.label
		}
	}
	return "概览"
}
