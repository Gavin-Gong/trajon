package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

func newReplaySettingsTester() (*application, *ui.Tester) {
	candles := demoCandles()
	app := &application{
		activePage:        "market",
		currentInstrument: defaultInstrument(),
		candles:           candles,
		chart:             newKlineState(len(candles)),
		replaySettings:    defaultReplaySettings(),
		marketLoadStarted: true,
	}
	tt := ui.NewTester(app.content, 900, 600)
	tt.Frame()
	return app, tt
}

func clickReplaySetting(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func replaceReplaySetting(tt *ui.Tester, value string) {
	tt.Key(ui.Cmd, ui.KeyA)
	if value == "" {
		tt.Key(0, ui.KeyBackspace)
	} else {
		tt.Type(value)
	}
}

func TestReplaySettingsDialogSavesAndReopens(t *testing.T) {
	app, tt := newReplaySettingsTester()
	if app.replaySettings != (replaySettings{InitialCapital: 10000, Leverage: 1}) {
		t.Fatalf("unexpected defaults: %+v", app.replaySettings)
	}
	bar, _ := tt.Find("K 线图操作栏")
	button, ok := tt.Find("回放设置")
	if !ok || button.X < bar.X || button.X+button.W > bar.X+bar.W || button.Y < bar.Y || button.Y+button.H > bar.Y+bar.H {
		t.Fatalf("settings button is outside the action bar: bar=%+v, button=%+v", bar, button)
	}

	clickReplaySetting(t, tt, "回放设置")
	if !app.replaySettingsOpen || !tt.Focused("初始资金") {
		t.Fatal("opening settings did not focus the initial capital input")
	}
	replaceReplaySetting(tt, "25000.50")
	for range 3 {
		tt.Frame()
	}
	clickReplaySetting(t, tt, "资金杠杆")
	replaceReplaySetting(tt, "3.5")
	want := replaySettings{InitialCapital: 25000.5, Leverage: 3.5}
	if app.replaySettingsDraft != want {
		t.Fatalf("draft = %+v, want %+v", app.replaySettingsDraft, want)
	}
	if app.replaySettings != defaultReplaySettings() {
		t.Fatal("editing the dialog changed the saved settings before Save")
	}
	clickReplaySetting(t, tt, "保存") // No Enter required to commit the focused field.
	if app.replaySettingsOpen || app.replaySettings != want {
		t.Fatalf("save left open=%v, settings=%+v", app.replaySettingsOpen, app.replaySettings)
	}
	for range 3 {
		tt.Frame()
	}
	if !tt.Focused("回放设置") {
		t.Fatal("closing the dialog did not restore focus to its action bar button")
	}
	tt.Key(0, ui.KeySpace) // The restored opener must also work from the keyboard.
	if !app.replaySettingsOpen || app.replaySettingsDraft != want || !tt.Focused("初始资金") {
		t.Fatal("reopening did not display saved values and focus the initial capital input")
	}
}

func TestReplaySettingsDismissalDiscardsDraft(t *testing.T) {
	for _, dismissal := range []string{"cancel", "escape", "backdrop"} {
		t.Run(dismissal, func(t *testing.T) {
			app, tt := newReplaySettingsTester()
			for range 2 {
				clickReplaySetting(t, tt, "回放设置")
				replaceReplaySetting(tt, "80000")
				clickReplaySetting(t, tt, "资金杠杆")
				replaceReplaySetting(tt, "5")
				switch dismissal {
				case "cancel":
					clickReplaySetting(t, tt, "取消")
				case "escape":
					tt.Key(0, ui.KeyEscape)
				case "backdrop":
					tt.ClickAt(22, 90)
				}
				if app.replaySettingsOpen || app.replaySettings != defaultReplaySettings() {
					t.Fatalf("%s committed the draft or left the dialog open", dismissal)
				}
				if !tt.Focused("回放设置") {
					t.Fatalf("%s did not restore focus to the settings button", dismissal)
				}
				for range 3 {
					tt.Frame()
				}
				clickReplaySetting(t, tt, "回放设置")
				if app.replaySettingsDraft != defaultReplaySettings() || !tt.Focused("初始资金") {
					t.Fatal("reopening retained dismissed edits or did not focus the input")
				}
				clickReplaySetting(t, tt, "取消")
			}
		})
	}
}

func TestReplaySettingsRejectsInvalidNumbers(t *testing.T) {
	for _, tc := range []struct {
		name, capital, leverage string
	}{
		{name: "zero capital and leverage", capital: "0", leverage: "0"},
		{name: "negative values", capital: "-10", leverage: "-2"},
		{name: "below minimum", capital: "0.001", leverage: "0.5"},
		{name: "non-finite", capital: "NaN", leverage: "Inf"},
		{name: "overflow", capital: "1e999", leverage: "1e999"},
		{name: "invalid text", capital: "abc", leverage: "abc"},
		{name: "empty", capital: "", leverage: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, tt := newReplaySettingsTester()
			clickReplaySetting(t, tt, "回放设置")
			replaceReplaySetting(tt, tc.capital)
			clickReplaySetting(t, tt, "资金杠杆")
			replaceReplaySetting(tt, tc.leverage)
			clickReplaySetting(t, tt, "保存")
			if app.replaySettings != defaultReplaySettings() {
				t.Fatalf("invalid numbers entered saved settings: %+v", app.replaySettings)
			}
		})
	}
}

func TestReplaySettingsAvailableOnlyWithReplayStart(t *testing.T) {
	app, tt := newReplaySettingsTester()
	assertSettings := func(want bool, width float32) {
		t.Helper()
		for range 3 {
			tt.Frame()
		}
		_, shown := tt.Find("回放设置")
		_, startShown := tt.Find("回放起点")
		bar, ok := tt.Find("K 线图操作栏")
		if shown != want || startShown != shown || !ok || bar.W != width || bar.X < 0 || bar.X+bar.W > app.chart.chartWidth {
			t.Fatalf("settings shown=%v, want %v; action bar=%+v, want width=%v", shown, want, bar, width)
		}
	}

	for range 2 {
		assertSettings(true, klineActionBarWidth)
		clickReplaySetting(t, tt, "回放起点")
		assertSettings(true, klineActionBarWidth)
		clickReplaySetting(t, tt, "回放设置")
		replaceReplaySetting(tt, "15000")
		clickReplaySetting(t, tt, "资金杠杆")
		replaceReplaySetting(tt, "2")
		clickReplaySetting(t, tt, "保存")
		if !app.chart.replaySelecting || app.chart.replayHasStart || app.chart.dragging {
			t.Fatal("interacting with the modal committed a replay start or panned the chart")
		}

		center := app.chart.actionBarX + app.chart.actionBarWidth/2
		g := currentKlineGeometry(app)
		tt.ClickAt(g.left+g.width*.4, g.top+g.priceHeight*.5)
		assertSettings(false, klineReplayActionBarWidth)
		if app.chart.actionBarX+app.chart.actionBarWidth/2 != center {
			t.Fatal("confirming replay start moved the action bar's center")
		}
		app.openReplaySettings()
		if app.replaySettingsOpen {
			t.Fatal("settings opened after replay start confirmation")
		}
		clickReplaySetting(t, tt, "播放")
		assertSettings(false, klineReplayActionBarWidth)
		clickReplaySetting(t, tt, "暂停")
		assertSettings(false, klineReplayActionBarWidth)
		clickReplaySetting(t, tt, "重置")
		assertSettings(false, klineReplayActionBarWidth)
		app.completeReplay()
		assertSettings(false, klineReplayActionBarWidth)
		clickReplaySetting(t, tt, "重置")
		assertSettings(false, klineReplayActionBarWidth)
		clickReplaySetting(t, tt, "结束")
		assertSettings(true, klineActionBarWidth)
		if app.replaySettings != (replaySettings{InitialCapital: 15000, Leverage: 2}) {
			t.Fatal("ending replay lost the saved settings")
		}
	}
}

func TestReplaySettingsDialogClosesIfReplayStarts(t *testing.T) {
	app, tt := newReplaySettingsTester()
	clickReplaySetting(t, tt, "回放设置")
	replaceReplaySetting(tt, "90000")
	app.chart.selectReplayStart(app.chart.Start, len(app.candles))
	app.saveReplaySettings()
	tt.Frame()
	if app.replaySettingsOpen || app.replaySettings != defaultReplaySettings() || tt.HasText("保存") {
		t.Fatal("a dialog left open when replay started committed edits or remained visible")
	}
}

func TestReplaySettingsSurviveMarketSeriesChange(t *testing.T) {
	app, tt := newReplaySettingsTester()
	clickReplaySetting(t, tt, "回放设置")
	replaceReplaySetting(tt, "40000")
	clickReplaySetting(t, tt, "资金杠杆")
	replaceReplaySetting(tt, "4")
	clickReplaySetting(t, tt, "保存")
	want := app.replaySettings
	app.applyMarketSnapshot(market.Snapshot{
		Instrument: binanceInstrumentCatalog[2],
		Interval:   "4h",
		Candles:    demoCandles(),
	})
	tt.Frame()
	clickReplaySetting(t, tt, "回放设置")
	if app.replaySettings != want || app.replaySettingsDraft != want {
		t.Fatal("switching market series reset replay settings")
	}
}
