package app

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"trajon/internal/locales"
	"trajon/internal/market"
)

func newReplayPositionTester(t *testing.T, side market.ReplaySide) (*application, *ui.Tester) {
	t.Helper()
	app, tt := newReplaySettingsTester()
	app.chart.Start = 100
	app.chart.selectReplayStart(120, len(app.candles))
	stamp := app.candles[120].Time
	app.candles[120] = Candle{Time: stamp, Open: 100, High: 110, Low: 90, Close: 100, EMA20: 100, Volume: 100}
	if _, err := app.placeReplayTrade(side); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	return app, tt
}

func dragReplayLine(t *testing.T, tt *ui.Tester, label string, dy float32) {
	t.Helper()
	line, ok := tt.Find(label)
	if !ok {
		t.Fatalf("line %q missing", label)
	}
	x, y := line.X+60, line.Y+line.H/2
	tt.Move(x, y)
	tt.Press(x, y)
	tt.Move(x+30, y+dy)
	tt.Release(x+30, y+dy)
}

func TestReplayPositionDragCreatesLongAndShortBrackets(t *testing.T) {
	for _, tc := range []struct {
		side  market.ReplaySide
		dy    float32
		label string
		kind  replayLineKind
	}{
		{market.ReplayBuy, -45, "止盈", replayTakeProfitLine},
		{market.ReplayBuy, 45, "止损", replayStopLossLine},
		{market.ReplaySell, -45, "止损", replayStopLossLine},
		{market.ReplaySell, 45, "止盈", replayTakeProfitLine},
	} {
		t.Run(replayPositionCaption(map[market.ReplaySide]float64{market.ReplayBuy: 1, market.ReplaySell: -1}[tc.side], locales.In(string(locales.Chinese)))+tc.label, func(t *testing.T) {
			app, tt := newReplayPositionTester(t, tc.side)
			line, _ := tt.Find("仓位线")
			x, y := line.X+60, line.Y+line.H/2
			initialStart := app.chart.Start
			tt.Move(x, y)
			if app.chart.hover != -1 {
				t.Fatal("hovering a position line retained the chart crosshair")
			}
			tt.Press(x, y)
			tt.Move(x+30, y+tc.dy)
			if _, ok := tt.Find(tc.label); !ok || !app.chart.replayLineDrag.Moved || app.chart.replayLineDrag.Target != tc.kind {
				t.Fatal("drag did not preview the expected bracket")
			}
			if app.chart.replayAccount.TakeProfit.Price != 0 || app.chart.replayAccount.StopLoss.Price != 0 {
				t.Fatal("preview placed an exit before pointer release")
			}
			draftPrice := app.chart.replayLineDrag.Price
			caption := fmt.Sprintf("%s %+.2f", tc.label, (draftPrice-100)*app.chart.replayAccount.Quantity)
			if _, ok := tt.Find(caption); !ok {
				t.Fatalf("preview missing P&L caption %q; texts: %v", caption, tt.Texts())
			}
			tt.Release(x+30, y+tc.dy)
			exit := app.chart.replayAccount.TakeProfit
			if tc.kind == replayStopLossLine {
				exit = app.chart.replayAccount.StopLoss
			}
			if math.Abs(exit.Price-draftPrice) > .0001 || !exit.ActiveAfter.Equal(app.candles[120].Time) || app.chart.replayLineDrag.Source != 0 {
				t.Fatalf("release did not commit the preview: exit=%+v draft=%v", exit, draftPrice)
			}
			if _, ok := tt.Find(caption); !ok {
				t.Fatalf("committed bracket missing P&L caption %q", caption)
			}
			if app.chart.Start != initialStart || app.chart.dragging || app.chart.annotationGesture != annotationGestureNone || len(app.chart.annotations) != 0 {
				t.Fatal("bracket drag panned the chart or created an annotation")
			}
		})
	}
}

func TestReplayBracketCaptionsTrackRemainingPositionPnL(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		side, closeSide           market.ReplaySide
		takeProfit, stopLoss, add float64
	}{
		{"long", market.ReplayBuy, market.ReplaySell, 120, 80, 110},
		{"short", market.ReplaySell, market.ReplayBuy, 80, 120, 90},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, tt := newReplayPositionTester(t, tc.side)
			app.replaySettings.Leverage = 3 // Leverage must not multiply P&L.
			stamp := app.candles[120].Time
			if err := app.chart.replayAccount.SetExit(market.ReplayTakeProfit, tc.takeProfit, stamp); err != nil {
				t.Fatal(err)
			}
			if err := app.chart.replayAccount.SetExit(market.ReplayStopLoss, tc.stopLoss, stamp); err != nil {
				t.Fatal(err)
			}
			previousProfit, previousLoss := "", ""
			assertCaptions := func(profit, loss string) {
				t.Helper()
				app.chart.replayOrderQuantity = 99 // Use held quantity, not the order input.
				tt.Frame()
				for _, caption := range []string{profit, loss} {
					if _, ok := tt.Find(caption); !ok {
						t.Fatalf("missing P&L caption %q; texts: %v", caption, tt.Texts())
					}
				}
				for _, caption := range []string{
					fmt.Sprintf("止盈 %g", tc.takeProfit), fmt.Sprintf("止损 %g", tc.stopLoss),
					previousProfit, previousLoss,
				} {
					if _, ok := tt.Find(caption); caption != "" && ok {
						t.Fatalf("price or stale P&L caption %q remained visible", caption)
					}
				}
				previousProfit, previousLoss = profit, loss
			}
			assertCaptions("止盈 +20.00", "止损 -20.00")
			app.candles[120].Close = tc.add
			app.chart.replayOrderQuantity = 1
			if _, err := app.placeReplayTrade(tc.side); err != nil {
				t.Fatal(err)
			}
			assertCaptions("止盈 +30.00", "止损 -50.00")
			app.candles[120].Close = 100
			app.chart.replayOrderQuantity = .5
			if _, err := app.placeReplayTrade(tc.closeSide); err != nil {
				t.Fatal(err)
			}
			if app.chart.replayAccount.RealizedPnL != -2.5 {
				t.Fatal("partial close did not realize the expected loss")
			}
			assertCaptions("止盈 +22.50", "止损 -37.50") // Exclude previously realized P&L.
		})
	}
}

func TestReplayBracketLinesAdjustCancelAndAlignTagsRight(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	dragReplayLine(t, tt, "仓位线", -45)
	dragReplayLine(t, tt, "仓位线", 45)
	originalTP := app.chart.replayAccount.TakeProfit.Price
	dragReplayLine(t, tt, "止盈", -25)
	if app.chart.replayAccount.TakeProfit.Price <= originalTP {
		t.Fatal("dragging an existing take-profit line did not update its price")
	}
	closePosition, _ := tt.Find("平仓")
	for _, label := range []string{"取消止盈", "取消止损"} {
		button, ok := tt.Find(label)
		if !ok || math.Abs(float64(button.X+button.W-closePosition.X-closePosition.W)) > .01 {
			t.Fatalf("%s tag is not right-aligned with the position tag: button=%+v position=%+v", label, button, closePosition)
		}
	}
	clickReplaySetting(t, tt, "取消止盈")
	if app.chart.replayAccount.TakeProfit.Price != 0 || app.chart.replayAccount.StopLoss.Price == 0 || app.chart.replayAccount.Quantity != 1 {
		t.Fatal("canceling take profit changed stop loss or closed the position")
	}
	if _, ok := tt.Find("止盈"); ok {
		t.Fatal("canceled take-profit line remained visible")
	}
	clickReplaySetting(t, tt, "取消止损")
	if app.chart.replayAccount.StopLoss.Price != 0 || len(app.chart.replayAccount.Trades) != 1 {
		t.Fatal("canceling stop loss executed a trade")
	}
}

func TestReplayLineClickEscapeAndInvalidAdjustmentKeepOrders(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	line, _ := tt.Find("仓位线")
	x, y := line.X+60, line.Y+line.H/2
	tt.ClickAt(x, y)
	if app.chart.replayAccount.TakeProfit.Price != 0 || app.chart.replayAccount.StopLoss.Price != 0 {
		t.Fatal("an undragged click created an exit")
	}
	tt.Press(x, y)
	tt.Move(x, y-45)
	tt.Key(0, ui.KeyEscape)
	tt.Release(x, y-45)
	if app.chart.replayLineDrag.Source != 0 || app.chart.replayAccount.TakeProfit.Price != 0 {
		t.Fatal("Escape committed the draft or retained a drag")
	}
	dragReplayLine(t, tt, "仓位线", -45)
	before := app.chart.replayAccount.TakeProfit
	dragReplayLine(t, tt, "止盈", 120) // Cross the entry line to the wrong side.
	if app.chart.replayAccount.TakeProfit != before || !tt.HasText("止盈止损价格无效") {
		t.Fatalf("invalid adjustment: before=%+v after=%+v message=%q", before, app.chart.replayAccount.TakeProfit, app.chart.replayTradeMessage)
	}
}

func TestReplayBracketPreviewWhilePlayingUsesReleaseCandle(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	clickReplaySetting(t, tt, "播放")
	app.chart.replayNextStep = time.Now().Add(time.Hour)
	line, _ := tt.Find("仓位线")
	x, y := line.X+60, line.Y+line.H/2
	tt.Press(x, y)
	tt.Move(x, y-45)
	price := app.chart.replayLineDrag.Price
	stamp := app.candles[121].Time
	app.candles[121] = Candle{Time: stamp, Open: 100, High: price + 2, Low: 95, Close: 100, EMA20: 100}
	app.advanceReplayStep()
	tt.Frame()
	if app.chart.replayAccount.Quantity != 1 || !app.chart.actionPlaying {
		t.Fatal("an uncommitted preview triggered or paused playback")
	}
	tt.Release(x, y-45)
	if !app.chart.replayAccount.TakeProfit.ActiveAfter.Equal(stamp) || !app.chart.actionPlaying {
		t.Fatal("release did not activate after the latest revealed candle")
	}
}

func TestReplayExitTriggersOnAdvanceAndKeepsPlaybackRunning(t *testing.T) {
	for _, side := range []market.ReplaySide{market.ReplayBuy, market.ReplaySell} {
		app, tt := newReplayPositionTester(t, side)
		stamp := app.candles[120].Time
		tp := 110.0
		if side == market.ReplaySell {
			tp = 90
		}
		if err := app.chart.replayAccount.SetExit(market.ReplayTakeProfit, tp, stamp); err != nil {
			t.Fatal(err)
		}
		stamp = app.candles[121].Time
		app.candles[121] = Candle{Time: stamp, Open: 100, High: 115, Low: 85, Close: 100, EMA20: 100}
		clickReplaySetting(t, tt, "播放")
		app.chart.replayNextStep = time.Now().Add(-time.Second)
		tt.Frame()
		account := app.chart.replayAccount
		if account.Quantity != 0 || len(account.Trades) != 2 || account.Trades[1].Price != tp || account.Trades[1].ExitKind != market.ReplayTakeProfit || app.chart.replayEnd != 122 || !app.chart.actionPlaying {
			t.Fatalf("replay step did not execute the bracket once: %+v", account)
		}
		if _, ok := tt.Find("仓位线"); ok || !tt.HasText("止盈 1 @") {
			t.Fatal("triggered exit retained the position line or omitted feedback")
		}
	}
}

func TestReplayTriggeredExitCancelsAnInFlightDrag(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	if err := app.chart.replayAccount.SetExit(market.ReplayTakeProfit, 105, app.candles[120].Time); err != nil {
		t.Fatal(err)
	}
	app.candles[121] = Candle{Time: app.candles[121].Time, Open: 100, High: 110, Low: 95, Close: 100, EMA20: 100}
	tt.Frame()
	line, _ := tt.Find("仓位线")
	x, y := line.X+60, line.Y+line.H/2
	tt.Press(x, y)
	tt.Move(x, y+45)
	app.chart.actionPlaying = true
	app.chart.replayNextStep = time.Now().Add(time.Hour)
	app.advanceReplayStep()
	tt.Frame()
	tt.Release(x, y+45)
	if app.chart.replayAccount.Quantity != 0 || app.chart.replayLineDrag.Source != 0 || app.chart.replayAccount.StopLoss.Price != 0 || len(app.chart.replayAccount.Trades) != 2 {
		t.Fatal("releasing a vanished position line restored a draft after automatic close")
	}
}

func TestReplayBracketTagsStayCompactAndRightAlignedAcrossViewports(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	dragReplayLine(t, tt, "仓位线", -45)
	dragReplayLine(t, tt, "仓位线", 45)
	for _, width := range []int{900, 420, 900} {
		tt.SetSize(width, 520)
		chart, _ := tt.Find("BTC/USDT K 线图")
		g := currentKlineGeometry(app)
		wantRight := chart.X + g.left + g.width - 15 // Shared tag inset plus border/padding.
		for _, pair := range [][2]string{{"平仓", "仓位线"}, {"取消止盈", "止盈"}, {"取消止损", "止损"}} {
			button, ok := tt.Find(pair[0])
			line, lineOK := tt.Find(pair[1])
			if !ok || !lineOK || button.X < chart.X || button.X+button.W > chart.X+chart.W || button.Y < chart.Y || button.Y+button.H > chart.Y+chart.H {
				t.Fatalf("%s is outside the resized chart at width %d: %+v", pair[0], width, button)
			}
			if button.W != 18 || button.H != 18 || math.Abs(float64(button.X+button.W-wantRight)) > .01 || math.Abs(float64(button.Y+button.H/2-line.Y-line.H/2)) > .01 {
				t.Fatalf("%s tag is not compact, centered on its line, and right-aligned at width %d: button=%+v line=%+v wantRight=%v", pair[0], width, button, line, wantRight)
			}
		}
	}
}

func TestReplayExitOnLastCandleAndInvalidCandleFailure(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		app, tt := newReplayPositionTester(t, market.ReplayBuy)
		end := len(app.candles)
		app.chart.replayEnd = end - 1
		stamp := app.candles[end-2].Time
		if err := app.chart.replayAccount.SetExit(market.ReplayStopLoss, 90, stamp); err != nil {
			t.Fatal(err)
		}
		last := Candle{Time: app.candles[end-1].Time, Open: 100, High: 115, Low: 85, Close: 100, EMA20: 100}
		if invalid {
			last.Low = math.NaN()
		}
		app.candles[end-1] = last
		tt.Frame()
		clickReplaySetting(t, tt, "播放")
		app.chart.replayNextStep = time.Now().Add(-time.Second)
		tt.Frame()
		if invalid {
			if app.chart.replayEnd != end-1 || app.chart.replayAccount.Quantity != 1 || app.chart.actionPlaying || app.chart.replayExitError == nil {
				t.Fatal("failed exit advanced past the invalid candle or mutated the position")
			}
		} else if !app.chart.replayComplete || app.chart.replayAccount.Quantity != 0 || app.chart.replayAccount.Trades[1].Price != 90 {
			t.Fatal("last candle completed replay before executing the stop loss")
		}
	}
}

func TestReplayPositionCloseSupportsKeyboardAndDisappearingFocus(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	// Tester.Focused also accepts a focused ancestor; reach the button itself,
	// not its focusable line container.
	for i := 0; i < 30 && (!tt.Focused("平仓") || tt.Focused("仓位线")); i++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("平仓") || tt.Focused("仓位线") {
		t.Fatal("position close button cannot be reached with Tab")
	}
	tt.Key(0, ui.KeySpace)
	if app.chart.replayAccount.Quantity != 0 || len(app.chart.replayAccount.Trades) != 2 {
		t.Fatal("keyboard activation did not close the position")
	}
	tt.Key(0, ui.KeySpace)
	for range 3 {
		tt.Frame()
	}
	if len(app.chart.replayAccount.Trades) != 2 {
		t.Fatal("focus from the disappearing close button activated another trade")
	}
}

func TestReplayBracketIdentitySurvivesHistoryPrepend(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	activeAfter := app.candles[120].Time
	if err := app.chart.replayAccount.SetExit(market.ReplayTakeProfit, 105, activeAfter); err != nil {
		t.Fatal(err)
	}
	nextStamp := app.candles[121].Time
	app.candles[121] = Candle{Time: nextStamp, Open: 100, High: 110, Low: 95, Close: 100, EMA20: 100}
	older := app.candles[0]
	older.Time = older.Time.Add(-time.Hour)
	app.applyMarketSnapshot(market.Snapshot{HistoryPrepended: 1, Candles: append([]Candle{older}, app.candles...)})
	tt.Frame()
	if app.chart.replayEnd != 122 || app.chart.replayAccount.TakeProfit.ActiveAfter != activeAfter {
		t.Fatal("history prepend changed bracket activation or the replay cutoff")
	}
	app.chart.actionPlaying = true
	app.advanceReplayStep()
	trade := app.chart.replayAccount.Trades[1]
	if trade.Price != 105 || !trade.Time.Equal(nextStamp) || app.chart.replayAccount.Quantity != 0 {
		t.Fatal("history prepend changed the candle used to trigger the bracket")
	}
}

func TestReplayCompletedPositionIsReadOnlyAndResetClearsBrackets(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplaySell)
	dragReplayLine(t, tt, "仓位线", -45)
	app.completeReplay()
	tt.Frame()
	clickReplaySetting(t, tt, "平仓")
	clickReplaySetting(t, tt, "取消止损")
	dragReplayLine(t, tt, "仓位线", 45)
	if app.chart.replayAccount.Quantity != -1 || len(app.chart.replayAccount.Trades) != 1 || app.chart.replayAccount.StopLoss.Price == 0 || app.chart.replayLineDrag.Source != 0 {
		t.Fatal("completed replay accepted a position edit")
	}
	clickReplaySetting(t, tt, "重置")
	for _, label := range []string{"仓位线", "止盈", "止损", "平仓"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("reset retained %s", label)
		}
	}
}

func TestReplayEndClearsBracketDragAndAllLines(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	dragReplayLine(t, tt, "仓位线", -45)
	dragReplayLine(t, tt, "仓位线", 45)
	line, _ := tt.Find("仓位线")
	x, y := line.X+60, line.Y+line.H/2
	tt.Press(x, y)
	tt.Move(x, y-60)
	app.chart.endReplay(len(app.candles))
	tt.Frame()
	tt.Release(x, y-60)
	for range 3 {
		tt.Frame()
	}
	for _, label := range []string{"仓位线", "止盈", "止损", "平仓"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("%s survived End", label)
		}
	}
	if app.chart.replayLineDrag.Source != 0 || len(app.chart.replayAccount.Trades) != 0 {
		t.Fatal("ending replay retained a gesture or restored an old bracket")
	}
}
