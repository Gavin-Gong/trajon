package app

import (
	"errors"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"trajon/internal/journal"
	"trajon/internal/locales"
)

func newRecordsTester() (*application, *ui.Tester) {
	app := &application{language: locales.Chinese, records: recordsState{book: journal.New(time.Now()), ready: true}}
	app.syncRecordKind()
	tt := ui.NewTester(app.recordsPanelView, 400, 700)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.SetLocale("zh-Hans")
	return app, tt
}

func seedClosedRecord(t *testing.T, app *application) {
	t.Helper()
	r := app.records.book.Active()
	now := time.Now()
	for i, draft := range []journal.Draft{{Kind: journal.Plan, Text: "计划"}, {Kind: journal.Open, Quantity: "100", Price: "10"}, {Kind: journal.Close, Quantity: "-100", Price: "11"}} {
		r.Draft = draft
		if err := r.Commit(now.Add(time.Duration(i) * time.Millisecond)); err != nil {
			t.Fatal(err)
		}
	}
	app.syncRecordKind()
}

func saveRecordEvidence(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("TRAJON_RECORDS_SHOTS")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func TestRecordsComposerAndReviewPreserveDraftsAcrossSwitches(t *testing.T) {
	app, tt := newRecordsTester()
	r := app.records.book.Active()
	tt.Type("手动标的")
	app.records.composer.Focus()
	tt.Frame()
	tt.Type("等待确认")
	tt.Key(0, ui.KeyEnter)
	tt.Type("不追高")
	if r.Draft.Text != "等待确认\n不追高" || len(r.Entries) != 0 {
		t.Fatalf("ordinary Enter should edit, not submit: %+v", r.Draft)
	}
	tt.Key(ui.Cmd, ui.KeyEnter)
	if len(r.Entries) != 1 || r.Entries[0].Text != "等待确认\n不追高" || r.Draft.Text != "" {
		t.Fatalf("scoped submit lost text: %+v", r)
	}
	r.Draft = journal.Draft{Kind: journal.Open, Quantity: "100", Price: "10"}
	app.commitRecordEntry()
	if app.records.entryError != nil {
		t.Fatal(app.records.entryError)
	}
	tt.Frame()
	tt.Type("还没提交的下一条")
	saveRecordEvidence(t, tt, "recording")
	// Record management is hidden, but switching must still retain active drafts.
	app.newRecord()
	tt.Frame()
	if app.records.book.Active().ID == r.ID || r.Draft.Text != "还没提交的下一条" {
		t.Fatal("new record discarded the holding draft")
	}
	app.records.book.ActiveID = r.ID
	app.selectRecord()
	for range 3 {
		tt.Frame()
	}
	if r.Draft.Text != "还没提交的下一条" || r.Instrument != "手动标的" {
		t.Fatal("switching lost the holding draft")
	}
	app.editRecordEntry(r.Entries[0].ID)
	tt.Frame()
	app.records.composer.Focus()
	tt.Frame()
	replaceReplaySetting(tt, "修正后的原始记录")
	clickReplaySetting(t, tt, app.text().Save())
	if r.Edit != nil || r.Entries[0].Text != "修正后的原始记录" || r.Draft.Text != "还没提交的下一条" {
		t.Fatal("editing destroyed the pending note")
	}
	app.records.composer.Focus()
	tt.Frame()
	app.applyLocale("en")
	tt.SetLocale("en")
	tt.Type("!")
	if r.Draft.Text != "还没提交的下一条!" {
		t.Fatalf("locale switch lost focus or draft: %q", r.Draft.Text)
	}
	tt.Key(ui.Cmd, ui.KeyEnter) // Notes can be added while holding, not after close.
	if err := r.SelectKind(journal.Close); err != nil {
		t.Fatal(err)
	}
	r.Draft.Price = "11"
	app.commitRecordEntry()
	if app.records.entryError != nil {
		t.Fatal(app.records.entryError)
	}
	tt.Frame()
	if _, ok := tt.Find(app.text().JournalAddEntry()); ok {
		t.Fatal("completed record kept its new-entry composer")
	}
	clickReplaySetting(t, tt, app.text().JournalReview())
	tt.Type("## 总结\n\n- 等待确认后开仓\n- 加仓偏早")
	if r.Summary != "## 总结\n\n- 等待确认后开仓\n- 加仓偏早" {
		t.Fatal("Markdown source was changed by editing")
	}
	saveRecordEvidence(t, tt, "review")
	app.newRecord()
	tt.Frame()
	app.records.book.ActiveID = r.ID
	app.selectRecord()
	app.setRecordTab(true)
	tt.Frame()
	if r.Summary != "## 总结\n\n- 等待确认后开仓\n- 加仓偏早" {
		t.Fatal("switching lost the completed review draft")
	}
}

func TestRecordImageAcceptanceRejectsWrongTabAndStaleWork(t *testing.T) {
	app, tt := newRecordsTester()
	var services ui.Services
	probe := ui.NewTester(func(c *ui.Context) { services = c.Services() }, 100, 100)
	probe.SetClipboardData(transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, []byte("image")))))
	if app.pasteRecordImages(services) {
		t.Fatal("recording accepted an image paste")
	}
	if len(app.records.book.Active().Screenshots) != 0 {
		t.Fatal("image leaked onto recording tab")
	}
	seedClosedRecord(t, app)
	app.setRecordTab(true)
	app.records.imageJob = 7
	id := app.records.book.ActiveID
	if !app.recordImageJobCurrent(7, id) {
		t.Fatal("current review job rejected")
	}
	// The nested image preview consumes Escape and restores its opener's focus.
	file := strings.Repeat("a", 64) + ".png"
	app.records.book.Active().Screenshots = []journal.Screenshot{{File: file, Name: "review.png"}}
	app.records.images = map[string]*ui.Bitmap{file: ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 8, 6)))}
	app.records.reviewScroll.Y = math.MaxFloat32
	tt.Frame()
	clickReplaySetting(t, tt, "review.png")
	if app.records.previewImage != file {
		t.Fatal("image preview did not open")
	}
	tt.Key(0, ui.KeyEscape)
	for range 3 {
		tt.Frame()
	}
	if app.records.previewImage != "" || !tt.Focused("review.png") {
		t.Fatal("image dismissal lost opener focus")
	}
	app.setRecordTab(false)
	if app.recordImageJobCurrent(7, id) {
		t.Fatal("image work survived leaving review")
	}
	app.setRecordTab(true)
	app.newRecord()
	if app.recordImageJobCurrent(app.records.imageJob, id) {
		t.Fatal("image work attached to a different record")
	}
	tt.SetSize(320, 480)
	saveRecordEvidence(t, tt, "narrow")
}

func TestRecordsGateReviewAndKeepCloseQuantityReadOnly(t *testing.T) {
	app, tt := newRecordsTester()
	r := app.records.book.Active()
	clickReplaySetting(t, tt, app.text().JournalReview())
	app.setRecordTab(true) // Keyboard/programmatic activation follows the same gate.
	if app.records.review || app.records.tabChoice != 0 {
		t.Fatal("review opened before a closed trade")
	}
	r.Draft.Text = "计划"
	app.commitRecordEntry()
	r.Draft = journal.Draft{Kind: journal.Open, Quantity: "-2", Price: "10"}
	app.commitRecordEntry()
	if app.records.entryError != nil {
		t.Fatal(app.records.entryError)
	}
	if err := r.SelectKind(journal.Close); err != nil {
		t.Fatal(err)
	}
	app.syncRecordKind()
	tt.Frame()
	if r.Draft.Quantity != "2" {
		t.Fatalf("short close quantity = %q", r.Draft.Quantity)
	}
	clickReplaySetting(t, tt, app.text().JournalQuantity())
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("999")
	if r.Draft.Quantity != "2" {
		t.Fatal("the automatic close quantity was editable")
	}
	r.Draft.Price = "11"
	app.commitRecordEntry()
	if app.records.entryError != nil {
		t.Fatal(app.records.entryError)
	}
	app.setRecordTab(true)
	tt.Frame()
	if !app.records.review || !r.CanReview() {
		t.Fatal("full close did not unlock review")
	}
	app.removeRecordEntry(r.Entries[len(r.Entries)-1].ID)
	tt.Frame()
	if app.records.review || r.CanReview() || r.NetPosition().Quantity != "-2" {
		t.Fatal("removing close left review accessible")
	}
}

func TestRecordsReopenStartsNewAfterExplicitSave(t *testing.T) {
	app, tt := newRecordsTester()
	seedClosedRecord(t, app)
	r := app.records.book.Active()
	r.Instrument, r.Summary, r.TagDraft = "手动标的", "## 复盘", "纪律"
	r.UpdatedAt = time.Now()
	app.setRecordTab(true)
	tt.Frame()
	clickReplaySetting(t, tt, app.text().Save())
	if !app.applyRecordSave(journal.Result{Revision: app.records.revision}) {
		t.Fatal("successful explicit save should dismiss the panel")
	}
	if app.records.book.Active() != r {
		t.Fatal("saving should not clear or replace the current review")
	}
	app.openRecords()
	for range 3 {
		tt.Frame()
	}
	next := app.records.book.Active()
	if next.ID == r.ID || len(app.records.book.Records) != 2 || app.records.review || app.records.tabChoice != 0 {
		t.Fatal("reopening a saved review did not start a new recording")
	}
	if next.Instrument != "" || next.Summary != "" || len(next.Entries) != 0 || next.Draft.Kind != journal.Plan {
		t.Fatal("the next record was not blank")
	}
	if r.Instrument != "手动标的" || r.Summary != "## 复盘" || len(r.Entries) != 3 || len(r.Tags) != 1 || r.Tags[0] != "纪律" {
		t.Fatal("starting a new record changed the saved review")
	}
	tt.Type("下一篇")
	if next.Instrument != "下一篇" {
		t.Fatal("new record did not focus its instrument input")
	}
	app.openRecords()
	if app.records.book.Active() != next || len(app.records.book.Records) != 2 {
		t.Fatal("reopening an unfinished record created another one")
	}
}

func TestRecordsReopenWaitsForSaveAndRetainsUnsavedWork(t *testing.T) {
	for _, name := range []string{"pending", "failed", "edited", "autosaved"} {
		t.Run(name, func(t *testing.T) {
			app := &application{records: recordsState{book: journal.New(time.Now()), ready: true}}
			seedClosedRecord(t, app)
			r := app.records.book.Active()
			r.Summary, r.UpdatedAt = "复盘草稿", time.Now()
			if name != "autosaved" {
				app.saveRecord()
			}
			app.queueRecords(true) // The panel also flushes on dismissal.
			app.openRecords()
			if app.records.book.Active() != r {
				t.Fatal("reopening replaced a record before its save completed")
			}
			app.applyRecordSave(journal.Result{Revision: app.records.revision - 1})
			if app.records.book.Active() != r {
				t.Fatal("an earlier save result replaced the current record")
			}
			if name == "edited" {
				r.Summary = "保存后的修改"
				app.changeRecord()
			}
			result := journal.Result{Revision: app.records.revision}
			if name == "failed" {
				result.Err = errors.New("disk unavailable")
			}
			app.applyRecordSave(result)
			if name == "pending" {
				if app.records.book.Active() == r || len(app.records.book.Records) != 2 || r.Summary != "复盘草稿" {
					t.Fatal("successful pending save did not finish opening a fresh record")
				}
				return
			}
			app.openRecords()
			if app.records.book.Active() != r || len(app.records.book.Records) != 1 || r.Summary == "" {
				t.Fatal("unsaved work or a failed save was replaced on reopening")
			}
		})
	}
}

func TestRecordsReopenRestoredSavedReview(t *testing.T) {
	app := &application{records: recordsState{book: journal.New(time.Now()), ready: true}}
	seedClosedRecord(t, app)
	r := app.records.book.Active()
	r.Instrument, r.Summary, r.UpdatedAt = "手动标的", "已保存的复盘", time.Now()
	dir := t.TempDir()
	app.records.store = journal.NewStore(dir, nil)
	app.saveRecord()
	if err := app.records.store.Close(); err != nil {
		t.Fatal(err)
	}
	book, err := journal.Load(dir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	restored := &application{records: recordsState{book: book, ready: true, store: journal.NewStore(dir, nil)}}
	restored.openRecords()
	if restored.records.book.ActiveID == r.ID || len(restored.records.book.Records) != 2 || restored.records.book.Record(r.ID).Summary != r.Summary {
		t.Fatal("restored saved review was not preserved behind a new record")
	}
	if err := restored.records.store.Close(); err != nil {
		t.Fatal(err)
	}
	book, err = journal.Load(dir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	restored.records = recordsState{book: book, ready: true}
	restored.openRecords()
	if len(restored.records.book.Records) != 2 || restored.records.book.ActiveID == r.ID {
		t.Fatal("the new active draft was not persisted across restart")
	}
}

func TestRecordSaveFailureRetainsDraftAndRejectsStaleResults(t *testing.T) {
	app, tt := newRecordsTester()
	seedClosedRecord(t, app)
	r := app.records.book.Active()
	r.Summary, r.Draft.Text = "## 复盘", "未提交笔记"
	app.setRecordTab(true)
	app.records.revision = 2
	app.saveRecord()
	failure := errors.New("disk unavailable")
	if app.applyRecordSave(journal.Result{Revision: 2, Err: failure}) || app.records.saveError != nil {
		t.Fatal("stale failure closed the panel or replaced current save state")
	}
	if app.applyRecordSave(journal.Result{Revision: 2}) {
		t.Fatal("stale success closed the panel before the explicit save finished")
	}
	if app.applyRecordSave(journal.Result{Revision: 3, Err: failure}) {
		t.Fatal("failed save closed the panel")
	}
	if app.records.saveError != failure || r.Summary == "" || r.Draft.Text == "" {
		t.Fatal("save failure discarded drafts")
	}
	tt.Frame()
	clickReplaySetting(t, tt, app.text().Retry())
	if app.records.saveError != nil || app.records.revision != 4 {
		t.Fatal("retry did not queue the retained snapshot")
	}
	if !app.applyRecordSave(journal.Result{Revision: 4}) {
		t.Fatal("successful retry did not dismiss the panel")
	}
	if app.records.savedRevision != 4 || r.Summary != "## 复盘" {
		t.Fatal("save completion lost retained content")
	}
	if app.applyRecordSave(journal.Result{Revision: 4}) {
		t.Fatal("duplicate completion dismissed the panel again")
	}
}

func TestRecordAutoCloseIgnoresDraftsAndSupersededSaves(t *testing.T) {
	for _, action := range []string{"autosave", "edit", "dismiss", "reopen", "switch-tab"} {
		t.Run(action, func(t *testing.T) {
			app := &application{records: recordsState{book: journal.New(time.Now()), ready: true}}
			seedClosedRecord(t, app)
			r := app.records.book.Active()
			r.Summary, r.UpdatedAt = "复盘草稿", time.Now()
			app.setRecordTab(true)
			if action == "autosave" {
				app.changeRecord()
			} else {
				app.saveRecord()
			}
			revision := app.records.revision
			switch action {
			case "edit":
				r.Summary = "保存后的修改"
				app.changeRecord()
			case "dismiss":
				app.hideRecordsPanel()
			case "reopen":
				app.openRecords()
			case "switch-tab":
				app.setRecordTab(false)
			}
			if app.applyRecordSave(journal.Result{Revision: revision}) {
				t.Fatal("draft or superseded explicit save dismissed the panel")
			}
			if app.applyRecordSave(journal.Result{Revision: app.records.revision}) {
				t.Fatal("later draft save inherited an earlier auto-close request")
			}
			if r.Summary == "" {
				t.Fatal("save completion discarded the retained review")
			}
		})
	}
}
