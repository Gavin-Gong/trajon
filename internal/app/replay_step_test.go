package app

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

func TestReplayRightStepsOneCandleAndStaysPaused(t *testing.T) {
	app, tt := newKlineToolbarTester()
	const start = 120
	app.chart.selectReplayStart(start, len(app.candles))
	app.chart.Visible = 40
	app.chart.Start = start - app.chart.Visible + 1
	app.chart.replaySpeed = 5
	app.chart.replayOrderQuantity = 0 // Stepping is independent of order availability.
	app.chart.chartHandle.Focus()
	tt.Frame()

	for i := 1; i <= 3; i++ {
		tt.Key(0, ui.KeyRight)
		if app.chart.replayEnd != start+1+i || app.chart.Start != start-app.chart.Visible+1+i {
			t.Fatalf("Right did not reveal and follow exactly one candle: cutoff=%d viewport=%d", app.chart.replayEnd, app.chart.Start)
		}
		if app.chart.actionPlaying || !app.chart.replayNextStep.IsZero() || !app.chart.replaySessionStarted || app.chart.replaySpeed != 5 {
			t.Fatal("single stepping started a timer or changed playback speed")
		}
		for range 3 {
			tt.Frame()
		}
		if app.chart.replayEnd != start+1+i {
			t.Fatal("extra frames advanced paused replay")
		}
	}
	end := app.chart.replayEnd
	tt.Key(ui.Shift, ui.KeyRight)
	tt.Key(ui.Cmd, ui.KeyRight)
	if app.chart.replayEnd != end {
		t.Fatal("modified Right activated single stepping")
	}

	clickReplaySetting(t, tt, app.text().Play())
	app.chart.replayNextStep = time.Now().Add(time.Hour)
	nextStep := app.chart.replayNextStep
	tt.Key(0, ui.KeyRight)
	if app.chart.replayEnd != end || !app.chart.actionPlaying || !app.chart.replayNextStep.Equal(nextStep) {
		t.Fatal("Right while playing advanced or rescheduled playback")
	}
	tt.Key(0, ui.KeySpace)
	tt.Key(0, ui.KeyRight) // The playback button retains focus after pausing.
	if app.chart.replayEnd != end+1 || app.chart.actionPlaying || !app.chart.replayNextStep.IsZero() {
		t.Fatal("Right on the paused playback button did not single-step")
	}
	clickReplaySetting(t, tt, app.text().Reset())
	tt.Key(0, ui.KeyRight)
	if app.chart.replayEnd != start+2 || app.chart.actionPlaying {
		t.Fatal("Right did not single-step after reset")
	}
}

func TestReplayRightIgnoresUnavailableSessions(t *testing.T) {
	for _, mode := range []string{"no start", "selection", "completed", "discarded", "loading", "summary", "empty"} {
		t.Run(mode, func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			app.chart.selectReplayStart(120, len(app.candles))
			switch mode {
			case "no start":
				app.chart.replayHasStart = false
			case "selection":
				app.chart.replaySelecting = true
			case "completed":
				app.completeReplay()
			case "discarded":
				app.chart.endReplay(len(app.candles))
			case "loading":
				app.marketLoading = true
			case "summary":
				app.openReplaySummary()
			case "empty":
				app.candles = nil
			}
			app.chart.chartHandle.Focus()
			tt.Frame()
			end := app.chart.replayEnd
			tt.Key(0, ui.KeyRight)
			if app.chart.replayEnd != end || app.chart.actionPlaying {
				t.Fatalf("Right advanced an unavailable replay: cutoff %d -> %d, playing=%v", end, app.chart.replayEnd, app.chart.actionPlaying)
			}
		})
	}
}

func TestReplayRightChecksExitsBeforeCompletingLastCandle(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	end := len(app.candles)
	app.chart.replayEnd = end - 1
	if err := app.chart.replayAccount.SetExit(market.ReplayStopLoss, 90, app.candles[end-2].Time); err != nil {
		t.Fatal(err)
	}
	app.candles[end-1] = Candle{Time: app.candles[end-1].Time, Open: 100, High: 115, Low: 85, Close: 100, EMA20: 100}
	app.chart.chartHandle.Focus()
	tt.Frame()
	tt.Key(0, ui.KeyRight)
	account := app.chart.replayAccount
	if !app.chart.replayComplete || app.chart.replayEnd != end || app.chart.actionPlaying || !app.chart.replayNextStep.IsZero() {
		t.Fatal("last single step did not complete paused replay")
	}
	if account.Quantity != 0 || len(account.Trades) != 2 || account.Trades[1].Price != 90 || account.Trades[1].ExitKind != market.ReplayStopLoss {
		t.Fatalf("last single step did not execute the stop loss: %+v", account)
	}
	if !tt.HasText("止损 1 @") || !tt.HasText(app.text().ReplayComplete()) {
		t.Fatal("single step omitted exit feedback or the completion toast")
	}
	tt.Key(0, ui.KeyRight)
	if app.chart.replayEnd != end || len(app.chart.replayAccount.Trades) != 2 {
		t.Fatal("Right advanced or duplicated fills after replay completion")
	}
}

func TestReplayRightReportsExitFailureAndCanRetry(t *testing.T) {
	app, tt := newReplayPositionTester(t, market.ReplayBuy)
	if err := app.chart.replayAccount.SetExit(market.ReplayTakeProfit, 110, app.candles[120].Time); err != nil {
		t.Fatal(err)
	}
	app.candles[121] = Candle{Time: app.candles[121].Time, Open: 100, High: 115, Low: math.NaN(), Close: 100, EMA20: 100}
	app.chart.chartHandle.Focus()
	tt.Frame()
	tt.Key(0, ui.KeyRight)
	if app.chart.replayEnd != 121 || app.chart.replayAccount.Quantity != 1 || len(app.chart.replayAccount.Trades) != 1 || app.chart.replayExitError == nil || !tt.HasText(app.text().ExitFailed()) {
		t.Fatal("failed single step advanced or mutated the position without reporting the error")
	}
	app.candles[121].Low = 95
	tt.Key(0, ui.KeyRight)
	if app.chart.replayEnd != 122 || app.chart.replayExitError != nil || app.chart.replayAccount.Quantity != 0 || len(app.chart.replayAccount.Trades) != 2 || app.chart.replayAccount.Trades[1].Price != 110 {
		t.Fatal("single step could not retry the failed candle")
	}
	if app.chart.actionPlaying || !app.chart.replayNextStep.IsZero() || !tt.HasText("止盈 1 @") {
		t.Fatal("successful retry started playback or omitted its fill feedback")
	}
}
