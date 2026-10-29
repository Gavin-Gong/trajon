package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
)

func (a *application) indicatorsDialog(c *ui.Context) {
	wasOpen := a.indicatorsOpen
	text := a.text()
	components.Dialog(c.Key("indicators-dialog"), &a.indicatorsOpen, components.DialogOptions{Label: text.Indicators()}, func(panel ui.Element) {
		panel.Width(700).Height(480).Padding(18).Gap(16)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			components.Text(c, text.Indicators()).FontSize(18).Bold().SingleLine()
			ui.Spacer(c)
			components.Button(c.Key("indicators-close"), "", components.ButtonGhost).
				Size(28, 28).Padding(0).Radius(14).Center().Label(text.CloseIndicators()).
				OnClick(a.closeIndicators).Children(func() { components.Icon(c, lucide.X).Size(15, 15) })
		})
		ui.Row(c).FillWidth().Grow(1).MinHeight(0).Gap(16).Children(func() {
			ui.Column(c).Width(132).FillHeight().Justify(ui.Start).Gap(4).Role(ui.RoleGroup).Label(text.Indicators()).Children(func() {
				for i, name := range []string{text.Volume(), "EMA", "MACD"} {
					enabled := a.indicatorDraft.ShowVolume
					if i == 1 {
						enabled = a.indicatorDraft.EMAEnabled
					}
					if i == 2 {
						enabled = a.indicatorDraft.MACDEnabled
					}
					selected := a.indicatorSelected == i
					button := components.Button(c.Key(fmt.Sprintf("indicator-choice-%d", i)), "", components.ButtonGhost).
						FillWidth().Height(36).MinWidth(0).Padding(8, 10).Label(name).Selected(selected).
						Transition(ui.ElementTransition{Duration: 120 * time.Millisecond, Colors: true})
					if selected {
						button.Background(c.Theme().Text.Alpha(.1)).TextColor(c.Theme().Text).AutoFocus()
					}
					button.Children(func() {
						components.Text(c, name).SingleLine()
						ui.Spacer(c)
						if enabled {
							components.Icon(c, lucide.Check).Size(14, 14).TextColor(c.Theme().TextMuted)
						}
					})
					button.OnClick(func() { a.indicatorSelected = i })
				}
			})
			ui.Box(c).Width(1).FillHeight().Background(c.Theme().Border)
			components.ScrollView(c.Key("indicator-options-scroll"), components.ScrollViewOptions{}).Grow(1).MinWidth(0).FillHeight().Children(func() {
				ui.Column(c.Key(fmt.Sprintf("indicator-options-%d", a.indicatorSelected))).FillWidth().Gap(16).Children(func() {
					switch a.indicatorSelected {
					case 1:
						a.emaOptions(c)
					case 2:
						a.macdOptions(c)
					default:
						a.volumeOptions(c)
					}
				})
			})
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			if !a.indicatorDraft.valid() {
				message := text.IndicatorPeriodsInvalid()
				if a.indicatorDraft.MACDFast >= a.indicatorDraft.MACDSlow {
					message = text.MACDPeriodsInvalid()
				}
				components.Text(c, message).TextColor(c.Theme().TextMuted).Wrap().Grow(1).MinWidth(0)
			} else {
				ui.Spacer(c)
			}
			components.Button(c.Key("indicators-cancel"), text.Cancel(), components.ButtonSecondary).OnClick(a.closeIndicators)
			components.Button(c.Key("indicators-save"), text.Save(), components.ButtonPrimary).
				Disabled(!a.indicatorDraft.valid()).OnClick(a.saveIndicators)
		})
	})
	if wasOpen && !a.indicatorsOpen {
		a.closeIndicators()
	}
}

func (a *application) volumeOptions(c *ui.Context) {
	text, draft := a.text(), &a.indicatorDraft
	components.Switch(c.Key("volume-enabled"), &draft.ShowVolume, text.ShowIndicator(text.Volume()))
	indicatorColorField(c.Key("volume-up"), text.RisingColor(), &draft.VolumeUp)
	indicatorColorField(c.Key("volume-down"), text.FallingColor(), &draft.VolumeDown)
}

func (a *application) emaOptions(c *ui.Context) {
	text, draft := a.text(), &a.indicatorDraft
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		components.Switch(c.Key("ema-enabled"), &draft.EMAEnabled, text.ShowIndicator("EMA"))
		ui.Spacer(c)
		components.Button(c.Key("ema-add"), text.AddEMALine(), components.ButtonSecondary).Height(28).OnClick(a.addEMALine)
	})
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).TextColor(c.Theme().TextMuted).Children(func() {
		components.Text(c, text.Display()).Width(76).SingleLine()
		components.Text(c, text.Period()).Grow(1).MinWidth(0).SingleLine()
		components.Text(c, text.Color()).Width(48).SingleLine()
		components.Text(c, text.LineWidth()).Width(76).SingleLine()
		ui.Box(c).Width(28)
	})
	for i := range draft.EMALines {
		line := &draft.EMALines[i]
		id, name := line.ID, fmt.Sprintf("EMA %d", i+1)
		ui.Row(c.Key(fmt.Sprintf("ema-line-%d", id))).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			// Onevra v0.2026.11 captures a frame-local Element while painting
			// checkboxes, which expires during the dialog's exit transition.
			ui.Checkbox(c.Key("enabled"), &line.Enabled, name).Width(76)
			components.NumberInput(c.Key("period"), &line.Period, 1, maxIndicatorPeriod, 1).
				Label(text.EMAPeriod(i + 1)).Grow(1).MinWidth(0)
			components.ColorPicker(c.Key("color"), &line.Color).Label(text.EMAColor(i+1)).Size(48, 28)
			components.NumberInput(c.Key("width"), &line.Width, .5, 4, .5).
				Label(text.EMAWidth(i + 1)).Width(76).MinWidth(0)
			components.Button(c.Key("remove"), "", components.ButtonGhost).Size(28, 28).Padding(0).Center().
				Label(text.RemoveEMALine(i + 1)).OnClick(func() { a.removeEMALine(id) }).
				Children(func() { components.Icon(c, lucide.X).Size(14, 14) })
		})
	}
}

func (a *application) macdOptions(c *ui.Context) {
	text, draft := a.text(), &a.indicatorDraft
	components.Switch(c.Key("macd-enabled"), &draft.MACDEnabled, text.ShowIndicator("MACD"))
	indicatorPeriodField(c.Key("macd-fast"), text.FastPeriod(), &draft.MACDFast)
	indicatorPeriodField(c.Key("macd-slow"), text.SlowPeriod(), &draft.MACDSlow)
	indicatorPeriodField(c.Key("macd-signal"), text.SignalPeriod(), &draft.MACDSignal)
	indicatorColorField(c.Key("macd-dif"), text.DIFColor(), &draft.MACDDIF)
	indicatorColorField(c.Key("macd-dea"), text.DEAColor(), &draft.MACDDEA)
	indicatorColorField(c.Key("macd-positive"), text.PositiveColor(), &draft.MACDPositive)
	indicatorColorField(c.Key("macd-negative"), text.NegativeColor(), &draft.MACDNegative)
}

func indicatorPeriodField(c *ui.Context, label string, value *float64) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
		components.Text(c, label).SingleLine()
		ui.Spacer(c)
		components.NumberInput(c.Key("input"), value, 1, maxIndicatorPeriod, 1).Label("MACD " + label).Width(120)
	})
}

func indicatorColorField(c *ui.Context, label string, value *ui.Color) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
		components.Text(c, label).SingleLine()
		ui.Spacer(c)
		components.ColorPicker(c.Key("picker"), value).Label(label).Size(48, 28)
	})
}
