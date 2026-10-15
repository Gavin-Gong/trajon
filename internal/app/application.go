package app

import (
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"github.com/oneo-me/onevra/source/shells/sidebar"
	"github.com/oneo-me/onevra/source/theme"
	"trajon/internal/locales"
	"trajon/internal/market"
	"trajon/internal/market/defaults"
	appui "trajon/internal/ui"
)

type application struct {
	activePage         string
	settingsOpen       bool
	settingsButton     ui.Handle
	settingsLanguage   ui.Handle
	informationOpen    bool
	informationButton  ui.Handle
	updates            updateState
	language           locales.Language
	languagePreference locales.Preference
	preferredLanguages []string // Standalone views only; native windows use MyGo.
	changeLocale       func(string)
	transientLocale    bool // MYGO_LOCALE must not overwrite saved preferences.
	localeStore        *localePreferenceStore
	languageSaveError  error
	candles            []Candle
	chart              klineState

	indicatorsOpen    bool
	indicatorsButton  ui.Handle
	indicatorSelected int
	indicatorDraft    indicatorDraft
	indicatorCache    indicatorCache

	replaySettings       replaySettings
	replaySettingsDraft  replaySettings
	replaySettingsOpen   bool
	replaySettingsButton ui.Handle
	replayEndButton      ui.Handle

	currentInstrument      instrument
	instrumentPickerOpen   bool
	instrumentPickerQuery  string
	instrumentPickerTab    instrumentTab
	instrumentPickerCursor int
	instrumentPickerList   components.ListState
	instrumentSearchHandle ui.Handle
	instrumentSearch       instrumentSearchState
	instrumentHistory      []instrument
	instrumentHistoryStore *instrumentHistoryStore
	pendingHistoryIdentity string

	shell                    *sidebar.Shell
	window                   *mygo.Window
	marketUpdates            chan market.Snapshot
	dataSources              *marketDataRegistry
	marketLoader             *market.Loader
	marketLoadStarted        bool
	marketToolbarInTitle     bool // Layout configuration, never derived during a build.
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
	label func(locales.Messages) string
	icon  func() *ui.SVG
}

var navigationItems = []navigationItem{
	{id: "overview", label: func(t locales.Messages) string { return t.Overview() }, icon: lucide.LayoutDashboard},
	{id: "market", label: func(t locales.Messages) string { return t.Replay() }, icon: lucide.MonitorPlay},
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
		// Shell owns the toolbar's title host, even while another page is active.
		marketToolbarInTitle: true,
		updates: updateState{
			check:    checkAppUpdate,
			relaunch: mygo.App.Relaunch,
		},
	}
	app.marketLoader = market.NewLoader(app.dataSources, app.publishMarketSnapshot)
	if path, err := instrumentHistoryPath(); err != nil {
		log.Printf("instrument history path: %v", err)
	} else {
		app.instrumentHistory, err = loadInstrumentHistory(path)
		if err != nil {
			log.Printf("load instrument history: %v", err)
		}
		app.instrumentHistoryStore = newInstrumentHistoryStore(path)
		defer func() {
			if err := app.instrumentHistoryStore.Close(); err != nil {
				log.Printf("close instrument history: %v", err)
			}
		}()
	}

	app.languagePreference = locales.System
	app.transientLocale = i18n.Normalize(os.Getenv("MYGO_LOCALE")) != ""
	if path, err := localePreferencePath(); err != nil {
		log.Printf("language preference path: %v", err)
		app.languageSaveError = err
	} else {
		app.languagePreference, err = loadLocalePreference(path)
		if err != nil {
			log.Printf("load language preference: %v", err)
		}
		app.localeStore = newLocalePreferenceStore(path, app.publishLocaleSaveResult)
		defer func() {
			if err := app.localeStore.Close(); err != nil {
				log.Printf("close language preference: %v", err)
			}
		}()
	}
	app.changeLocale = mygo.App.SetLocale
	stopLocaleChanges := mygo.App.OnLocaleChanged(app.applyLocale)
	defer stopLocaleChanges()
	// Before readiness, MYGO_LOCALE (mygo dev -lang) takes precedence.
	app.changeLocale(app.languagePreference.Locale())

	mygo.App.WhenReady(func() {
		app.applyLocale(mygo.App.Locale())
		shell := sidebar.NewShell(sidebar.ShellOptions{
			Title:        applicationName,
			Width:        1120,
			Height:       760,
			MinWidth:     760,
			MinHeight:    520,
			LeftWidth:    165,
			LeftMinWidth: 133,
			LeftMaxWidth: 227,
			Left:         app.left,
			Content:      app.content,
			ContentTitle: app.contentTitle,
			LeftToolbar:  app.leftToolbar,
			Theme:        theme.Onevra,
		})
		app.shell = shell
		shell.SetLabels(app.shellLabels())
		app.window = shell.Window()
		app.checkForUpdates()
		app.window.OnClosed(func() {
			stopLocaleChanges()
			app.closeUpdates()
			app.closeIndicatorCalculations()
			app.stopInstrumentSearch()
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
				components.SidebarSection(c, a.text().Navigation(), func() {
					for _, item := range navigationItems {
						if appui.NavigationItem(c.Key(item.id), item.label(a.text()), item.icon(), a.activePage == item.id).Clicked() {
							a.activePage = item.id
						}
					}
				})
			})
		})
	})
}

func (a *application) leftToolbar(c *ui.Context) {
	if sidebar.IconButton(c.Key("settings"), lucide.Settings(), a.text().Settings()).Bind(&a.settingsButton).Clicked() {
		a.settingsOpen = true
	}
	a.informationToolbarButton(c)
}

func (a *application) contentTitle(c *ui.Context) {
	if a.activePage == "market" {
		a.klineToolbarInTitle(c)
		return
	}
	components.Text(c, a.pageTitle()).Bold().SingleLine()
}

func (a *application) content(c *ui.Context) {
	components.WithLocalization(c, a.componentLocalization(), func() { a.localizedContent(c) })
}

func (a *application) localizedContent(c *ui.Context) {
	a.drainUpdateResults()
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
				components.Text(c, a.text().PagePlaceholder()).TextColor(c.Theme().TextMuted).Wrap()
			})
		})
	})

	a.settingsDialog(c)
	a.informationDialog(c)
	a.instrumentPickerDialog(c)
	a.indicatorsDialog(c)
	a.replaySettingsDialog(c)
	a.replaySummaryDialog(c)
}

func (a *application) pageTitle() string {
	for _, item := range navigationItems {
		if item.id == a.activePage {
			return item.label(a.text())
		}
	}
	return a.text().Overview()
}
