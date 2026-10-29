package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

// Both journal selectors share one capsule track and raised selected face.
// TabsBase owns selection, arrow keys, focus and accessibility. Keys depend
// on option indices, never on translated labels.
func recordSegmented(c *ui.Context, selected *int, fontSize float32, disabled func(int) bool, labels ...string) ui.TabsParts {
	tabs := ui.TabsBase(c, selected, len(labels))
	t := c.Theme()
	tabs.List.AlignSelf(ui.Start).Padding(3).Gap(2).Radius(18).Background(t.Text.Alpha(.065))
	tabs.List.Children(func() {
		for i, label := range labels {
			c.Key(i)
			tab := tabs.Tab(i).Shrink(0).Height(28).Padding(4, 10).Radius(14).Center().
				Border(1, ui.Transparent).TextColor(t.TextMuted).Background(ui.Transparent).Transition(journalColors)
			unavailable := disabled != nil && disabled(i)
			tab.Disabled(unavailable)
			if unavailable {
				tab.TextColor(t.TextMuted.Alpha(.5))
			}
			on := i == *selected
			if on && !unavailable {
				tab.Background(t.Surface).Border(1, t.Border.Alpha(.4)).TextColor(t.Text).
					Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, .09))
			}
			tab.Children(func() {
				text := components.Text(c, label).FontSize(fontSize).SingleLine().TextAlign(ui.Center)
				if on {
					text.FontWeight(600)
				}
			})
		}
	})
	return tabs
}
