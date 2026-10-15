package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

func (a *application) instrumentHistoryChoices(c *ui.Context) {
	if len(a.instrumentHistory) == 0 {
		return
	}
	t := c.Theme()
	ui.Column(c.Key("instrument-history")).FillWidth().Gap(6).Children(func() {
		components.Text(c, a.text().InstrumentHistory()).FontSize(11).TextColor(t.TextMuted).SingleLine()
		components.ScrollView(c.Key("instrument-history-scroll"), components.ScrollViewOptions{Axes: components.ScrollHorizontal}).
			FillWidth().Height(36).Gap(6).AlignItems(ui.Start).Children(func() {
			for _, item := range a.instrumentHistory {
				label := item.Symbol + " · " + a.instrumentTypeLabel(item)
				components.Button(c.Key("instrument-history-"+item.Identity()), label, components.ButtonSecondary).
					Height(28).Padding(0, 8).FontSize(11).Shrink(0).
					Label(a.text().HistoryInstrument(label)).OnClick(func() { a.selectInstrument(item) })
			}
		})
	})
}
