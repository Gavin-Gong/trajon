package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"trajon/internal/locales"
	"trajon/internal/market"
)

func newLocaleSettingsTester() (*application, *ui.Tester) {
	app := &application{
		activePage:           "overview",
		preferredLanguages:   []string{"en-US"},
		languagePreference:   locales.System,
		language:             locales.Resolve(locales.System, "en-US"),
		chart:                newKlineState(0),
		marketToolbarInTitle: true,
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c.Key("toolbar")).Height(40).Children(func() {
				app.leftToolbar(c)
				app.contentTitle(c)
			})
			ui.Box(c.Key("content")).Fill().Children(func() { app.content(c) })
		})
	}, 760, 520)
	return app, tt
}

func TestLanguageSettingsSwitchImmediatelyAndPersist(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	path := filepath.Join(t.TempDir(), "trajon", "locale.json")
	app.localeStore = newLocalePreferenceStore(path, nil)
	t.Cleanup(func() { _ = app.localeStore.Close() })
	clickReplaySetting(t, tt, "Settings")
	if !app.settingsOpen || !tt.Focused("Language") || !tt.HasText("System") {
		t.Fatal("settings did not show and focus the system-language selector")
	}
	for _, choice := range []struct {
		trigger, option, title string
		preference             locales.Preference
	}{
		{"System", "简体中文", "设置", locales.PreferChinese},
		{"简体中文", "English", "Settings", locales.PreferEnglish},
		{"English", "System", "Settings", locales.System},
		{"System", "简体中文", "设置", locales.PreferChinese},
	} {
		clickReplaySetting(t, tt, choice.trigger)
		clickReplaySetting(t, tt, choice.option)
		for range 3 {
			tt.Frame()
		}
		if app.languagePreference != choice.preference || !app.settingsOpen || !tt.HasText(choice.title) || !tt.Focused(app.text().LanguageLabel()) {
			t.Fatalf("choice %q left preference=%q, open=%v, texts=%v", choice.option, app.languagePreference, app.settingsOpen, tt.Texts())
		}
	}
	tt.Key(0, ui.KeyEscape)
	for range 3 {
		tt.Frame()
	}
	if app.settingsOpen || !tt.Focused("设置") || !tt.HasText("概览") || tt.HasText("Overview") {
		t.Fatal("dismissal lost localized opener focus or left untranslated content")
	}
	if err := app.localeStore.Close(); err != nil {
		t.Fatal(err)
	}
	preference, err := loadLocalePreference(path)
	if err != nil || preference != locales.PreferChinese || locales.Resolve(preference, "en-GB") != locales.Chinese {
		t.Fatalf("restored preference=%q, err=%v", preference, err)
	}
	clickReplaySetting(t, tt, "设置")
	clickReplaySetting(t, tt, "关闭设置")
	for range 3 {
		tt.Frame()
	}
	if app.settingsOpen || !tt.Focused("设置") {
		t.Fatal("close button did not restore settings focus")
	}
}

func TestLanguageSaveFailureKeepsChoiceAndOffersRetry(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	parent := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(parent, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	app.localeStore = newLocalePreferenceStore(filepath.Join(parent, "locale.json"), func(_ locales.Preference, err error) { results <- err })
	t.Cleanup(func() { _ = app.localeStore.Close() })
	clickReplaySetting(t, tt, "Settings")
	clickReplaySetting(t, tt, "System")
	clickReplaySetting(t, tt, "简体中文")
	app.applyLocaleSaveResult(locales.PreferChinese, <-results)
	tt.Frame()
	if !tt.HasText("无法保存语言设置") || app.language != locales.Chinese || !app.settingsOpen {
		t.Fatal("failed save discarded the selected language or hid the error")
	}
	if err := os.Remove(parent); err != nil {
		t.Fatal(err)
	}
	clickReplaySetting(t, tt, "重试")
	app.applyLocaleSaveResult(locales.PreferChinese, <-results)
	tt.Frame()
	if tt.HasText("无法保存语言设置") || app.languageSaveError != nil {
		t.Fatal("retry did not clear the failure")
	}
	app.applyLocaleSaveResult(locales.PreferEnglish, errors.New("stale failure"))
	if app.languageSaveError != nil {
		t.Fatal("a stale result replaced the current save result")
	}
}

func TestLanguageChangeRetainsNavigationAndPickerFocus(t *testing.T) {
	app := &application{activePage: "overview"}
	tt := ui.NewTester(app.left, 180, 400)
	clickReplaySetting(t, tt, "回放")
	app.setLanguagePreference(locales.PreferEnglish)
	for range 3 {
		tt.Frame()
	}
	if !tt.Focused("Replay") || app.activePage != "market" || app.pageTitle() != "Replay" {
		t.Fatal("navigation lost identity on language change")
	}
	tt.Key(0, ui.KeySpace)
	if app.activePage != "market" {
		t.Fatal("localized navigation lost keyboard activation")
	}

	app.currentInstrument = defaultInstrument()
	app.openInstrumentPicker()
	app.instrumentPickerQuery = "btc"
	app.instrumentHistory = []instrument{defaultInstrument()}
	tt = ui.NewTester(app.instrumentPickerDialog, 900, 600)
	clickReplaySetting(t, tt, "Spot")
	app.setLanguagePreference(locales.PreferChinese)
	for range 3 {
		tt.Frame()
	}
	if !tt.Focused("现货") || app.instrumentPickerTab != instrumentTabSpot || app.instrumentPickerQuery != "btc" || len(app.filteredInstruments()) != 2 {
		t.Fatal("picker lost tab identity, focus, or query on language change")
	}
	if !tt.HasText("OKX · 现货") || !tt.HasText("历史标的") {
		t.Fatal("picker rows/history were not translated")
	}
	clickReplaySetting(t, tt, "搜索标的")
	app.setLanguagePreference(locales.PreferEnglish)
	for range 3 {
		tt.Frame()
	}
	if !tt.Focused("Search instruments") || !tt.HasText("OKX · Spot") || !tt.HasText("Recent instruments") {
		t.Fatal("picker search focus or translations did not survive the switch")
	}
}

func TestEnglishReplayControlsAndLanguageChangeRetainSession(t *testing.T) {
	app, tt := newReplaySettingsTester()
	app.setLanguagePreference(locales.PreferEnglish)
	tt.Frame()
	clickReplaySetting(t, tt, "Replay settings")
	if !tt.Focused("Initial capital") || !tt.HasText("Leverage (×)") {
		t.Fatal("replay settings not translated")
	}
	replaceReplaySetting(tt, "25000")
	app.setLanguagePreference(locales.PreferChinese)
	tt.Frame()
	if !tt.Focused("初始资金") || app.replaySettingsDraft.InitialCapital != 25000 {
		t.Fatal("language switch discarded a focused numeric draft")
	}
	clickReplaySetting(t, tt, "保存")
	app.setLanguagePreference(locales.PreferEnglish)
	app.chart.selectReplayStart(120, len(app.candles))
	tt.Frame()
	for _, label := range []string{"Play", "Playback speed", "Order quantity", "Buy", "Sell", "End"} {
		if _, ok := tt.Find(label); !ok {
			t.Errorf("missing English control %q", label)
		}
	}
	clickReplaySetting(t, tt, "Buy")
	if !tt.HasText("Buy 1 @") || !tt.HasText("Long 1") {
		t.Fatal("trade feedback or position tag not translated")
	}
	candle := app.candles[app.chart.replayEnd-1]
	if err := app.chart.replayAccount.SetExit(market.ReplayTakeProfit, candle.Close+20, candle.Time); err != nil {
		t.Fatal(err)
	}
	if err := app.chart.replayAccount.SetExit(market.ReplayStopLoss, candle.Close-20, candle.Time); err != nil {
		t.Fatal(err)
	}
	app.chart.annotations = []klineAnnotation{{ID: 1, Kind: annotationText, Start: annotationAnchor{Time: candle.Time, Price: candle.Close}, Text: "用户标注"}}
	tt.Frame()
	if !tt.HasText("Take profit +20.00") || !tt.HasText("Stop loss -20.00") {
		t.Fatal("bracket tags not translated")
	}
	clickReplaySetting(t, tt, "Order quantity")
	replaceReplaySetting(tt, ".25")
	account, start, end, visible := app.chart.replayAccount, app.chart.replayStart, app.chart.replayEnd, app.chart.Visible
	app.setLanguagePreference(locales.PreferChinese)
	for range 3 {
		tt.Frame()
	}
	if !tt.Focused("下单数量") || app.chart.replayOrderQuantity != .25 || !reflect.DeepEqual(app.chart.replayAccount, account) ||
		app.chart.replayStart != start || app.chart.replayEnd != end || app.chart.Visible != visible || app.chart.annotations[0].Text != "用户标注" {
		t.Fatal("language switch changed replay/trade/annotation state or input focus")
	}
	if !tt.HasText("止盈 +20.00") || !tt.HasText("止损 -20.00") || !tt.HasText("多 1") {
		t.Fatal("position overlays did not switch language")
	}
	app.setLanguagePreference(locales.PreferEnglish)
	tt.Frame()
	clickReplaySetting(t, tt, "End")
	for _, text := range []string{"Trading summary", "Initial capital", "Equity", "Realized P&L", "Unrealized P&L", "Closing fills", "Win rate", "Average win/loss", "Discard", "Save"} {
		if !tt.HasText(text) {
			t.Errorf("missing translated summary text %q", text)
		}
	}
}

func TestInstrumentSearchClearIsLocalized(t *testing.T) {
	app := &application{currentInstrument: defaultInstrument()}
	app.openInstrumentPicker()
	t.Cleanup(app.stopInstrumentSearch)
	tt := ui.NewTester(app.instrumentPickerDialog, 900, 600)
	tt.Type("btc")
	if _, found := tt.Find("清除搜索"); !found {
		t.Fatal("missing localized clear action")
	}
	app.setLanguagePreference(locales.PreferEnglish)
	tt.Frame()
	if !tt.Focused("Search instruments") || app.instrumentPickerQuery != "btc" {
		t.Fatal("search switch lost focus/query")
	}
	clickReplaySetting(t, tt, "Clear search")
	if app.instrumentPickerQuery != "" || !tt.Focused("Search instruments") ||
		app.instrumentSearch.loading || app.instrumentSearch.cancel != nil {
		t.Fatal("clear action did not clear/cancel search and restore localized placeholder/focus")
	}
	tt.Type("eth")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if !app.instrumentPickerOpen || app.instrumentPickerQuery != "" {
		t.Fatal("first Escape should only clear a non-empty search")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if app.instrumentPickerOpen {
		t.Fatal("second Escape should dismiss the picker")
	}
}

func TestMarketStatusUpdatesLanguageWithoutReloading(t *testing.T) {
	app, tt := newReplaySettingsTester()
	request := candleRequest{Instrument: defaultInstrument(), Interval: "1h", Limit: 100}
	app.pendingCandleLoad = &request
	app.marketError = &market.NoCandlesError{Symbol: request.Instrument.Symbol}
	app.setLanguagePreference(locales.PreferEnglish)
	tt.Frame()
	if !tt.HasText("Could not load OKX · Spot · BTC/USDT") || !tt.HasText("No candles available for BTC/USDT") || !tt.HasText("Retry") {
		t.Fatal("market status not translated to English")
	}
	app.setLanguagePreference(locales.PreferChinese)
	tt.Frame()
	if !tt.HasText("无法加载 OKX · 现货 · BTC/USDT") || !tt.HasText("BTC/USDT 没有可用 K 线") || app.pendingCandleLoad != &request {
		t.Fatal("language change did not update the existing error or changed the request")
	}
}
