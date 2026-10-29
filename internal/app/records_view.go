package app

import (
	"errors"
	"strings"
	"time"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/journal"
)

var journalColors = ui.ElementTransition{Duration: 120 * time.Millisecond, Colors: true}
var journalReveal = ui.ElementTransition{Duration: 140 * time.Millisecond, Colors: true, Enter: &ui.Motion{Opacity: 0, Y: 3}}

func (a *application) recordsContent(c *ui.Context) {
	s := &a.records
	ui.Column(c.Key("journal")).Fill().Background(c.Theme().Background.Alpha(.66)).Children(func() {
		a.recordsHeader(c)
		if !s.ready {
			a.recordsLoadState(c)
			return
		}
		a.recordBody(c)
		a.recordFooter(c, true)
	})
	a.recordImagePreview(c)
}

// The current instrument and close action share the native draggable header.
// History and new-record controls live in the journal page, not the quick-entry panel.
func (a *application) recordsHeader(c *ui.Context) {
	s, text := &a.records, a.text()
	ui.Row(c.Key("journal-header")).FillWidth().Height(56).Padding(8, 16).Gap(8).
		AlignItems(ui.Center).DragWindow().Shrink(0).Children(func() {
		if s.ready {
			a.recordIdentity(c, s.book.Active(), true)
		} else {
			components.Text(c, text.Records()).FontSize(13).Bold()
			ui.Spacer(c)
		}
		recordIconButton(c, "records-close", lucide.X(), text.CloseRecords()).Shrink(0).OnClick(a.hideRecordsPanel)
	})
}

func (a *application) recordBody(c *ui.Context) {
	s := &a.records
	r := s.book.Active()
	ui.Column(c.Key(r.ID)).FillWidth().Grow(1).MinHeight(0).Transition(journalReveal).Children(func() {
		a.recordTabs(c)
		if s.review {
			a.recordReview(c, r)
		} else {
			a.recordTimeline(c, r)
			if r.Edit == nil && len(r.AvailableKinds()) > 0 {
				a.recordComposer(c, r)
			}
		}
	})
}

func (a *application) recordIdentity(c *ui.Context, r *journal.Record, autofocus bool) {
	text := a.text()
	ui.Row(c.Key(r.ID)).Grow(1).MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() {
		input := recordTextBox(c, "instrument", &r.Instrument, text.JournalInstrument(), false).
			Grow(1).MinWidth(0).Height(38).Padding(4, 0).Border(0, ui.Transparent).Background(ui.Transparent).
			FontSize(21).Bold().Placeholder(text.JournalInstrumentHint()).Bind(&a.records.instrument).OnChange(a.changeRecord)
		if autofocus {
			input.AutoFocus()
		}
		position := r.NetPosition()
		if position.Recorded {
			label := a.recordDirection(position.Direction)
			if position.Direction != "flat" {
				label += " · " + position.Quantity
			}
			components.Text(c, label).FontSize(12).SingleLine().MaxWidth(150).Padding(5, 10).Radius(13).
				Background(c.Theme().Text.Alpha(.045)).Tooltip(text.JournalNetQuantity(position.Quantity))
		}
	})
}

func (a *application) recordTabs(c *ui.Context) {
	s, text := &a.records, a.text()
	ui.Column(c.Key("tabs")).FillWidth().Padding(0, 16, 8).Shrink(0).Children(func() {
		tabs := recordSegmented(c, &s.tabChoice, 13, func(i int) bool { return i == 1 && !s.book.Active().CanReview() }, text.Records(), text.JournalReview())
		tabs.List.Label(text.Records()).OnChange(func() { a.setRecordTab(s.tabChoice == 1) })
	})
}

func (a *application) recordDirection(direction string) string {
	if direction == "long" {
		return a.text().Long()
	}
	if direction == "short" {
		return a.text().Short()
	}
	return a.text().JournalFlat()
}

func (a *application) recordKind(kind journal.Kind) string {
	t := a.text()
	switch kind {
	case journal.Plan:
		return t.JournalPlan()
	case journal.Open:
		return t.JournalOpen()
	case journal.Add:
		return t.JournalAdd()
	case journal.Reduce:
		return t.JournalReduce()
	case journal.Close:
		return t.JournalClose()
	default:
		return t.JournalNote()
	}
}

func (a *application) recordTimeline(c *ui.Context, r *journal.Record) {
	components.ScrollView(c.Key("journal-timeline"), components.ScrollViewOptions{}).
		FillWidth().Grow(1).MinHeight(0).TrackScroll(&a.records.timeline).Children(func() {
		ui.Column(c.Key("entries")).FillWidth().Padding(14, 16, 8).Children(func() { a.recordEntries(c, r) })
	})
}

// Time, action and prose share a single reading column; operations stay in a
// native menu instead of competing with every note on screen.

func (a *application) recordEntries(c *ui.Context, r *journal.Record) {
	text := a.text()
	var editID, deleteID string
	var day string
	for i, entry := range r.Entries {
		date := entry.At.Local().Format("2006-01-02")
		if day != date {
			day = date
			components.Text(c.Key("date-"+entry.ID), date).FontSize(12).TextColor(c.Theme().TextMuted).Margin(0, 0, 10)
		}
		menu := func(m *ui.Menu) {
			if m.Item(text.JournalEdit()).Chosen() {
				editID = entry.ID
			}
			m.Separator()
			if m.Item(text.JournalDelete()).Chosen() {
				deleteID = entry.ID
			}
		}
		row := ui.Row(c.Key(entry.ID)).FillWidth().Gap(14).Transition(journalReveal).ContextMenu(menu)
		row.Children(func() {
			ui.Column(c).Width(40).Shrink(0).Padding(5, 0, 0).Children(func() {
				components.Text(c, entry.At.Local().Format("15:04")).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine()
			})
			if i < len(r.Entries)-1 {
				ui.Box(c).Absolute().Left(46).Top(18).Bottom(0).Width(1).Background(c.Theme().Text.Alpha(.09))
			}
			ui.Box(c).Absolute().Left(44).Top(9).Size(5, 5).Radius(3).Background(c.Theme().TextMuted.Alpha(.5))
			ui.Column(c).Grow(1).MinWidth(0).Gap(6).Padding(0, 0, 20).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					components.Text(c, a.recordKind(entry.Kind)).FontSize(13).Bold().SingleLine()
					ui.Spacer(c)
					recordIconButton(c, "entry-menu", lucide.Ellipsis(), text.JournalEntryActions(a.recordKind(entry.Kind), entry.At.Local().Format("01-02 15:04"))).
						Size(24, 24).TextColor(c.Theme().TextMuted).Menu(menu)
				})
				if !a.records.review && r.Edit != nil && r.Edit.EntryID == entry.ID {
					a.recordInlineEditor(c, r.Edit)
				} else {
					if entry.Quantity != "" || entry.Price != "" {
						detail := entry.Quantity
						if entry.Price != "" {
							if detail == "" {
								detail = text.JournalPrice() + " " + entry.Price
							} else {
								detail += " @ " + entry.Price
							}
						}
						components.Text(c, detail).FontSize(13).TextColor(c.Theme().TextMuted).SingleLine()
					}
					if entry.Text != "" {
						components.Text(c, entry.Text).FontSize(14).LineHeight(1.55).Wrap().Selectable()
					}
				}
			})
		})
	}
	// Mutate only after iterating, including when a native menu supplies input.
	if editID != "" {
		a.editRecordEntry(editID)
	}
	if deleteID != "" {
		a.removeRecordEntry(deleteID)
	}
}

func (a *application) recordReview(c *ui.Context, r *journal.Record) {
	s, text := &a.records, a.text()
	services := c.Services()
	root := components.ScrollView(c.Key("journal-review"), components.ScrollViewOptions{}).FillWidth().Grow(1).MinHeight(0).TrackScroll(&s.reviewScroll)
	if files := root.DroppedFiles(); len(files) > 0 {
		a.importRecordImages(files, nil)
	}
	if drop, ok := ui.DropData(root, transfer.DropOptions{Formats: []transfer.Format{transfer.PNG, transfer.FileList}, Operations: transfer.Copy}); ok {
		a.acceptRecordImageData(drop.Data)
	}
	root.Children(func() {
		c.OnShortcut(ui.Cmd, ui.KeyV, func() { a.pasteRecordImages(services) })
		c.OnShortcut(ui.Ctrl, ui.KeyV, func() { a.pasteRecordImages(services) })
		ui.Column(c).FillWidth().Padding(16).Gap(22).Children(func() {
			ui.Column(c.Key("summary-section")).FillWidth().Gap(10).Children(func() {
				components.Text(c, text.JournalSummary()).FontSize(14).Bold()
				_, height := c.Size()
				editorHeight := max(180, min(340, height*.36)) * .75
				recordTextBox(c, "summary", &r.Summary, text.JournalSummary(), true).Height(editorHeight).Padding(10, 12).
					Background(c.Theme().Surface.Alpha(.5)).Border(0, ui.Transparent).Radius(10).LineHeight(1.6).
					Placeholder(text.JournalSummaryHint()).Bind(&s.summary).OnChange(a.changeRecord).
					OnPaste(func(_ string) bool { return a.pasteRecordImages(services) })
			})
			a.recordTags(c, r)
			ui.Column(c.Key("attachments-section")).FillWidth().Gap(10).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					components.Text(c, text.JournalScreenshots()).FontSize(13).TextColor(c.Theme().TextMuted)
					ui.Spacer(c)
					recordIconButton(c, "paste-screenshot", lucide.Clipboard(), text.JournalPasteScreenshot()).Disabled(s.imageBusy).OnClick(func() { a.pasteRecordImages(services) })
				})
				a.recordScreenshots(c, r)
				if s.imageBusy {
					components.Text(c, text.JournalImageLoading()).FontSize(12).TextColor(c.Theme().TextMuted)
				}
				if s.imageError != nil {
					label := text.JournalImageFailed()
					if errors.Is(s.imageError, journal.ErrImage) {
						label = text.JournalImageInvalid()
					}
					if errors.Is(s.imageError, errRecordImageLimit) {
						label = text.JournalImageLimit()
					}
					components.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted).Tooltip(s.imageError.Error()).Wrap()
				}
			})
		})
	})
}

func (a *application) recordTags(c *ui.Context, r *journal.Record) {
	s, text := &a.records, a.text()
	ui.Column(c.Key("tags-section")).FillWidth().Gap(8).Children(func() {
		components.Text(c, text.JournalTags()).FontSize(13).TextColor(c.Theme().TextMuted)
		ui.Row(c.Key("tags")).FillWidth().Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
			for _, tag := range r.Tags {
				ui.Row(c.Key(tag)).MaxWidth(240).Height(27).Padding(3, 7, 3, 10).Gap(4).Radius(14).
					Background(c.Theme().Text.Alpha(.055)).AlignItems(ui.Center).Children(func() {
					components.Text(c, tag).FontSize(12).SingleLine().MinWidth(0).Tooltip(tag)
					recordIconButton(c, "remove", lucide.X(), text.JournalRemoveTag(tag)).Size(18, 18).OnClick(func() { a.removeRecordTag(tag) })
				})
			}
			recordTextBox(c, "tag-draft", &r.TagDraft, text.JournalTags(), false).Width(120).Height(28).Padding(4, 3).
				Background(ui.Transparent).Border(0, ui.Transparent).FontSize(12).Placeholder(text.JournalAddTag()).
				Bind(&s.tagInput).OnChange(a.changeRecord).OnSubmit(a.addRecordTag)
			if strings.TrimSpace(r.TagDraft) != "" {
				recordIconButton(c, "add-tag", lucide.Check(), text.JournalAddTag()).Size(24, 24).OnClick(a.addRecordTag)
			}
		})
	})
}

func (a *application) recordScreenshots(c *ui.Context, r *journal.Record) {
	s, text := &a.records, a.text()
	tiles := ui.Row(c.Key("screenshots")).FillWidth().Wrap().Gap(8)
	width := tiles.LayoutBounds().W
	tiles.Draw(func(_ *ui.Painter, r ui.Rect) {
		if r.W != width {
			c.Invalidate()
		}
	})
	room := width
	if room == 0 {
		room = 280
	}
	columns := max(1, min(4, int((room+8)/120)))
	tileWidth := max(1, (room-float32(columns-1)*8)/float32(columns))
	tiles.Children(func() {
		for _, screenshot := range r.Screenshots {
			if s.imageButtons == nil {
				s.imageButtons = map[string]*ui.Handle{}
			}
			if s.imageButtons[screenshot.File] == nil {
				s.imageButtons[screenshot.File] = &ui.Handle{}
			}
			ui.Box(c.Key(screenshot.File)).Size(tileWidth, 96).Radius(8).Background(c.Theme().Text.Alpha(.025)).Children(func() {
				button := components.Button(c.Key("preview"), "", components.ButtonGhost).Fill().Padding(3).Radius(8).
					Label(screenshot.Name).Tooltip(screenshot.Name).Bind(s.imageButtons[screenshot.File]).OnClick(func() { s.previewImage = screenshot.File })
				button.Children(func() {
					if bitmap := s.images[screenshot.File]; bitmap != nil {
						components.Image(c, bitmap).Fill().Fit(ui.Contain).Radius(6)
					} else {
						components.Icon(c, lucide.Image).Size(24, 24).TextColor(c.Theme().TextMuted)
					}
				})
				remove := recordIconButton(c, "remove", lucide.X(), text.JournalRemoveScreenshot(screenshot.Name)).Absolute().Top(4).Right(4).Size(22, 22).Radius(11).
					Background(c.Theme().Background.Alpha(.94)).OnClick(func() { a.removeRecordImage(screenshot.File) })
				if !button.Hovered() && !remove.Hovered() && !button.FocusWithin() && !remove.FocusWithin() {
					remove.Opacity(.55)
				}
			})
		}
		components.Button(c.Key("add-screenshot"), "", components.ButtonGhost).Width(tileWidth).Height(96).Padding(10).Gap(7).Column().Radius(8).
			Background(c.Theme().Text.Alpha(.025)).Border(1, c.Theme().Text.Alpha(.09)).Label(text.JournalAddScreenshot()).Disabled(s.imageBusy).
			OnClick(a.chooseRecordImages).Children(func() {
			components.Icon(c, lucide.ImagePlus).Size(20, 20).TextColor(c.Theme().TextMuted)
			components.Text(c, text.JournalAddScreenshot()).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine()
		})
	})
	failed := false
	for _, err := range s.imageErrors {
		failed = failed || err != nil
	}
	if failed {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			components.Text(c, text.JournalImageFailed()).FontSize(12).TextColor(c.Theme().TextMuted).Grow(1).MinWidth(0).Wrap()
			components.Button(c.Key("reload-images"), text.Retry(), components.ButtonGhost).OnClick(a.loadRecordImages)
		})
	}
}

func (a *application) recordImagePreview(c *ui.Context) {
	s, text := &a.records, a.text()
	open := s.previewImage != ""
	wasOpen := open
	components.Dialog(c.Key("screenshot-preview"), &open, components.DialogOptions{Label: text.JournalScreenshots()}, func(panel ui.Element) {
		w, h := c.Size()
		panel.Width(max(0, w-32)).Height(max(0, h-32)).Padding(10).Gap(8)
		ui.Row(c).FillWidth().Justify(ui.End).Shrink(0).Children(func() {
			recordIconButton(c, "close-preview", lucide.X(), text.JournalClosePreview()).AutoFocus().OnClick(a.closeRecordImagePreview)
		})
		if bitmap := s.images[s.previewImage]; bitmap != nil {
			components.Image(c, bitmap).FillWidth().Grow(1).MinHeight(0).Fit(ui.Contain)
		} else if s.imageErrors[s.previewImage] != nil {
			components.Text(c, text.JournalImageFailed()).Wrap()
		} else {
			components.Text(c, text.JournalImageLoading()).Wrap()
		}
	})
	if wasOpen && !open {
		a.closeRecordImagePreview()
	}
}

func (a *application) recordFooter(c *ui.Context, panel bool) {
	s, text := &a.records, a.text()
	ui.Row(c.Key("journal-footer")).FillWidth().Padding(6, 16, 12).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
		label := text.JournalDraftSaved()
		if s.savedRevision < s.revision {
			label = text.JournalSaving()
		} else if r := s.book.Active(); !r.SavedAt.IsZero() && !r.SavedAt.Before(r.UpdatedAt) {
			label = text.JournalSaved()
		}
		if s.saveError != nil {
			label = text.JournalSaveFailed()
		}
		status := components.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted).Grow(1).MinWidth(0).SingleLine()
		if s.saveError != nil {
			status.Tooltip(s.saveError.Error())
			components.Button(c.Key("retry-save"), text.Retry(), components.ButtonGhost).Height(28).OnClick(func() { a.queueRecords(true) })
		}
		if s.review {
			components.Button(c.Key("save-record"), text.Save(), components.ButtonPrimary).Height(30).Padding(6, 18).Radius(8).Disabled(s.imageBusy).OnClick(func() {
				a.saveRecord()
				s.closeAfterSave = panel
			})
		}
	})
}

func recordTextBox(c *ui.Context, key string, value *string, label string, multiline bool) ui.Element {
	return components.TextBox(c.Key(key), value, components.TextBoxOptions{Multiline: multiline}).FillWidth().Label(label)
}

func recordIconButton(c *ui.Context, key string, icon *ui.SVG, label string) ui.Element {
	return components.Button(c.Key(key), "", components.ButtonGhost).Size(28, 28).Padding(0).Center().Radius(7).
		Label(label).Tooltip(label).Transition(journalColors).Children(func() { components.Image(c, icon).Size(15, 15) })
}
