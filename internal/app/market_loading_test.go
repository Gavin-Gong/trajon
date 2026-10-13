package app

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

func TestFirstMarketSnapshotInitializesDefaultViewport(t *testing.T) {
	item := defaultInstrument()
	app := &application{
		currentInstrument: item,
		chart:             newKlineState(0),
	}
	app.applyMarketSnapshot(market.Snapshot{
		Instrument: item,
		Interval:   "1h",
		Candles:    demoCandles(),
	})
	if app.chart.Visible != initialVisibleCandles {
		t.Fatalf("initial visible candles = %d, want %d", app.chart.Visible, initialVisibleCandles)
	}
	if app.chart.Start != len(app.candles)-initialVisibleCandles {
		t.Fatalf("initial viewport start = %d, want %d", app.chart.Start, len(app.candles)-initialVisibleCandles)
	}
}

func TestPrependedHistoryKeepsReplayStartAndCutoffOnTheSameCandles(t *testing.T) {
	item := defaultInstrument()
	candles := demoCandles()
	app := &application{
		currentInstrument: item,
		candles:           cloneCandles(candles),
		chart:             newKlineState(len(candles)),
	}
	app.chart.selectReplayStart(12, len(candles))
	wantStart, wantEnd := app.chart.replayStart+4, app.chart.replayEnd+4
	prepended := append(cloneCandles(candles[:4]), candles...)

	app.applyMarketSnapshot(market.Snapshot{
		Instrument:       item,
		Interval:         "1h",
		Candles:          prepended,
		HistoryPrepended: 4,
	})
	if app.chart.replayStart != wantStart || app.chart.replayEnd != wantEnd {
		t.Fatalf("replay range after history prepend = [%d,%d), want [%d,%d)", app.chart.replayStart, app.chart.replayEnd, wantStart, wantEnd)
	}
	if app.candles[app.chart.replayStart].Time != candles[12].Time {
		t.Fatal("replay start shifted to a different candle after loading older history")
	}
}

func TestMarketErrorIdentifiesSourceAndExposesCause(t *testing.T) {
	item := defaultInstrument()
	request := candleRequest{Instrument: item, Interval: "1h", Limit: binanceCandleLimit}
	app := &application{
		currentInstrument: item,
		candles:           demoCandles()[:2],
		chart:             newKlineState(2),
		marketError:       errors.New("request OKX candles: context deadline exceeded"),
		pendingCandleLoad: &request,
	}
	tt := ui.NewTester(func(c *ui.Context) { app.klineChart(c) }, 900, 560)
	if !tt.HasText("无法加载 OKX · 现货 · BTC/USDT") {
		t.Fatal("market error did not identify the pending OKX instrument")
	}
	if !tt.HasText("context deadline exceeded") {
		t.Fatal("market error did not expose the underlying cause")
	}
}

func TestOlderMarketErrorShowsToast(t *testing.T) {
	item := defaultInstrument()
	candles := demoCandles()
	app := &application{
		currentInstrument: item,
		candles:           cloneCandles(candles),
		chart:             newKlineState(len(candles)),
	}

	app.applyMarketSnapshot(market.Snapshot{
		Instrument: item,
		Interval:   "1h",
		Candles:    candles,
		OlderError: errors.New("request failed"),
	})

	tt := ui.NewTester(func(c *ui.Context) { app.klineChart(c) }, 900, 560)
	if !tt.HasText("无法加载更早行情") {
		t.Fatal("older-history failure toast title was not shown")
	}
	if !tt.HasText("request failed") {
		t.Fatal("older-history failure toast did not include the underlying cause")
	}
	if !tt.HasText("重试") {
		t.Fatal("older-history failure toast did not offer retry")
	}
}
