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

func (a *application) replaySettingsDialog(c *ui.Context) {
	if !a.chart.canEditReplaySettings() {
		a.replaySettingsOpen = false
	}
	wasOpen := a.replaySettingsOpen
	components.Dialog(c.Key("replay-settings-dialog"), &a.replaySettingsOpen, components.DialogOptions{Label: "回放设置"}, func(panel ui.Element) {
		panel.Width(360).Padding(18).Gap(16)
		components.Text(c, "回放设置").FontSize(18).Bold().SingleLine()

		ui.Column(c).FillWidth().Gap(6).Children(func() {
			components.Text(c, "初始资金").SingleLine()
			components.NumberInput(c.Key("replay-initial-capital"), &a.replaySettingsDraft.InitialCapital, .01, math.MaxFloat64, 1000).
				Label("初始资金").AutoFocus().FillWidth()
		})
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			components.Text(c, "资金杠杆（倍）").SingleLine()
			components.NumberInput(c.Key("replay-leverage"), &a.replaySettingsDraft.Leverage, 1, math.MaxFloat64, 1).
				Label("资金杠杆").FillWidth()
		})
		ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
			components.Button(c.Key("replay-settings-cancel"), "取消", components.ButtonSecondary).
				OnClick(a.closeReplaySettings)
			components.Button(c.Key("replay-settings-save"), "保存", components.ButtonPrimary).
				OnClick(a.saveReplaySettings)
		})
	})
	if wasOpen && !a.replaySettingsOpen {
		a.closeReplaySettings()
	}
}
