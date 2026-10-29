package app

import (
	"log"
	"math"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"trajon/internal/journal"
)

type recordsState struct {
	book                                         journal.Book
	dir                                          string
	store                                        *journal.Store
	loading, ready, closed, opening              bool
	loadError, saveError, entryError, imageError error
	revision, savedRevision                      uint64
	closeAfterSave                               bool
	reviewSaves                                  map[string]uint64
	detached                                     bool // The editor belongs to exactly one window at a time.
	returnToDetail                               bool
	instrument                                   ui.Handle
	review                                       bool
	tabChoice, kindChoice                        int
	timeline, reviewScroll                       ui.ScrollState
	composer                                     ui.Handle
	summary                                      ui.Handle
	tagInput                                     ui.Handle
	entryFocusID                                 string
	imageJob                                     uint64
	imageBusy                                    bool
	images                                       map[string]*ui.Bitmap
	imageErrors                                  map[string]error
	previewImage                                 string
	imageButtons                                 map[string]*ui.Handle
}

// Opening waits for pending loads/writes before replacing an explicitly saved
// review. Autosaved drafts, failed saves and post-save edits stay on the record.
func (a *application) openRecords() {
	s := &a.records
	s.closeAfterSave = false
	s.opening = !s.closed
	if !s.ready || !s.opening {
		return
	}
	r := s.book.Active()
	if !r.CanReview() || r.SavedAt.IsZero() || r.SavedAt.Before(r.UpdatedAt) || s.saveError != nil {
		s.opening = false
		return
	}
	if s.savedRevision < s.revision {
		return
	}
	s.opening = false
	a.newRecord()
}

func (a *application) loadRecords() {
	s, win := &a.records, a.recordsUpdateWindow()
	if s.closed || s.loading || s.ready || win == nil {
		return
	}
	s.loading, s.loadError = true, nil
	go func() {
		dir, err := journal.Directory()
		var book journal.Book
		if err == nil {
			book, err = journal.Load(dir, time.Now())
		}
		a.updateRecords(win, func() {
			if s.closed {
				return
			}
			s.loading, s.loadError = false, err
			if err != nil {
				return
			}
			s.book, s.dir, s.ready = book, dir, true
			a.defaultRecordTab()
			a.syncRecordKind()
			s.store = journal.NewStore(dir, func(result journal.Result) {
				a.updateRecords(win, func() {
					if a.applyRecordSave(result) {
						a.hideRecordsPanel()
					}
				})
			})
			a.recordsBrowser.detailOpen = !book.Active().Empty()
			a.loadRecordImages()
			if s.detached && book.Active().Instrument != "" {
				a.focusRecordInput()
			}
			if s.opening {
				a.openRecords()
			}
		})
	}()
}

// applyRecordSave returns whether the current explicit save should dismiss the
// native panel. Draft saves and stale results must never close a later edit/open.
func (a *application) applyRecordSave(result journal.Result) bool {
	s := &a.records
	if s.closed {
		return false
	}
	if result.Err == nil {
		s.savedRevision = max(s.savedRevision, result.Revision)
	}
	if result.Revision == s.revision {
		s.saveError = result.Err
		if result.Err == nil && s.closeAfterSave {
			s.closeAfterSave = false
			return true
		}
	}
	if s.opening {
		a.openRecords()
	}
	return false
}

// Only explicit input actions call this: never persist from an unguarded build.
func (a *application) changeRecord() {
	s := &a.records
	if !s.ready || s.closed {
		return
	}
	s.opening, s.closeAfterSave = false, false
	a.syncRecordKind()
	s.book.Active().UpdatedAt = time.Now()
	s.entryError = nil
	a.queueRecords(false)
}

func (a *application) queueRecords(immediate bool) {
	s := &a.records
	if !s.ready || s.closed {
		return
	}
	s.revision++
	s.saveError = nil
	s.store.Save(journal.Snapshot{Revision: s.revision, Book: s.book, Immediate: immediate})
	a.invalidateRecords()
}

func (a *application) saveRecord() {
	s := &a.records
	if !s.ready || s.closed || !s.book.Active().CanReview() {
		return
	}
	s.opening = false
	r := s.book.Active()
	r.AddTag()
	r.SavedAt = time.Now()
	a.queueRecords(true)
	if s.reviewSaves == nil {
		s.reviewSaves = map[string]uint64{}
	}
	s.reviewSaves[r.ID] = s.revision
	s.closeAfterSave = true
}

func (a *application) newRecord() {
	s := &a.records
	// Reuse a genuinely blank record, never a partial number or entry draft.
	var blank *journal.Record
	if active := s.book.Active(); active != nil && active.Empty() {
		blank = active
	} else {
		for _, r := range s.book.Records {
			if r.Empty() {
				blank = r
				break
			}
		}
	}
	if blank != nil {
		s.book.ActiveID = blank.ID
	} else {
		s.book.AddRecord(time.Now())
	}
	s.review, s.tabChoice = false, 0
	a.resetRecordView()
	s.instrument.Focus()
	a.queueRecords(false)
}

func (a *application) defaultRecordTab() {
	s := &a.records
	s.review = s.book.Active().CanReview() && s.book.Active().Edit == nil
	s.tabChoice = 0
	if s.review {
		s.tabChoice = 1
	}
}

func (a *application) selectRecord() {
	a.defaultRecordTab()
	a.resetRecordView()
	a.focusRecordInput()
	a.queueRecords(false)
}

func (a *application) syncRecordKind() {
	r := a.records.book.Active()
	r.ReconcileEditor()
	a.records.kindChoice = max(0, slices.Index(r.EditorKinds(), r.Editor().Kind))
	if a.records.review && !r.CanReview() {
		a.setRecordTab(false)
	}
}

func (a *application) focusRecordInput() {
	s := &a.records
	if s.review {
		s.summary.Focus()
		return
	}
	if len(s.book.Active().EditorKinds()) == 0 {
		s.entryFocusID = ""
		s.composer.CancelFocus()
		return
	}
	s.entryFocusID = "new-entry"
	if edit := s.book.Active().Edit; edit != nil {
		s.entryFocusID = edit.EntryID
	}
}

func (a *application) resetRecordView() {
	s := &a.records
	s.opening, s.closeAfterSave = false, false
	s.imageJob++
	s.imageBusy, s.imageError, s.entryError = false, nil, nil
	s.previewImage = ""
	s.entryFocusID = ""
	a.syncRecordKind()
	s.instrument.CancelFocus()
	s.composer.CancelFocus()
	s.summary.CancelFocus()
	for _, handle := range s.imageButtons {
		handle.CancelFocus()
	}
	// Preserve the bound Handle; only reset the scroll offsets.
	s.timeline.X, s.timeline.Y, s.timeline.MaxY = 0, 0, 0
	s.reviewScroll.X, s.reviewScroll.Y, s.reviewScroll.MaxY = 0, 0, 0
	a.loadRecordImages()
}

func (a *application) setRecordTab(review bool) {
	s := &a.records
	if review && (!s.ready || !s.book.Active().CanReview()) {
		review = false
	}
	s.tabChoice = 0
	if review {
		s.tabChoice = 1
	}
	if s.review == review {
		return
	}
	s.closeAfterSave = false
	s.review = review
	s.entryFocusID = ""
	s.composer.CancelFocus()
	s.summary.CancelFocus()
	a.focusRecordInput()
	for _, handle := range s.imageButtons {
		handle.CancelFocus()
	}
	s.imageJob++ // Reject a late file-picker or image import after leaving Review.
	s.imageBusy, s.imageError, s.previewImage = false, nil, ""
	if review {
		a.loadRecordImages()
	}
	a.invalidateRecords()
}

func (a *application) commitRecordEntry() {
	s := &a.records
	r := s.book.Active()
	follow := s.timeline.Y >= s.timeline.MaxY-2
	editing := r.Edit != nil
	if err := r.Commit(time.Now()); err != nil {
		s.entryError = err
		return
	}
	if follow && !editing {
		s.timeline.Y = math.MaxFloat32
	}
	a.syncRecordKind()
	a.focusRecordInput()
	a.changeRecord()
}

func (a *application) editRecordEntry(id string) {
	s := &a.records
	s.book.Active().EditEntry(id)
	a.setRecordTab(false)
	a.syncRecordKind()
	s.entryFocusID = id
	a.changeRecord()
}

func (a *application) cancelRecordEntryEdit() {
	a.records.book.Active().Edit = nil
	a.syncRecordKind()
	a.focusRecordInput()
	a.changeRecord()
}

func (a *application) removeRecordEntry(id string) {
	r := a.records.book.Active()
	if err := r.RemoveEntry(id); err != nil {
		a.records.entryError = err
		return
	}
	a.syncRecordKind()
	a.focusRecordInput()
	a.changeRecord()
}

func (a *application) addRecordTag() {
	a.records.book.Active().AddTag()
	a.records.tagInput.Focus()
	a.changeRecord()
}

func (a *application) removeRecordTag(tag string) {
	r := a.records.book.Active()
	r.Tags = slices.DeleteFunc(r.Tags, func(s string) bool { return s == tag })
	a.changeRecord()
}

func (a *application) closeRecordImagePreview() {
	s := &a.records
	if handle := s.imageButtons[s.previewImage]; handle != nil {
		handle.Focus()
	}
	s.previewImage = ""
}

func (a *application) removeRecordImage(file string) {
	r := a.records.book.Active()
	r.Screenshots = slices.DeleteFunc(r.Screenshots, func(s journal.Screenshot) bool { return s.File == file })
	if a.records.previewImage == file {
		a.records.previewImage = ""
	}
	delete(a.records.images, file)
	delete(a.records.imageErrors, file)
	if handle := a.records.imageButtons[file]; handle != nil {
		handle.CancelFocus()
	}
	a.changeRecord()
}

func (a *application) recordSavePending(id string) bool {
	return a.records.reviewSaves[id] > a.records.savedRevision
}

func (a *application) cancelRecordWork() {
	s := &a.records
	s.opening, s.closeAfterSave = false, false
	s.imageJob++
	s.imageBusy, s.previewImage = false, ""
}

func (a *application) recordsEditorWindow() *mygo.Window {
	if a.records.detached {
		return a.recordsPanel
	}
	return a.window
}

// The main window owns journal publication even while the panel is hidden.
func (a *application) recordsUpdateWindow() *mygo.Window {
	if a.window != nil {
		return a.window
	}
	return a.recordsPanel
}

func (a *application) updateRecords(win *mygo.Window, apply func()) {
	win.Update(func() {
		if a.records.closed {
			return
		}
		apply()
		a.invalidateRecords()
	})
}

func (a *application) invalidateRecords() {
	if a.window != nil {
		a.window.Invalidate()
	}
	if a.recordsPanel != nil {
		a.recordsPanel.Invalidate()
	}
}

// Called after App.Run, outside the native event loop. Pending drafts finish
// writing even if the floating panel was hidden when the main window closed.
func (a *application) closeRecordsStore() {
	if err := a.records.store.Close(); err != nil {
		log.Printf("close journal: %v", err)
	}
}
