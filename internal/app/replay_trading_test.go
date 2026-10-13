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

func TestReplayTradingControlsAreCompactAndFollowSession(t *testing.T) {
	app, tt := newKlineToolbarTester()
	for _, label := range []string{"下单数量", "买入", "卖出"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("%s shown before replay start confirmation", label)
		}
	}
	clickReplaySetting(t, tt, "回放起点")
	if _, ok := tt.Find("买入"); ok {
		t.Fatal("trading shown during replay selection")
	}
	app.chart.selectReplayStart(app.chart.Start+20, len(app.candles))
	tt.Frame()
	bar, _ := tt.Find("K 线图操作栏")
	previousX := float32(-1)
	for _, label := range []string{"播放", "播放速度", "下单数量", "买入", "卖出", "重置", "结束"} {
		control, ok := tt.Find(label)
		if !ok || control.X < bar.X || control.X+control.W > bar.X+bar.W || control.Y < bar.Y || control.Y+control.H > bar.Y+bar.H || control.X <= previousX {
			t.Fatalf("%s out of order or outside bar: bar=%+v control=%+v", label, bar, control)
		}
		if control.H < 27 || control.H > 29 {
			t.Fatalf("%s height=%v, want compact height near 28", label, control.H)
		}
		previousX = control.X + control.W
	}
	clickReplaySetting(t, tt, "下单数量")
	replaceReplaySetting(tt, "0.25")
	for range 3 {
		tt.Frame()
	}
	if !tt.Focused("下单数量") || app.chart.replayOrderQuantity != .25 {
		t.Fatal("quantity input lost focus or its edit across rebuilds")
	}
	tt.Key(0, ui.KeyUp)
	if app.chart.replayOrderQuantity != 1.25 {
		t.Fatal("quantity input did not step up by 1 from the keyboard")
	}
	tt.Key(0, ui.KeyDown)
	if app.chart.replayOrderQuantity != .25 {
		t.Fatal("quantity input did not step down by 1 from the keyboard")
	}
	clickReplaySetting(t, tt, "下单数量 Increase")
	if app.chart.replayOrderQuantity != 1.25 {
		t.Fatal("quantity stepper did not increase by 1")
	}
	clickReplaySetting(t, tt, "下单数量 Decrease")
	if app.chart.replayOrderQuantity != .25 {
		t.Fatal("quantity stepper did not decrease by 1")
	}
	// Clicking the buy button must use the draft without requiring Enter.
	clickReplaySetting(t, tt, "买入")
	if account := app.chart.replayAccount; account.Quantity != .25 || len(account.Trades) != 1 {
		t.Fatalf("edited quantity was not used: %+v", account)
	}
	if app.chart.canEditReplaySettings() {
		t.Fatal("settings remained editable after replay start confirmation")
	}
	if app.chart.dragging || app.chart.replaySelecting {
		t.Fatal("trading controls leaked pointer events into the chart")
	}
	clickReplaySetting(t, tt, "重置") // Also available after a pre-play trade.
	if !reflect.DeepEqual(app.chart.replayAccount, market.ReplayAccount{}) || app.chart.replayOrderQuantity != .25 {
		t.Fatal("reset did not clear trades while preserving order quantity")
	}
	clickReplaySetting(t, tt, "结束")
	clickReplaySetting(t, tt, "放弃")
	for range 3 {
		tt.Frame()
	}
	for _, label := range []string{"下单数量", "买入", "卖出"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("%s shown after End", label)
		}
	}
}

func TestReplayTradesUseLatestPlayedCloseWithoutPausing(t *testing.T) {
	app, tt := newKlineToolbarTester()
	const start = 120
	app.candles[start].Close = 100
	app.candles[start+1].Close = 125
	app.chart.Start = 100
	app.chart.selectReplayStart(start, len(app.candles))
	tt.Frame()
	clickReplaySetting(t, tt, "下单数量")
	replaceReplaySetting(tt, "0.5")
	clickReplaySetting(t, tt, "播放")
	app.chart.replayNextStep = time.Now().Add(time.Hour)
	nextStep := app.chart.replayNextStep
	app.chart.hover = 100
	clickReplaySetting(t, tt, "买入")
	assertTrade := func(index int, side market.ReplaySide, candleIndex int) {
		t.Helper()
		trade := app.chart.replayAccount.Trades[index]
		candle := app.candles[candleIndex]
		if trade.Side != side || trade.Quantity != .5 || trade.Price != candle.Close || !trade.Time.Equal(candle.Time) {
			t.Fatalf("trade=%+v, want side=%v quantity=.5 at candle %+v", trade, side, candle)
		}
	}
	assertTrade(0, market.ReplayBuy, start)
	if app.advanceReplayStep() {
		t.Fatal("playback ended before the next trade")
	}
	app.chart.Start = 500 // Pan the viewport away from the replay cursor.
	tt.Frame()
	clickReplaySetting(t, tt, "卖出")
	assertTrade(1, market.ReplaySell, start+1)
	if app.chart.replayAccount.Quantity != 0 || app.chart.replayAccount.RealizedPnL != 12.5 {
		t.Fatalf("sell did not close the position: %+v", app.chart.replayAccount)
	}
	clickReplaySetting(t, tt, "卖出")
	assertTrade(2, market.ReplaySell, start+1)
	if app.chart.replayAccount.Quantity != -.5 || !app.chart.actionPlaying || !app.chart.replayNextStep.Equal(nextStep) {
		t.Fatal("selling short paused/rescheduled playback or used the wrong quantity")
	}
	clickReplaySetting(t, tt, "暂停")
	clickReplaySetting(t, tt, "买入")
	assertTrade(3, market.ReplayBuy, start+1)
	if app.chart.replayAccount.Quantity != 0 || app.chart.actionPlaying {
		t.Fatal("paused trade did not close the short or resumed playback")
	}
	before := len(app.chart.replayAccount.Trades)
	for range 4 {
		tt.Frame()
	}
	if len(app.chart.replayAccount.Trades) != before {
		t.Fatal("rebuilding the UI duplicated a trade")
	}
}

func TestReplayTradingUsesSavedCapitalAndLeverage(t *testing.T) {
	app, tt := newReplaySettingsTester()
	clickReplaySetting(t, tt, "回放设置")
	replaceReplaySetting(tt, "100")
	clickReplaySetting(t, tt, "资金杠杆")
	replaceReplaySetting(tt, "2")
	clickReplaySetting(t, tt, "保存")
	app.chart.selectReplayStart(app.chart.Start, len(app.candles))
	app.candles[app.chart.replayStart].Close = 100
	tt.Frame()
	clickReplaySetting(t, tt, "下单数量")
	replaceReplaySetting(tt, "2")
	clickReplaySetting(t, tt, "买入")
	if app.chart.replayAccount.Quantity != 2 {
		t.Fatal("trade did not use saved capital and leverage")
	}
	clickReplaySetting(t, tt, "买入")
	if len(app.chart.replayAccount.Trades) != 1 || !tt.HasText("资金不足") {
		t.Fatal("unaffordable order was recorded or did not show feedback")
	}
	clickReplaySetting(t, tt, "卖出")
	if app.chart.replayAccount.Quantity != 0 || len(app.chart.replayAccount.Trades) != 2 {
		t.Fatal("closing an existing position did not release margin")
	}
}

func TestReplayTradingRejectsUnavailableStatesAndInvalidOrders(t *testing.T) {
	for _, mode := range []string{"no start", "selection", "completed", "loading", "empty", "bad cutoff", "zero quantity", "negative quantity", "nan quantity", "zero price", "infinite price"} {
		t.Run(mode, func(t *testing.T) {
			app, _ := newKlineToolbarTester()
			app.chart.selectReplayStart(120, len(app.candles))
			want := errReplayTradingUnavailable
			switch mode {
			case "no start":
				app.chart.replayHasStart = false
			case "selection":
				app.chart.replaySelecting = true
			case "completed":
				app.completeReplay()
			case "loading":
				app.marketLoading = true
			case "empty":
				app.candles = nil
			case "bad cutoff":
				app.chart.replayEnd = len(app.candles) + 1
			case "zero quantity":
				app.chart.replayOrderQuantity, want = 0, market.ErrInvalidReplayTrade
			case "negative quantity":
				app.chart.replayOrderQuantity, want = -1, market.ErrInvalidReplayTrade
			case "nan quantity":
				app.chart.replayOrderQuantity, want = math.NaN(), market.ErrInvalidReplayTrade
			case "zero price":
				app.candles[120].Close, want = 0, market.ErrInvalidReplayTrade
			case "infinite price":
				app.candles[120].Close, want = math.Inf(1), market.ErrInvalidReplayTrade
			}
			if _, err := app.placeReplayTrade(market.ReplayBuy); !errors.Is(err, want) {
				t.Fatalf("error=%v, want %v", err, want)
			}
			if !reflect.DeepEqual(app.chart.replayAccount, market.ReplayAccount{}) {
				t.Fatal("rejected order changed the account")
			}
		})
	}
}

func TestReplayTradingClearsWithSessionAndMarketSeries(t *testing.T) {
	for _, mode := range []string{"reset", "end", "new start", "empty candles", "instrument", "timeframe"} {
		t.Run(mode, func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			app.currentInstrument = defaultInstrument()
			app.chart.selectReplayStart(120, len(app.candles))
			app.chart.replayOrderQuantity = .125
			tt.Frame()
			clickReplaySetting(t, tt, "买入")
			if len(app.chart.replayAccount.Trades) != 1 || app.chart.replayTradeMessage == "" {
				t.Fatal("initial trade did not create a position and feedback")
			}
			switch mode {
			case "reset":
				app.chart.resetReplay()
			case "end":
				app.chart.endReplay(len(app.candles))
			case "new start":
				app.chart.selectReplayStart(130, len(app.candles))
			case "empty candles":
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
			if !reflect.DeepEqual(app.chart.replayAccount, market.ReplayAccount{}) || app.chart.replayTradeMessage != "" || app.replaySettings != defaultReplaySettings() {
				t.Fatal("session transition retained trades or lost saved settings")
			}
		})
	}
}

func TestReplayTradeFeedbackDoesNotCoverControlsAndExpires(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.selectReplayStart(app.chart.Start, len(app.candles))
	price := 56.516416370720655
	app.candles[app.chart.replayStart].Close = price
	tt.Frame()
	clickReplaySetting(t, tt, "买入")
	message := app.chart.replayTradeMessage
	feedback, ok := tt.Find(message)
	bar, _ := tt.Find("K 线图操作栏")
	if !ok || feedback.Y+feedback.H > bar.Y || message != "买入 1 @ 56.516416" {
		t.Fatalf("feedback is not concise or overlaps controls: message=%q feedback=%+v bar=%+v", message, feedback, bar)
	}
	if got := app.chart.replayAccount.Trades[0].Price; got != price {
		t.Fatalf("display formatting rounded the recorded price: %v", got)
	}
	clickReplaySetting(t, tt, "卖出")
	if app.chart.replayAccount.Quantity != 0 || len(app.chart.replayAccount.Trades) != 2 {
		t.Fatal("trade feedback blocked a subsequent order")
	}
	message = app.chart.replayTradeMessage
	app.chart.replayTradeMessageUntil = time.Now().Add(-time.Second)
	tt.Frame()
	if tt.HasText(message) {
		t.Fatal("expired trade feedback remained visible")
	}
}

func TestReplayTradingDisabledAfterCompletionButResetRestoresIt(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.selectReplayStart(app.chart.Start, len(app.candles))
	tt.Frame()
	clickReplaySetting(t, tt, "买入")
	app.completeReplay()
	tt.Frame()
	clickReplaySetting(t, tt, "买入")
	clickReplaySetting(t, tt, "卖出")
	if len(app.chart.replayAccount.Trades) != 1 {
		t.Fatal("completed replay accepted trades")
	}
	clickReplaySetting(t, tt, "重置")
	clickReplaySetting(t, tt, "卖出")
	if app.chart.replayAccount.Quantity != -1 || len(app.chart.replayAccount.Trades) != 1 {
		t.Fatalf("reset did not restore trading with a fresh account: account=%+v complete=%v end=%v hasStart=%v quantity=%v", app.chart.replayAccount, app.chart.replayComplete, app.chart.replayEnd, app.chart.replayHasStart, app.chart.replayOrderQuantity)
	}
}

func TestReplayTradeTimestampSurvivesHistoryPrepend(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.selectReplayStart(120, len(app.candles))
	first, err := app.placeReplayTrade(market.ReplayBuy)
	if err != nil {
		t.Fatal(err)
	}
	older := Candle{Time: app.candles[0].Time.Add(-time.Hour), Open: 90, High: 110, Low: 80, Close: 100}
	candles := append([]Candle{older}, app.candles...)
	app.applyMarketSnapshot(market.Snapshot{HistoryPrepended: 1, Candles: candles})
	tt.Frame()
	second, err := app.placeReplayTrade(market.ReplaySell)
	if err != nil {
		t.Fatal(err)
	}
	if first.Price != second.Price || !first.Time.Equal(second.Time) || len(app.chart.replayAccount.Trades) != 2 {
		t.Fatal("history prepend changed the replay trade's candle identity")
	}
}
