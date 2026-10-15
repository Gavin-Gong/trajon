package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
)

// The pinned MyGo SearchField hardcodes its internal placeholder and clear
// label. Compose native controls so both can follow the application's language.
func (a *application) instrumentSearchField(c *ui.Context) {
	text, t := a.text(), c.Theme()
	clear := func() {
		a.instrumentPickerQuery = ""
		a.searchInstruments()
		a.instrumentSearchHandle.Focus(c)
	}
	ui.Box(c.Key("instrument-picker-search")).FillWidth().Children(func() {
		input := components.TextBox(c.Key("input"), &a.instrumentPickerQuery, components.TextBoxOptions{LeadingIcon: lucide.Search()}).
			FillWidth().Label(text.SearchInstrument()).Placeholder(text.SearchSymbol()).
			Bind(&a.instrumentSearchHandle).AutoFocus().OnChange(a.searchInstruments)
		if a.instrumentPickerQuery != "" {
			input.Padding(t.Space(1.5), t.Space(7), t.Space(1.5), t.Space(8)).
				OnShortcut(0, ui.KeyEscape, clear)
			components.Button(c.Key("clear"), "", components.ButtonGhost).
				Absolute().Right(t.Space(1.5)).Top(t.Space(1.5)).Size(t.Space(4), t.Space(4)).
				Padding(0).Radius(t.Space(2)).Center().Label(text.ClearSearch()).OnClick(clear).
				Children(func() {
					components.Icon(c, lucide.X).Size(t.Space(3), t.Space(3)).TextColor(t.TextMuted)
				})
		}
	})
}
