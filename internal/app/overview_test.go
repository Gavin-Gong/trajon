package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/theme"
	"trajon/internal/journal"
	"trajon/internal/locales"
)

func TestOverviewHeatmapSelectionPreservesDraftAndOpensExistingRecord(t *testing.T) {
	now := time.Now()
	app := &application{activePage: "overview", language: locales.Chinese, records: recordsState{book: journal.New(now), ready: true}}
	seedClosedRecord(t, app)
	closed := app.records.book.Active()
	closed.Instrument = "交易记录"
	app.newRecord()
	draft := app.records.book.Active()
	draft.Draft.Text = "尚未提交的计划"
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); app.content(c) }, 980, 1150)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.SetLocale("zh-Hans")
	for range 4 {
		tt.Frame()
	}
	saveRecordEvidence(t, tt, "overview-month")
	if app.overview.result == nil || app.overview.result.PnL != 100 {
		t.Fatal("overview did not derive journal P&L")
	}
	day := app.overview.result.Days[0]
	label := app.text().OverviewDayDetail(day.Date.Format("2006-01-02"), overviewMoney(day.PnL), day.Count, day.Wins, day.Losses, day.Breakeven)
	clickReplaySetting(t, tt, label)
	if app.overview.selectedDay == "" || app.records.book.Active() != draft || draft.Draft.Text != "尚未提交的计划" {
		t.Fatal("heatmap drilldown changed the active journal or its draft")
	}
	clickReplaySetting(t, tt, app.text().JournalOpenRecord(closed.Instrument))
	if app.activePage != "journal" || app.records.book.Active() != closed || !app.records.review || draft.Draft.Text != "尚未提交的计划" || len(app.records.book.Records) != 2 {
		t.Fatal("overview navigation replaced records/drafts or bypassed historical review selection")
	}
	app.activePage = "overview"
	app.overview.period = 3
	for range 3 {
		tt.Frame()
	}
	saveRecordEvidence(t, tt, "overview-year")
	tt.SetSize(540, 1150)
	for range 4 {
		tt.Frame()
	}
	saveRecordEvidence(t, tt, "overview-narrow")
	app.applyLocale("en")
	tt.SetLocale("en")
	tt.Frame()
	if draft.Draft.Text != "尚未提交的计划" {
		t.Fatal("locale/layout switch lost the draft")
	}
}

func TestOverviewRejectsResultsAfterEditsFiltersAndClosure(t *testing.T) {
	app := &application{}
	app.overview.generation = 1
	request := app.overviewRequest(time.Now())
	result := journal.Performance{PerformanceSummary: journal.PerformanceSummary{Count: 1}}
	app.records.revision++
	app.applyOverview(1, request, result)
	if app.overview.result != nil {
		t.Fatal("accepted statistics from before a journal edit")
	}
	request = app.overviewRequest(time.Now())
	app.overview.instrument = "another instrument"
	app.applyOverview(1, request, result)
	if app.overview.result != nil {
		t.Fatal("accepted a replaced filter")
	}
	request = app.overviewRequest(time.Now())
	app.applyOverview(0, request, result)
	if app.overview.result != nil {
		t.Fatal("accepted a superseded generation")
	}
	app.applyOverview(1, request, result)
	if app.overview.result == nil || app.overview.result.Count != 1 {
		t.Fatal("rejected the current result")
	}
	app.closeOverview()
	app.overview.result = nil
	app.applyOverview(1, app.overviewRequest(time.Now()), result)
	if app.overview.result != nil {
		t.Fatal("accepted statistics after closure")
	}
}
