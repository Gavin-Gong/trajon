package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/locales"
	"trajon/internal/market"
	appui "trajon/internal/ui"
)

func (a *application) text() locales.Messages {
	// A zero-value application retains Chinese until startup resolves the locale.
	language := a.language
	if language == "" {
		language = locales.Chinese
	}
	return locales.In(string(language))
}

func (a *application) instrumentName(item instrument) string {
	if (a.language == locales.Chinese || a.language == "") && item.NameCN != "" {
		return item.NameCN
	}
	return item.Name
}

func (a *application) instrumentTypeLabel(item instrument) string {
	return appui.InstrumentTypeLabel(a.text(), item)
}

func (a *application) setLanguagePreference(preference locales.Preference) {
	if !preference.Valid() {
		return
	}
	a.languagePreference = preference
	if a.changeLocale != nil {
		// The host reports the language actually shown, including dev overrides.
		a.changeLocale(preference.Locale())
	} else {
		a.applyLocale(string(locales.Resolve(preference, a.preferredLanguages...)))
	}
	if a.localeStore != nil && !a.transientLocale {
		a.languageSaveError = nil
		a.localeStore.Save(preference)
	}
}

// applyLocale runs on the UI thread, from MyGo's OnLocaleChanged listener.
// It updates presentation only: drafts, focus and market/replay state survive.
func (a *application) applyLocale(locale string) {
	a.language = locales.Language(locale)
	if a.recordsPanel != nil {
		a.recordsPanel.SetTitle(a.text().Records())
		a.recordsPanel.Invalidate()
	}
}

func (a *application) publishLocaleSaveResult(preference locales.Preference, err error) {
	if a.window != nil {
		a.window.Update(func() { a.applyLocaleSaveResult(preference, err) })
	}
}

func (a *application) applyLocaleSaveResult(preference locales.Preference, err error) {
	if a.languagePreference == preference {
		a.languageSaveError = err
	}
}

func (a *application) closeSettings() {
	a.cancelLongbridgeAuthorization()
	a.settingsOpen = false
	if a.settingsReturnToPicker {
		a.settingsReturnToPicker = false
		a.instrumentPickerOpen = true
		a.searchInstruments()
		a.instrumentSearchHandle.Focus()
		return
	}
	a.settingsButton.Focus()
}

func (a *application) settingsDialog(c *ui.Context) {
	text := a.text()
	wasOpen := a.settingsOpen
	components.Dialog(c.Key("settings-dialog"), &a.settingsOpen, components.DialogOptions{Label: text.Settings()}, func(panel ui.Element) {
		panel.Width(440).Padding(18).Gap(16)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			components.Text(c, text.Settings()).FontSize(18).Bold().SingleLine()
			ui.Spacer(c)
			components.Button(c.Key("settings-close"), "", components.ButtonGhost).
				Size(28, 28).Padding(0).Radius(14).Center().Label(text.CloseSettings()).
				OnClick(a.closeSettings).Children(func() {
				components.Icon(c, lucide.X).Size(15, 15)
			})
		})
		ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Center).Children(func() {
			components.Text(c, text.LanguageLabel()).SingleLine()
			ui.Spacer(c)
			preference := a.languagePreference
			if !preference.Valid() {
				preference = locales.System
			}
			label := func(value locales.Preference) string {
				switch value {
				case locales.PreferChinese:
					return text.ChineseName()
				case locales.PreferEnglish:
					return text.EnglishName()
				default:
					return text.SystemLanguage()
				}
			}
			sel := components.Select(c.Key("settings-language"), &preference)
			sel.Trigger.Width(200).Label(text.LanguageLabel()).Bind(&a.settingsLanguage).AutoFocus().
				OnChange(func() {
					a.setLanguagePreference(preference)
					a.settingsLanguage.Focus(c)
				}).
				Children(func() { components.Text(c, label(preference)).SingleLine() })
			sel.Popup(func(panel ui.Element) {
				panel.Width(200)
				for _, option := range []locales.Preference{locales.System, locales.PreferChinese, locales.PreferEnglish} {
					sel.Item(option).Label(label(option)).Children(func() {
						components.Text(c, label(option)).SingleLine()
					})
				}
			})
		})
		if a.languageSaveError != nil {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				components.Text(c, text.LanguageSaveFailed()).TextColor(c.Theme().TextMuted).Wrap().
					Tooltip(a.languageSaveError.Error()).Grow(1).MinWidth(0)
				components.Button(c.Key("settings-language-retry"), text.Retry(), components.ButtonSecondary).
					OnClick(func() { a.setLanguagePreference(a.languagePreference) })
			})
		}
		a.longbridgeSettings(c)
	})
	if wasOpen && !a.settingsOpen {
		a.closeSettings()
	}
}

func (a *application) replayPositionCaption(quantity float64) string {
	return replayPositionCaption(quantity, a.text())
}

func (a *application) replayExitLabel(kind market.ReplayExitKind) string {
	if kind == market.ReplayTakeProfit {
		return a.text().TakeProfit()
	}
	return a.text().StopLoss()
}
