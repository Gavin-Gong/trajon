package ui

import (
	myui "github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

// NavigationItem retains Onevra's sidebar appearance without its label-based
// key. The caller supplies Context.Key before construction so focus survives
// language changes. Native ButtonBase owns activation and accessibility.
func NavigationItem(c *myui.Context, label string, icon *myui.SVG, selected bool) myui.Element {
	t := c.Theme()
	button := myui.ButtonBase(c).Label(label).FillWidth().Justify(myui.Start).
		Padding(t.Space(1.75), t.Space(2)).Gap(t.Space(2)).Radius(t.Radius).
		Selected(selected).Background(myui.Color{}).TextColor(t.Text)
	if selected {
		button.Background(t.Text.Alpha(.1))
	}
	button.Children(func() {
		if icon != nil {
			components.Icon(c, func() *myui.SVG { return icon }).Size(t.Space(4.5), t.Space(4.5)).TextColor(t.Accent)
		}
		components.Text(c, label).SingleLine()
	})
	return button
}
