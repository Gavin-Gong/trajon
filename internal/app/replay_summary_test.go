package app

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

func TestReplaySummaryPausesAndShowsLastRevealedTradingStatistics(t *testing.T) {
	app, tt := newReplaySettingsTester()
	const start = 120
	app.chart.selectReplayStart(start, len(app.candles))
	for _, trade := range []market.ReplayTrade{
		{Side: market.ReplayBuy, Quantity: 2, Price: 100, Time: app.candles[start].Time},
		{Side: market.ReplaySell, Quantity: 1, Price: 120, Time: app.candles[start].Time},
		{Side: market.ReplayBuy, Quantity: 1, Price: 130, Time: app.candles[start].Time},
	} {
		if err := app.chart.replayAccount.Execute(app.replaySettings, trade); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.chart.replayAccount.SetExit(market.ReplayTakeProfit, 135, app.candles[start].Time); err != nil {
		t.Fatal(err)
	}
	app.candles[start].Close = 95
	app.candles[len(app.candles)-1].Close = 9999
	app.chart.Start, app.chart.hover = 500, 550
	tt.Frame()
	clickReplaySetting(t, tt, "播放")
	app.chart.replayNextStep = time.Now().Add(time.Hour)
	before := app.chart.replayAccount
	before.Trades = append([]market.ReplayTrade(nil), before.Trades...)
	viewport := app.chart.Viewport
	clickReplaySetting(t, tt, "结束")

	if !app.chart.replaySummaryOpen || !app.chart.replayHasStart || app.chart.replayComplete || app.chart.actionPlaying || !app.chart.replayNextStep.IsZero() {
		t.Fatal("opening summary cleared/completed the session or left playback running")
	}
	if app.chart.replayEnd != start+1 || app.chart.Viewport != viewport || !reflect.DeepEqual(app.chart.replayAccount, before) {
		t.Fatal("opening summary changed the cursor, viewport, position, brackets, or fills")
	}
	if !tt.Focused("模拟交易统计") {
		t.Fatal("summary did not focus the dialog instead of its destructive action")
	}
	for _, text := range []string{
		"模拟交易统计", "初始资金", "账户权益", "已实现盈亏", "浮动盈亏", "收益率", "平仓次数", "胜率", "盈亏比", "放弃", "保存",
		"30000.00", "29980.00", "+20.00", "-40.00", "-0.07%", "100.00%", "1 / 0 / 0", "∞",
	} {
		if _, shown := tt.Find(text); !shown {
			t.Fatalf("summary missing %q; texts: %v", text, tt.Texts())
		}
	}
	snapshot := app.chart.replaySummary
	clickReplaySetting(t, tt, "保存")
	tt.Key(0, ui.KeyEnter)
	if !app.chart.replaySummaryOpen || !app.chart.replayHasStart {
		t.Fatal("disabled Save or Enter discarded the session")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("放弃") || tt.Focused("模拟交易统计") {
		t.Fatal("summary's discard action is not keyboard accessible")
	}
	tt.Key(0, ui.KeyTab)
	// Tester.Focused includes focused ancestors: panel focus also reports
	// every child focused. Verify the cycle returns to the panel, skipping Save.
	if !tt.Focused("模拟交易统计") || tt.Focused("结束") || tt.Focused("买入") {
		t.Fatal("focus reached disabled Save or escaped the modal")
	}
	if _, err := app.placeReplayTrade(market.ReplayBuy); !errors.Is(err, errReplayTradingUnavailable) {
		t.Fatalf("trading while summary open returned %v", err)
	}
	if _, err := app.closeReplayPosition(); !errors.Is(err, errReplayTradingUnavailable) {
		t.Fatalf("closing while summary open returned %v", err)
	}
	if app.advanceReplayStep() {
		t.Fatal("playback advanced with a summary open")
	}
	for range 4 {
		tt.Frame()
	}
	if app.chart.replayEnd != start+1 || app.chart.replaySummary != snapshot || !reflect.DeepEqual(app.chart.replayAccount, before) {
		t.Fatal("extra frames or blocked actions changed summary/session data")
	}
}

func TestReplaySummaryDismissalKeepsPausedSessionAndRestoresFocus(t *testing.T) {
	for _, dismissal := range []string{"escape", "backdrop"} {
		t.Run(dismissal, func(t *testing.T) {
			app, tt := newReplaySettingsTester()
			app.chart.selectReplayStart(120, len(app.candles))
			tt.Frame()
			clickReplaySetting(t, tt, "买入")
			before := app.chart.replayAccount
			before.Trades = append([]market.ReplayTrade(nil), before.Trades...)
			for range 2 {
				clickReplaySetting(t, tt, "播放")
				app.chart.replayNextStep = time.Now().Add(time.Hour)
				clickReplaySetting(t, tt, "结束")
				panel, _ := tt.Find("模拟交易统计")
				tt.ClickAt(panel.X+4, panel.Y+4)
				if !app.chart.replaySummaryOpen {
					t.Fatal("clicking empty panel space dismissed the summary")
				}
				switch dismissal {
				case "escape":
					tt.Key(0, ui.KeyEscape)
				case "backdrop":
					tt.ClickAt(22, 90)
				}
				for range 3 {
					tt.Frame()
				}
				if app.chart.replaySummaryOpen || app.chart.replaySummary != (market.ReplayStatistics{}) || !app.chart.replayHasStart || app.chart.actionPlaying || !app.chart.replayNextStep.IsZero() || !reflect.DeepEqual(app.chart.replayAccount, before) {
					t.Fatal("dismissal ended/resumed playback or lost trades")
				}
				if !tt.Focused("结束") {
					t.Fatal("dismissal did not restore focus to End")
				}
				tt.Key(0, ui.KeySpace)
				if !app.chart.replaySummaryOpen || !tt.Focused("模拟交易统计") {
					t.Fatal("keyboard reopening lost summary identity or focus")
				}
				tt.Key(0, ui.KeyEscape)
			}
		})
	}
}

func TestReplaySummaryDiscardClearsSessionAndKeepsSettings(t *testing.T) {
	app, tt := newReplaySettingsTester()
	app.replaySettings = replaySettings{InitialCapital: 25000, Leverage: 3}
	app.chart.Visible, app.chart.Start = 80, 100
	app.chart.selectReplayStart(120, len(app.candles))
	addKlineAnnotation(&app.chart, klineAnnotation{Kind: annotationText, Start: annotationAnchor{Time: app.candles[120].Time, Price: 100}, Text: "标注"})
	annotations := append([]klineAnnotation(nil), app.chart.annotations...)
	tt.Frame()
	clickReplaySetting(t, tt, "买入")
	clickReplaySetting(t, tt, "结束")
	clickReplaySetting(t, tt, "放弃")
	for range 3 {
		tt.Frame()
	}
	if app.chart.replaySummaryOpen || tt.HasText("模拟交易统计") || app.chart.replaySummary != (market.ReplayStatistics{}) || app.chart.replayHasStart || app.chart.replayComplete || !reflect.DeepEqual(app.chart.replayAccount, market.ReplayAccount{}) {
		t.Fatal("discard retained the dialog, replay session, statistics, or trades")
	}
	if app.chart.Start != len(app.candles)-80 || app.chart.Visible != 80 || app.chart.visibleEnd(len(app.candles)) != len(app.candles) || !reflect.DeepEqual(app.chart.annotations, annotations) {
		t.Fatal("discard did not retain zoom/annotations and restore the latest viewport")
	}
	if app.replaySettings != (replaySettings{InitialCapital: 25000, Leverage: 3}) {
		t.Fatal("discard changed saved capital or leverage")
	}
	// Focus the chart, not the returned start-selection button, which must
	// not activate accidentally.
	if !tt.Focused(app.displayInstrument().Symbol + " K 线图") {
		t.Fatal("discard did not restore chart focus")
	}
	for _, label := range []string{"回放起点", "回放设置"} {
		if _, shown := tt.Find(label); !shown {
			t.Fatalf("discard did not restore %s", label)
		}
	}
}

func TestReplaySummaryCompletedSessionKeepsUnclosedPosition(t *testing.T) {
	app, tt := newReplaySettingsTester()
	app.chart.selectReplayStart(120, len(app.candles))
	app.candles[120].Close = 100
	app.candles[len(app.candles)-1].Close = 125
	tt.Frame()
	clickReplaySetting(t, tt, "买入")
	app.completeReplay()
	tt.Frame()
	clickReplaySetting(t, tt, "结束")
	stats := app.chart.replaySummary
	if !app.chart.replaySummaryOpen || !app.chart.replayComplete || app.chart.replayAccount.Quantity != 1 || len(app.chart.replayAccount.Trades) != 1 {
		t.Fatal("completed summary closed the remaining position or lost the session")
	}
	if stats.Equity != 30025 || stats.UnrealizedPnL != 25 || stats.ClosingTrades != 0 {
		t.Fatalf("completed summary did not mark the unclosed position at the final candle: %+v", stats)
	}
	tt.Key(0, ui.KeyEscape)
	if app.chart.replaySummaryOpen || !app.chart.replayComplete || app.canTradeReplay() {
		t.Fatal("dismissing a completed summary restored trading")
	}
	clickReplaySetting(t, tt, "结束")
	clickReplaySetting(t, tt, "放弃")
	if app.chart.replayHasStart || app.chart.replayComplete || app.chart.replayAccount.Quantity != 0 {
		t.Fatal("discard did not clear the completed session")
	}
}

func TestReplaySummaryClosesOnSessionOrMarketChange(t *testing.T) {
	for _, mode := range []string{"reset", "new start", "empty candles", "instrument", "timeframe"} {
		t.Run(mode, func(t *testing.T) {
			app, tt := newReplaySettingsTester()
			app.chart.selectReplayStart(120, len(app.candles))
			tt.Frame()
			clickReplaySetting(t, tt, "买入")
			clickReplaySetting(t, tt, "结束")
			switch mode {
			case "reset":
				app.chart.resetReplay()
			case "new start":
				app.chart.selectReplayStart(130, len(app.candles))
			case "empty candles":
				app.candles = nil
				app.chart.clamp(0)
			case "instrument", "timeframe":
				item, interval := app.currentInstrument, app.chart.timeframe
				if mode == "instrument" {
					item = builtinInstrumentCatalog[0]
				} else {
					interval = "4h"
				}
				app.applyMarketSnapshot(market.Snapshot{Instrument: item, Interval: interval, Candles: demoCandles()})
			}
			for range 3 {
				tt.Frame()
			}
			if app.chart.replaySummaryOpen || app.chart.replaySummary != (market.ReplayStatistics{}) || tt.HasText("模拟交易统计") || !reflect.DeepEqual(app.chart.replayAccount, market.ReplayAccount{}) {
				t.Fatalf("%s retained a stale summary or account", mode)
			}
		})
	}
}

func TestReplaySummaryEmptyMetricsAndLayout(t *testing.T) {
	app, tt := newReplaySettingsTester()
	app.chart.selectReplayStart(120, len(app.candles))
	tt.Frame()
	clickReplaySetting(t, tt, "结束")
	dashes := 0
	for _, text := range tt.Texts() {
		if text == "—" {
			dashes++
		}
	}
	if dashes != 2 || !tt.HasText("0 / 0 / 0") || tt.HasText("NaN") || tt.HasText("Inf") {
		t.Fatalf("empty summary metrics are not well-defined: %v", tt.Texts())
	}
	for _, dark := range []bool{false, true} {
		tt.SetDark(dark)
		for _, size := range [][2]int{{900, 600}, {760, 520}, {512, 520}} {
			tt.SetSize(size[0], size[1])
			tt.Frame()
			panel, ok := tt.Find("模拟交易统计")
			if !ok || panel.X < 0 || panel.Y < 0 || panel.X+panel.W > float32(size[0]) || panel.Y+panel.H > float32(size[1]) {
				t.Fatalf("dialog overflows %v viewport: %+v", size, panel)
			}
			for _, label := range []string{"初始资金", "账户权益", "已实现盈亏", "浮动盈亏", "收益率", "平仓次数", "盈利 / 亏损 / 持平", "胜率", "盈亏比", "放弃", "保存"} {
				bounds, shown := tt.Find(label)
				if !shown || bounds.X < panel.X || bounds.Y < panel.Y || bounds.X+bounds.W > panel.X+panel.W || bounds.Y+bounds.H > panel.Y+panel.H {
					t.Fatalf("%s outside dialog at %v: panel=%+v control=%+v", label, size, panel, bounds)
				}
			}
			discard, _ := tt.Find("放弃")
			save, _ := tt.Find("保存")
			ratio, _ := tt.Find("盈亏比")
			if discard.X >= save.X || discard.Y < ratio.Y+ratio.H || discard.Y != save.Y {
				t.Fatal("summary footer buttons are not below the metrics in the requested order")
			}
		}
	}
}

func TestReplaySummaryMetricFormatting(t *testing.T) {
	for _, tc := range []struct {
		value   float64
		defined bool
		percent string
		ratio   string
	}{
		{0, false, "—", "—"},
		{0, true, "0.00%", "0.00"},
		{.4, true, "40.00%", "0.40"},
		{-.25, true, "-25.00%", "-0.25"},
	} {
		if got := replaySummaryPercent(tc.value, tc.defined); got != tc.percent {
			t.Fatalf("formatted percent = %q, want %q", got, tc.percent)
		}
		if got := replaySummaryRatio(tc.value, tc.defined); got != tc.ratio {
			t.Fatalf("formatted ratio = %q, want %q", got, tc.ratio)
		}
	}
	if got := replaySummaryRatio(math.Inf(1), true); got != "∞" {
		t.Fatalf("no-loss ratio = %q, want ∞", got)
	}
}
