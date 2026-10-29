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
	activePage                    string
	settingsOpen                  bool
	settingsButton                ui.Handle
	settingsLanguage              ui.Handle
	settingsReturnToPicker        bool
	retryMarketAfterAuthorization string
	informationOpen               bool
	informationButton             ui.Handle
	recordsButton                 ui.Handle
	recordsPanel                  *mygo.Window
	records                       recordsState
	recordsBrowser                recordsBrowserState
	overview                      overviewState
	updates                       updateState
	longbridge                    longbridgeState
	language                      locales.Language
	languagePreference            locales.Preference
	preferredLanguages            []string // Standalone views only; native windows use MyGo.
	changeLocale                  func(string)
	transientLocale               bool // MYGO_LOCALE must not overwrite saved preferences.
	localeStore                   *localePreferenceStore
	languageSaveError             error
	candles                       []Candle
	chart                         klineState

	timeframePickerOpen bool
	timeframeButton     ui.Handle
	timeframePickerList components.ListState
	timeframeCursor     int
	timeframeFavorites  []string // nil uses defaults; an empty slice means no favorites.
	timeframeStore      *timeframeFavoritesStore
	timeframeSaveError  error

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
	{id: "journal", label: func(t locales.Messages) string { return t.JournalTitle() }, icon: lucide.NotebookText},
	{id: "market", label: func(t locales.Messages) string { return t.Replay() }, icon: lucide.MonitorPlay},
}

func Run() {
	currentInstrument := defaultInstrument()
	authorization := newLongbridgeClient()
	app := &application{
		activePage:        "overview",
		currentInstrument: currentInstrument,
		chart:             newKlineState(0),
		replaySettings:    defaultReplaySettings(),
		marketUpdates:     make(chan market.Snapshot, 16),
		dataSources:       defaults.NewRegistry(nil, authorization),
		longbridge:        longbridgeState{client: authorization},
		// Shell owns the toolbar's title host, even while another page is active.
		marketToolbarInTitle: true,
		updates: updateState{
			check:    checkAppUpdate,
			relaunch: mygo.App.Relaunch,
		},
	}
	defer app.closeRecordsStore()
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

	if path, err := timeframeFavoritesPath(); err != nil {
		log.Printf("timeframe favorites path: %v", err)
		app.timeframeSaveError = err
	} else {
		app.timeframeFavorites, err = loadTimeframeFavorites(path)
		if err != nil {
			log.Printf("load timeframe favorites: %v", err)
		}
		app.timeframeStore = newTimeframeFavoritesStore(path, app.publishTimeframeSaveResult)
		defer func() {
			if err := app.timeframeStore.Close(); err != nil {
				log.Printf("close timeframe favorites: %v", err)
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
		app.window = shell.Window()
		app.loadRecords()
		app.checkForUpdates()
		app.restoreLongbridge()
		app.window.OnClosed(func() {
			app.destroyRecordsPanel()
			stopLocaleChanges()
			app.closeLongbridge()
			app.closeUpdates()
			app.closeIndicatorCalculations()
			app.closeOverview()
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
							a.timeframePickerOpen = false
							if a.activePage != item.id && !a.records.detached {
								a.cancelRecordWork()
							}
							a.activePage = item.id
							if item.id == "journal" && a.records.ready && !a.records.detached {
								a.loadRecordImages()
							}
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
		a.restoreLongbridge()
	}
	a.informationToolbarButton(c)
}

func (a *application) contentTitle(c *ui.Context) {
	if a.activePage == "market" {
		a.klineToolbarInTitle(c)
		return
	}
	ui.Row(c.Key("page-title")).Fill().AlignItems(ui.Center).Children(func() {
		components.Text(c, a.pageTitle()).Bold().SingleLine()
		ui.Spacer(c)
		sidebar.IconButton(c.Key("records"), lucide.NotebookPen(), a.text().JournalQuickRecord()).
			Bind(&a.recordsButton).OnClick(a.showRecordsPanel)
	})
}

func (a *application) content(c *ui.Context) {
	a.drainLongbridgeResults()
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
		if a.activePage == "journal" {
			a.recordsPage(c)
			return
		}
		a.overviewPage(c)
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
