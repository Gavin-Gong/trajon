package app

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"trajon/internal/market"
)

type replaySettings = market.ReplayTradingSettings

func defaultReplaySettings() replaySettings {
	return replaySettings{InitialCapital: 30000, Leverage: 1}
}

func (s *klineState) canEditReplaySettings() bool {
	return !s.replayHasStart && !s.replaySessionStarted && !s.replayComplete
}

func (a *application) openReplaySettings() {
	if !a.chart.canEditReplaySettings() {
		return
	}
	a.replaySettingsDraft = a.replaySettings
	a.replaySettingsOpen = true
}

func (a *application) saveReplaySettings() {
	if !a.replaySettingsOpen || !a.chart.canEditReplaySettings() {
		return
	}
	a.replaySettings = a.replaySettingsDraft
	a.closeReplaySettings()
}

func (a *application) closeReplaySettings() {
	a.replaySettingsOpen = false
	if a.chart.canEditReplaySettings() {
		a.replaySettingsButton.Focus()
	}
}

func (a *application) initialCapitalLabel() string {
	if currency := a.currentInstrument.Currency; currency != "" {
		return a.text().InitialCapitalInCurrency(currency)
	}
	return a.text().InitialCapital()
}

func (a *application) replaySettingsDialog(c *ui.Context) {
	if !a.chart.canEditReplaySettings() {
		a.replaySettingsOpen = false
	}
	wasOpen := a.replaySettingsOpen
	components.Dialog(c.Key("replay-settings-dialog"), &a.replaySettingsOpen, components.DialogOptions{Label: a.text().ReplaySettings()}, func(panel ui.Element) {
		panel.Width(360).Padding(18).Gap(16)
		components.Text(c, a.text().ReplaySettings()).FontSize(18).Bold().SingleLine()

		ui.Column(c).FillWidth().Gap(6).Children(func() {
			components.Text(c, a.initialCapitalLabel()).SingleLine()
			components.NumberInput(c.Key("replay-initial-capital"), &a.replaySettingsDraft.InitialCapital, .01, math.MaxFloat64, 1000).
				Label(a.initialCapitalLabel()).AutoFocus().FillWidth()
		})
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			components.Text(c, a.text().LeverageMultiple()).SingleLine()
			components.NumberInput(c.Key("replay-leverage"), &a.replaySettingsDraft.Leverage, 1, math.MaxFloat64, 1).
				Label(a.text().Leverage()).FillWidth()
		})
		ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
			components.Button(c.Key("replay-settings-cancel"), a.text().Cancel(), components.ButtonSecondary).
				OnClick(a.closeReplaySettings)
			components.Button(c.Key("replay-settings-save"), a.text().Save(), components.ButtonPrimary).
				OnClick(a.saveReplaySettings)
		})
	})
	if wasOpen && !a.replaySettingsOpen {
		a.closeReplaySettings()
	}
}
