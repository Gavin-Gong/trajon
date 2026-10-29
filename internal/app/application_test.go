package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/shells/sidebar"
	"trajon/internal/journal"
	"trajon/internal/locales"
)

func TestApplicationSidebarNavigation(t *testing.T) {
	for _, language := range []locales.Language{locales.English, locales.Chinese} {
		t.Run(string(language), func(t *testing.T) {
			app := &application{activePage: "overview", language: language}
			tt := ui.NewTester(app.left, 180, 400)
			for _, label := range []string{"Workspace", "Recent", "工作区", "最近访问"} {
				if _, ok := tt.Find(label); ok {
					t.Errorf("removed navigation item %q remains visible", label)
				}
			}
			for _, page := range []struct {
				id, label string
			}{
				{"market", app.text().Replay()},
				{"journal", app.text().JournalTitle()},
				{"overview", app.text().Overview()},
			} {
				clickReplaySetting(t, tt, page.label)
				if app.activePage != page.id || app.pageTitle() != page.label {
					t.Fatalf("navigation to %q left page=%q, title=%q", page.id, app.activePage, app.pageTitle())
				}
			}
		})
	}
}

func TestJournalPageNavigationKeepsDraftAndDoesNotReplaceOverview(t *testing.T) {
	for _, titleFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("title-first=%v", titleFirst), func(t *testing.T) {
			app, tt := newApplicationShellOrderTester(locales.Chinese, titleFirst)
			app.records = recordsState{book: journal.New(time.Now()), ready: true}
			r := app.records.book.Active()
			r.Instrument, r.Draft.Text = "手动标的", "未提交的计划"
			app.syncRecordKind()
			tt.Frame()
			if _, ok := tt.Find(app.text().JournalSearchRecords()); ok {
				t.Fatal("journal list was embedded in Overview")
			}
			if !tt.HasText(app.text().OverviewTotalPnL()) {
				t.Fatal("Overview performance dashboard was replaced by the journal")
			}

			clickReplaySetting(t, tt, app.text().JournalTitle())
			app.records.composer.Focus()
			tt.Frame()
			tt.Type("!")
			clickReplaySetting(t, tt, app.text().JournalSearchRecords())
			tt.Type("手动")
			width := float32(280)
			app.recordsBrowser.width = width

			clickReplaySetting(t, tt, app.text().Overview())
			for range 3 {
				tt.Frame()
			}
			if _, ok := tt.Find(app.text().JournalSearchRecords()); ok {
				t.Fatal("journal controls remained mounted after leaving its page")
			}
			clickReplaySetting(t, tt, app.text().Replay())
			tt.Frame()
			clickReplaySetting(t, tt, app.text().JournalTitle())
			for range 3 {
				tt.Frame()
			}
			if app.activePage != "journal" || app.records.book.Active() != r ||
				r.Draft.Text != "未提交的计划!" || app.recordsBrowser.query != "手动" || app.recordsBrowser.width != width {
				t.Fatal("page transitions lost journal selection, draft or list state")
			}
			if _, ok := tt.Find(app.text().JournalSearchRecords()); !ok {
				t.Fatal("returning to the journal page did not restore the list")
			}
		})
	}
}

// Shell builds its sidebar and content before its overlaid ContentTitle. Keep
// that order as the default here: title-first chart tests cannot catch this bug.
func newApplicationShellOrderTester(language locales.Language, titleFirst bool) (*application, *ui.Tester) {
	candles := demoCandles()
	app := &application{
		activePage: "overview", language: language,
		currentInstrument: defaultInstrument(), candles: candles,
		chart: newKlineState(len(candles)), replaySettings: defaultReplaySettings(),
		marketLoadStarted: true, marketToolbarInTitle: true,
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			header := func() {
				ui.Row(c.Key("header")).Absolute().Left(181).Top(0).Right(8).
					Height(sidebar.HeaderHeight(c)).AlignItems(ui.Center).
					Children(func() { app.contentTitle(c) })
			}
			if titleFirst {
				header()
			}
			ui.Row(c.Key("body")).Fill().Children(func() {
				ui.Column(c.Key("sidebar")).Width(165).FillHeight().Children(func() {
					app.left(c)
				})
				ui.Column(c.Key("content")).Grow(1).MinWidth(0).FillHeight().Children(func() {
					app.content(c)
				})
			})
			if !titleFirst {
				header()
			}
		})
	}, 1120, 760)
	return app, tt
}

func TestApplicationEnteringReplayWithContentBuiltBeforeTitle(t *testing.T) {
	for _, language := range []locales.Language{locales.English, locales.Chinese} {
		for _, titleFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/title-first=%v", language, titleFirst), func(t *testing.T) {
				app, tt := newApplicationShellOrderTester(language, titleFirst)
				for range 3 {
					clickReplaySetting(t, tt, app.text().Replay())
					for range 3 {
						tt.Frame()
					}
					if app.activePage != "market" || !app.marketToolbarInTitle {
						t.Fatal("navigation did not enter the Replay title-toolbar layout")
					}
					labels, toolbarCount := tt.Texts(), 0
					for _, label := range labels {
						if label == app.text().ChartToolbar() {
							toolbarCount++
						}
					}
					if toolbarCount != 1 {
						t.Fatalf("Replay built %d chart toolbars, want one: %v", toolbarCount, labels)
					}
					toolbar, ok := tt.Find(app.text().ChartToolbar())
					chart, chartOK := tt.Find(app.text().CandleChart(app.currentInstrument.Symbol))
					if !ok || !chartOK || toolbar.Y+toolbar.H > chart.Y {
						t.Fatalf("toolbar did not stay in the header: toolbar=%+v, chart=%+v", toolbar, chart)
					}
					textTool, textOK := tt.Find(app.text().Text())
					indicators, indicatorsOK := tt.Find(app.text().Indicators())
					if !textOK || !indicatorsOK || indicators.X <= textTool.X || indicators.X-(textTool.X+textTool.W) > 20 {
						t.Fatalf("header indicators should follow drawing tools: text=%+v, indicators=%+v", textTool, indicators)
					}
					if _, ok := tt.Find(app.text().DeleteAnnotation()); ok {
						t.Fatal("header should not render the delete annotation button")
					}
					clickReplaySetting(t, tt, app.text().Indicators())
					if !app.indicatorsOpen {
						t.Fatal("header indicator button did not open its dialog")
					}
					tt.Key(0, ui.KeyEscape)
					for range 3 {
						tt.Frame()
					}
					if app.indicatorsOpen || !tt.Focused(app.text().Indicators()) {
						t.Fatal("dialog dismissal did not restore header button focus")
					}
					clickReplaySetting(t, tt, app.text().Overview())
					for range 3 {
						tt.Frame()
					}
					if app.activePage != "overview" {
						t.Fatal("navigation did not leave Replay")
					}
					if _, ok := tt.Find(app.text().ChartToolbar()); ok {
						t.Fatal("chart toolbar remained on Overview")
					}
				}
			})
		}
	}
}
