package app

import (
	"errors"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/theme"
	"trajon/internal/journal"
	"trajon/internal/locales"
)

func TestJournalPageSelectionAndNarrowNavigationRetainEditors(t *testing.T) {
	app := &application{language: locales.Chinese, records: recordsState{book: journal.New(time.Now()), ready: true}}
	seedClosedRecord(t, app)
	closed := app.records.book.Active()
	closed.Instrument, closed.Summary, closed.Tags = "标的 B", "按计划止损，没有继续扛单。", []string{"纪律", "突破"}
	app.newRecord()
	holding := app.records.book.Active()
	holding.Instrument = "标的 A"
	for _, draft := range []journal.Draft{{Kind: journal.Plan, Text: "等待回踩确认"}, {Kind: journal.Open, Quantity: "100", Price: "12.50"}} {
		holding.Draft = draft
		app.commitRecordEntry()
	}
	holding.Draft.Text = "未提交的下一条"
	app.editRecordEntry(holding.Entries[0].ID)
	holding.Edit.Text = "正在编辑的计划"
	app.recordsBrowser.detailOpen = true

	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); app.recordsPage(c) }, 950, 700)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.SetLocale("zh-Hans")
	for range 4 {
		tt.Frame()
	}
	saveRecordEvidence(t, tt, "journal-recording")
	clickReplaySetting(t, tt, app.text().JournalOpenRecord(closed.Instrument))
	tt.Type("复盘补充")
	if app.records.book.Active() != closed || !app.records.review || closed.Summary != "按计划止损，没有继续扛单。复盘补充" {
		t.Fatal("selecting a closed record did not enter its review editor")
	}
	tt.Advance(200 * time.Millisecond)
	saveRecordEvidence(t, tt, "journal-review")
	clickReplaySetting(t, tt, app.text().JournalOpenRecord(holding.Instrument))
	tt.Type("!")
	if app.records.review || holding.Edit == nil || holding.Edit.Text != "正在编辑的计划!" || holding.Draft.Text != "未提交的下一条" {
		t.Fatal("switching records lost or replaced the in-place edit/new-entry draft")
	}

	tt.SetSize(520, 700)
	for range 4 {
		tt.Frame()
	}
	clickReplaySetting(t, tt, app.text().JournalBackToList())
	if app.recordsBrowser.detailOpen || holding.Edit == nil {
		t.Fatal("narrow back navigation discarded the edit")
	}
	clickReplaySetting(t, tt, app.text().JournalOpenRecord(holding.Instrument))
	tt.Type("?")
	if holding.Edit.Text != "正在编辑的计划!?" {
		t.Fatal("returning from the narrow list failed to restore editor focus")
	}
	app.applyLocale("en")
	tt.SetLocale("en")
	tt.Type(".")
	tt.SetSize(950, 700)
	for range 4 {
		tt.Frame()
	}
	if holding.Edit.Text != "正在编辑的计划!?." || holding.Draft.Text != "未提交的下一条" {
		t.Fatal("locale or layout changes replaced editor state")
	}
	// Moving ownership removes the embedded editor rather than binding its
	// composer a second time. Returning preserves the same unsubmitted edit.
	app.cancelRecordWork()
	app.records.detached = true
	tt.Frame()
	if _, ok := tt.Find(app.text().JournalPlan()); ok {
		t.Fatal("embedded editor remained mounted after detaching")
	}
	app.dockRecord()
	for range 3 {
		tt.Frame()
	}
	tt.Type("+")
	if holding.Edit.Text != "正在编辑的计划!?.+" {
		t.Fatal("docking failed to restore the same editor")
	}
}

func TestJournalPageNewRecordReusesOnlyBlankDrafts(t *testing.T) {
	app := &application{records: recordsState{book: journal.New(time.Now()), ready: true}}
	blank := app.records.book.Active()
	app.newJournalRecord()
	app.newJournalRecord()
	if len(app.records.book.Records) != 1 || app.records.book.Active() != blank {
		t.Fatal("repeated new actions multiplied blank records")
	}
	blank.Draft.Quantity = "-"
	app.newJournalRecord()
	if len(app.records.book.Records) != 2 || app.records.book.Active() == blank || blank.Draft.Quantity != "-" {
		t.Fatal("new action reused and overwrote partial input")
	}
}

func TestJournalPageHistoricalSelectionCancelsAutoCloseAndRequiresDurableReview(t *testing.T) {
	app := &application{records: recordsState{book: journal.New(time.Now()), ready: true}}
	seedClosedRecord(t, app)
	r := app.records.book.Active()
	r.Summary, r.UpdatedAt = "复盘", time.Now()
	app.saveRecord()
	revision := app.records.revision
	if app.recordStatus(r) != app.text().JournalAwaitingReview() {
		t.Fatal("pending explicit save was marked reviewed")
	}
	failure := errors.New("disk unavailable")
	app.applyRecordSave(journal.Result{Revision: revision, Err: failure})
	if app.recordStatus(r) != app.text().JournalAwaitingReview() {
		t.Fatal("failed explicit save was marked reviewed")
	}
	// Clicking history bypasses quick-entry's automatic next-record creation.
	app.openJournalRecord(r.ID)
	if app.records.book.Active() != r || len(app.records.book.Records) != 1 {
		t.Fatal("opening history created another record")
	}
	app.queueRecords(true)
	if app.applyRecordSave(journal.Result{Revision: app.records.revision}) {
		t.Fatal("late save closed a historical record's editor")
	}
	if app.recordStatus(r) != app.text().JournalReviewed() {
		t.Fatal("successful retry did not mark the review saved")
	}
	r.Summary = "保存后的修改"
	app.changeRecord()
	if app.recordStatus(r) != app.text().JournalAwaitingReview() {
		t.Fatal("post-save changes were marked reviewed")
	}
}
