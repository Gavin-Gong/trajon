package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/journal"
)

func (a *application) recordComposer(c *ui.Context, r *journal.Record) {
	ui.Column(c.Key("new-entry")).FillWidth().Padding(4, 16).Gap(8).Shrink(0).Children(func() {
		a.recordEntryTypes(c, &r.Draft)
		ui.Column(c.Key("composer-surface")).FillWidth().Padding(12).Gap(8).Radius(12).
			Background(c.Theme().Surface.Alpha(.88)).Border(1, c.Theme().Text.Alpha(.1)).Children(func() {
			a.recordEditorFields(c, &r.Draft, false)
		})
	})
}

func (a *application) recordInlineEditor(c *ui.Context, draft *journal.Draft) {
	ui.Column(c.Key("inline-editor")).FillWidth().Gap(8).Transition(journalReveal).Children(func() {
		a.recordEntryTypes(c, draft)
		a.recordEditorFields(c, draft, true)
	})
}

func (a *application) recordEntryTypes(c *ui.Context, draft *journal.Draft) {
	s := &a.records
	kinds := s.book.Active().EditorKinds()
	labels := make([]string, len(kinds))
	for i, kind := range kinds {
		labels[i] = a.recordKind(kind)
	}
	// Expose only actions allowed by the committed trading phase.
	components.ScrollView(c.Key("entry-types-scroll"), components.ScrollViewOptions{Axes: components.ScrollHorizontal}).
		FillWidth().Height(34).MinWidth(0).Children(func() {
		tabs := recordSegmented(c.Key("entry-types"), &s.kindChoice, 12, nil, labels...)
		tabs.List.Label(a.text().JournalEntryType()).OnChange(func() {
			if err := s.book.Active().SelectKind(kinds[s.kindChoice]); err != nil {
				s.entryError = err
				return
			}
			a.changeRecord()
		})
	})
}

func (a *application) recordEditorFields(c *ui.Context, draft *journal.Draft, inline bool) {
	s, text := &a.records, a.text()
	if !draft.Kind.IsText() {
		ui.Row(c.Key("execution-details")).FillWidth().Gap(8).Transition(journalReveal).Children(func() {
			for _, field := range []struct {
				key, label, hint string
				value            *string
			}{
				{"entry-quantity", text.JournalQuantity(), text.JournalQuantityHint(), &draft.Quantity},
				{"entry-price", text.JournalPrice(), text.JournalPrice(), &draft.Price},
			} {
				components.TextBox(c.Key(field.key), field.value, components.TextBoxOptions{LeadingText: field.label + " *", ReadOnly: draft.Kind == journal.Close && field.key == "entry-quantity"}).
					Width(0).Grow(1).MinWidth(0).Height(30).Radius(6).Border(0, ui.Transparent).
					Background(c.Theme().Text.Alpha(.035)).Label(field.label).Tooltip(field.hint).OnChange(a.changeRecord)
			}
		})
	}
	height, label, hint := float32(76), text.JournalNote(), text.JournalNoteHint()
	if inline {
		height = 96
	}
	if draft.Kind == journal.Plan {
		label, hint = text.JournalPlan(), text.JournalPlanHint()
	}
	input := recordTextBox(c, "entry-text", &draft.Text, label, true).
		Height(height).MinHeight(60).Padding(0).LineHeight(1.55).Border(0, ui.Transparent).Background(ui.Transparent).
		Placeholder(hint).Bind(&s.composer).OnChange(a.changeRecord)
	if inline {
		input.Padding(8, 10).Radius(8).Background(c.Theme().Surface.Alpha(.75))
	}
	focusID := "new-entry"
	if inline {
		focusID = draft.EntryID
	}
	if s.entryFocusID == focusID {
		input.Focus()
		s.entryFocusID = ""
	}
	input.OnTextKey(func(k ui.TextKey) bool {
		if k.Key != ui.KeyEnter || k.Mods != ui.Cmd && k.Mods != ui.Ctrl {
			return false
		}
		draft.Text = k.Text()
		a.commitRecordEntry()
		if s.entryError == nil {
			k.SetText("")
		}
		return true
	})
	if s.entryError != nil {
		message := text.JournalEntryEmpty()
		switch s.entryError {
		case journal.ErrNumber:
			message = text.JournalNumberInvalid()
		case journal.ErrExecutionFields:
			message = text.JournalExecutionRequired()
		case journal.ErrPhase:
			message = text.JournalPhaseUnavailable()
		case journal.ErrAddDirection:
			message = text.JournalAddDirectionInvalid()
		case journal.ErrReduceQuantity:
			message = text.JournalReduceInvalid()
		case journal.ErrCloseQuantity:
			message = text.JournalCloseInvalid()
		case journal.ErrFollowingTrade:
			message = text.JournalFollowingInvalid()
		}
		components.Text(c, message).FontSize(12).TextColor(c.Theme().TextMuted).Wrap()
	}
	ui.Row(c.Key("entry-actions")).FillWidth().Gap(6).AlignItems(ui.Center).Justify(ui.End).Children(func() {
		if inline {
			components.Button(c.Key("cancel-edit"), text.Cancel(), components.ButtonGhost).Height(28).Padding(4, 8).OnClick(a.cancelRecordEntryEdit)
			components.Button(c.Key("commit-entry"), text.Save(), components.ButtonSecondary).Height(28).Padding(4, 12).
				Disabled(draft.Empty()).OnClick(a.commitRecordEntry)
		} else {
			components.Button(c.Key("commit-entry"), "", components.ButtonPrimary).Size(30, 30).Padding(0).Center().Radius(8).
				Label(text.JournalAddEntry()).Tooltip(text.JournalAddEntry()).Disabled(draft.Empty()).OnClick(a.commitRecordEntry).
				Children(func() { components.Icon(c, lucide.ArrowUp).Size(17, 17) })
		}
	})
}
