package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/journal"
)

type recordsBrowserState struct {
	width      float32
	query      string
	filter     int
	detailOpen bool // Also selects the detail route in narrow windows.
	list       ui.ListState
	search     ui.Handle
	popout     ui.Handle
}

func (a *application) recordsPage(c *ui.Context) {
	s := &a.recordsBrowser
	root := ui.Column(c.Key("journal-page")).Fill().MinWidth(0).MinHeight(0)
	width := root.LayoutBounds().W
	root.Draw(func(_ *ui.Painter, r ui.Rect) {
		if r.W != width {
			c.Invalidate()
		}
	})
	if width == 0 {
		width, _ = c.Size()
	}
	if s.width == 0 {
		s.width = 260
	}
	root.Children(func() {
		if !a.records.ready {
			a.recordsLoadState(c)
			return
		}
		narrow := width < 600
		if narrow {
			if s.detailOpen {
				a.recordDetailPane(c, true)
			} else {
				a.recordList(c)
			}
			return
		}
		components.Split(c.Key("journal-split"), &s.width,
			func() { a.recordList(c) },
			func() { a.recordDetailPane(c, false) },
			components.SplitOptions{FirstMin: 220, SecondMin: 340, Default: 260}).
			Label(a.text().JournalResizeList()).Fill()
	})
}

func (a *application) recordsLoadState(c *ui.Context) {
	text := a.text()
	ui.Column(c.Key("journal-load-state")).Fill().Center().Padding(24).Gap(12).Children(func() {
		if a.records.loadError != nil {
			components.Text(c, text.JournalLoadFailed()).TextColor(c.Theme().TextMuted).Tooltip(a.records.loadError.Error()).Wrap()
			components.Button(c.Key("retry"), text.Retry(), components.ButtonSecondary).OnClick(a.loadRecords)
		} else {
			components.Text(c, text.JournalLoading()).TextColor(c.Theme().TextMuted)
		}
	})
}

func (a *application) recordList(c *ui.Context) {
	s, text := &a.recordsBrowser, a.text()
	items := a.records.book.List(s.query, journal.Filter(s.filter))
	ui.Column(c.Key("journal-list-pane")).Fill().MinWidth(0).Children(func() {
		ui.Column(c.Key("list-toolbar")).FillWidth().Padding(12).Gap(12).Shrink(0).Children(func() {
			ui.Row(c).FillWidth().Height(28).AlignItems(ui.Center).Children(func() {
				components.Text(c, text.JournalTitle()).FontSize(15).Bold().SingleLine()
				ui.Spacer(c)
				recordIconButton(c, "new-record", lucide.Plus(), text.JournalNewRecord()).OnClick(a.newJournalRecord)
			})
			ui.Box(c.Key("search")).FillWidth().Children(func() {
				input := components.TextBox(c.Key("input"), &s.query, components.TextBoxOptions{LeadingIcon: lucide.Search()}).
					FillWidth().Height(32).FontSize(12).Label(text.JournalSearchRecords()).Placeholder(text.JournalSearchHint()).
					Bind(&s.search).OnChange(func() { s.list.ScrollTo(0, ui.Start) })
				clear := func() {
					s.query = ""
					s.list.ScrollTo(0, ui.Start)
					s.search.Focus(c)
				}
				if s.query != "" {
					input.Padding(6, 30, 6, 30).OnShortcut(0, ui.KeyEscape, clear)
					recordIconButton(c, "clear", lucide.X(), text.ClearSearch()).Absolute().Right(3).Top(3).Size(26, 26).OnClick(clear)
				}
			})
			tabs := recordSegmented(c.Key("filter"), &s.filter, 12, nil, text.All(), text.JournalInProgress(), text.JournalClosed())
			tabs.List.Label(text.JournalFilter()).OnChange(func() { s.list.ScrollTo(0, ui.Start) })
		})
		if len(items) == 0 {
			label := text.JournalNoRecords()
			if s.query == "" && s.filter == 0 {
				label = text.JournalEmpty()
			}
			ui.Column(c.Key("empty-list")).FillWidth().Grow(1).Center().Padding(20).Gap(12).Children(func() {
				components.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted).Wrap()
				if s.query == "" && s.filter == 0 {
					components.Button(c.Key("new"), text.JournalNewRecord(), components.ButtonGhost).OnClick(a.newJournalRecord)
				}
			})
			return
		}
		ui.List(c.Key("records"), &s.list, len(items)).FillWidth().Grow(1).MinHeight(0).Label(text.JournalTitle()).
			ItemKey(func(i int) any { return items[i].ID }).Rows(func(row ui.ListRow) {
			c, i := row.Context, row.Index
			r := items[i]
			ui.Column(c).FillWidth().Padding(0, 8).Children(func() {
				if i == 0 || !sameRecordDay(items[i-1].CreatedAt, r.CreatedAt) {
					components.Text(c, a.recordDateGroup(r.CreatedAt)).FontSize(11).TextColor(c.Theme().TextMuted).Padding(12, 6, 6)
				}
				a.recordListRow(c, r)
			})
		})
	})
}

func sameRecordDay(a, b time.Time) bool {
	return a.Local().Format("2006-01-02") == b.Local().Format("2006-01-02")
}

func (a *application) recordDateGroup(at time.Time) string {
	now := time.Now()
	if sameRecordDay(at, now) {
		return a.text().JournalToday()
	}
	if sameRecordDay(at, now.AddDate(0, 0, -1)) {
		return a.text().JournalYesterday()
	}
	return at.Local().Format("2006-01-02")
}

func (a *application) recordListRow(c *ui.Context, r *journal.Record) {
	text := a.text()
	title := strings.TrimSpace(r.Instrument)
	if title == "" {
		title = text.JournalUntitled()
	}
	button := components.Button(c.Key(r.ID), "", components.ButtonGhost).FillWidth().MinWidth(0).
		Padding(10).Radius(8).Label(text.JournalOpenRecord(title)).TextColor(c.Theme().Text).Selected(r.ID == a.records.book.ActiveID).
		OnClick(func() { a.openJournalRecord(r.ID) })
	if r.ID == a.records.book.ActiveID {
		button.Background(c.Theme().Text.Alpha(.065))
	}
	button.Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				components.Text(c, title).FontSize(13).Bold().TextColor(c.Theme().Text).SingleLine().Grow(1).MinWidth(0)
				components.Text(c, a.recordStatus(r)).FontSize(11).TextColor(c.Theme().TextMuted).SingleLine().Shrink(0)
			})
			if summary := recordListSummary(r); summary != "" {
				components.Text(c, summary).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine().FillWidth().MinWidth(0)
			}
			ui.Row(c).FillWidth().Gap(5).AlignItems(ui.Center).Children(func() {
				for _, tag := range r.Tags[:min(2, len(r.Tags))] {
					components.Text(c, "#"+tag).FontSize(10).TextColor(c.Theme().TextMuted).SingleLine().MaxWidth(72)
				}
				if len(r.Tags) > 2 {
					components.Text(c, text.JournalMoreTags(len(r.Tags)-2)).FontSize(10).TextColor(c.Theme().TextMuted)
				}
				ui.Spacer(c)
				components.Text(c, r.CreatedAt.Local().Format("15:04")).FontSize(10).TextColor(c.Theme().TextMuted).
					Tooltip(r.CreatedAt.Local().Format("2006-01-02 15:04")).SingleLine().Shrink(0)
			})
		})
	})
}

func recordListSummary(r *journal.Record) string {
	value := strings.TrimSpace(r.Summary)
	if value == "" {
		for i := len(r.Entries) - 1; i >= 0; i-- {
			if value = strings.TrimSpace(r.Entries[i].Text); value != "" {
				break
			}
		}
	}
	if value == "" {
		value = r.Draft.Text
	}
	return strings.Join(strings.Fields(value), " ")
}

func (a *application) recordStatus(r *journal.Record) string {
	text := a.text()
	switch r.Phase() {
	case journal.Ready:
		return text.JournalReady()
	case journal.Holding:
		return text.JournalHolding()
	case journal.Completed:
		if !r.SavedAt.IsZero() && !r.SavedAt.Before(r.UpdatedAt) && !a.recordSavePending(r.ID) {
			return text.JournalReviewed()
		}
		return text.JournalAwaitingReview()
	default:
		return text.JournalDraft()
	}
}

func (a *application) recordDetailPane(c *ui.Context, narrow bool) {
	s, text := &a.records, a.text()
	ui.Column(c.Key("journal-detail")).Fill().MinWidth(0).Children(func() {
		if s.detached {
			ui.Column(c.Key("detached")).Fill().Center().Padding(24).Gap(14).Children(func() {
				components.Text(c, text.JournalDetached()).TextColor(c.Theme().TextMuted).Wrap()
				components.Button(c.Key("dock"), text.JournalDock(), components.ButtonSecondary).OnClick(a.dockRecord)
			})
			return
		}
		if !a.recordsBrowser.detailOpen && s.book.Active().Empty() {
			ui.Column(c.Key("empty")).Fill().Center().Padding(24).Gap(14).Children(func() {
				components.Text(c, text.JournalEmpty()).TextColor(c.Theme().TextMuted)
				components.Button(c.Key("new"), text.JournalNewRecord(), components.ButtonSecondary).OnClick(a.newJournalRecord)
			})
			return
		}
		c.OnShortcut(0, ui.KeyEscape, func() {
			if !a.escapeRecordEditor() && narrow {
				a.backToRecordList()
			}
		})
		ui.Row(c.Key("header")).FillWidth().Height(56).Padding(8, 16).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
			if narrow {
				recordIconButton(c, "back", lucide.ArrowLeft(), text.JournalBackToList()).OnClick(a.backToRecordList)
			}
			a.recordIdentity(c, s.book.Active(), false)
			recordIconButton(c, "popout", lucide.ExternalLink(), text.JournalPopout()).Bind(&a.recordsBrowser.popout).Shrink(0).OnClick(a.popoutRecord)
		})
		a.recordBody(c)
		a.recordFooter(c, false)
	})
	if !s.detached {
		a.recordImagePreview(c)
	}
}

func (a *application) openJournalRecord(id string) {
	s := &a.records
	if !s.ready || s.closed || s.book.Record(id) == nil {
		return
	}
	if s.detached {
		a.hideRecordsPanel()
	}
	s.book.ActiveID = id
	a.recordsBrowser.detailOpen = true
	a.selectRecord()
}

func (a *application) newJournalRecord() {
	if !a.records.ready || a.records.closed {
		return
	}
	if a.records.detached {
		a.hideRecordsPanel()
	}
	a.recordsBrowser.query, a.recordsBrowser.filter = "", 0
	a.newRecord()
	a.recordsBrowser.detailOpen = true
	a.recordsBrowser.list.ScrollTo(0, ui.Start)
}

func (a *application) backToRecordList() {
	a.recordsBrowser.detailOpen = false
	a.cancelRecordWork()
	a.recordsBrowser.search.Focus()
}

func (a *application) dockRecord() {
	a.hideRecordsPanel()
	a.recordsBrowser.detailOpen = true
	a.focusRecordInput()
}
