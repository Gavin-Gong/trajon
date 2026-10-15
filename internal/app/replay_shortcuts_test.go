package app

import (
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"trajon/internal/locales"
	"trajon/internal/market"
)

func TestReplayShortcutsPlayAndTradeFromChart(t *testing.T) {
	app, tt := newKlineToolbarTester()
	const start = 120
	app.candles[start].Close = 100
	app.chart.selectReplayStart(start, len(app.candles))
	app.chart.replayOrderQuantity = .5
	app.chart.chartHandle.Focus()
	tt.Frame()

	tt.Key(0, ui.KeySpace)
	if !app.chart.actionPlaying || app.chart.replayNextStep.IsZero() {
		t.Fatal("Space did not start playback from the chart")
	}
	app.chart.replayNextStep = time.Now().Add(time.Hour)
	nextStep := app.chart.replayNextStep
	app.chart.hover = start - 20
	app.chart.Start = 500
	for i, key := range []ui.Key{ui.KeyUp, ui.KeyDown, ui.KeyDown} {
		tt.Key(0, key)
		if len(app.chart.replayAccount.Trades) != i+1 {
			t.Fatalf("key %v did not submit exactly one order", key)
		}
		trade := app.chart.replayAccount.Trades[i]
		wantSide := market.ReplaySell
		if key == ui.KeyUp {
			wantSide = market.ReplayBuy
		}
		if trade.Side != wantSide || trade.Quantity != .5 || trade.Price != 100 || !trade.Time.Equal(app.candles[start].Time) {
			t.Fatalf("shortcut fill = %+v, want side %v, quantity .5 at the latest revealed close", trade, wantSide)
		}
		if !app.chart.actionPlaying || !app.chart.replayNextStep.Equal(nextStep) {
			t.Fatal("trading shortcut paused or rescheduled playback")
		}
	}
	if app.chart.replayAccount.Quantity != -.5 {
		t.Fatal("Down did not open a short after closing the long")
	}
	tt.Key(0, ui.KeySpace)
	if app.chart.actionPlaying || !app.chart.replayNextStep.IsZero() {
		t.Fatal("Space did not pause playback")
	}
	tt.Key(0, ui.KeyUp)
	if app.chart.replayAccount.Quantity != 0 || app.chart.actionPlaying {
		t.Fatal("Up did not close the short while retaining paused playback")
	}
	for range 3 {
		tt.Frame()
	}
	if len(app.chart.replayAccount.Trades) != 4 {
		t.Fatal("extra frames duplicated a shortcut order")
	}
	for _, key := range []ui.Key{ui.KeySpace, ui.KeyUp, ui.KeyDown} {
		tt.Key(ui.Shift, key)
		tt.Key(ui.Cmd, key)
	}
	if app.chart.actionPlaying || len(app.chart.replayAccount.Trades) != 4 {
		t.Fatal("modified keys activated unmodified replay shortcuts")
	}
}

func TestReplayShortcutsWorkAfterClickingActionButtons(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.selectReplayStart(120, len(app.candles))
	app.candles[120].Close = 100
	tt.Frame()
	clickReplaySetting(t, tt, app.text().Buy())
	tt.Key(0, ui.KeySpace)
	if !app.chart.actionPlaying || len(app.chart.replayAccount.Trades) != 1 {
		t.Fatal("Space on the buy button should play, not buy again")
	}
	tt.Key(0, ui.KeyDown)
	if app.chart.replayAccount.Quantity != 0 || len(app.chart.replayAccount.Trades) != 2 {
		t.Fatal("Down on the buy button did not sell exactly once")
	}
	clickReplaySetting(t, tt, app.text().Sell())
	tt.Key(0, ui.KeySpace)
	if app.chart.actionPlaying || len(app.chart.replayAccount.Trades) != 3 {
		t.Fatal("Space on the sell button should pause, not sell again")
	}
	tt.Key(0, ui.KeyUp)
	if app.chart.replayAccount.Quantity != 0 || len(app.chart.replayAccount.Trades) != 4 {
		t.Fatal("Up on the sell button did not buy exactly once")
	}
}

func TestReplayShortcutsRespectSessionAvailability(t *testing.T) {
	for _, mode := range []string{"no start", "selection", "completed", "discarded", "zero quantity", "loading"} {
		t.Run(mode, func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			app.candles[120].Close = 100
			if mode != "no start" && mode != "selection" {
				app.chart.selectReplayStart(120, len(app.candles))
			}
			switch mode {
			case "selection":
				app.chart.replaySelecting = true
			case "completed":
				app.completeReplay()
			case "discarded":
				app.chart.endReplay(len(app.candles))
			case "zero quantity":
				app.chart.replayOrderQuantity = 0
			case "loading":
				app.marketLoading = true
			}
			app.chart.chartHandle.Focus()
			tt.Frame()
			tt.Key(0, ui.KeyUp)
			tt.Key(0, ui.KeyDown)
			if len(app.chart.replayAccount.Trades) != 0 {
				t.Fatal("unavailable trading shortcut submitted an order")
			}
			if mode != "zero quantity" && mode != "loading" {
				tt.Key(0, ui.KeySpace)
				if app.chart.actionPlaying {
					t.Fatal("unavailable playback shortcut started replay")
				}
			}
		})
	}
}

func TestReplayShortcutsLeaveEditorsAndSpeedSelectionAlone(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.selectReplayStart(120, len(app.candles))
	tt.Frame()
	end := app.chart.replayEnd
	clickReplaySetting(t, tt, app.text().OrderQuantity())
	replaceReplaySetting(tt, "0.25")
	tt.Key(0, ui.KeyUp)
	if app.chart.replayOrderQuantity != 1.25 {
		t.Fatal("Up did not step the focused quantity input")
	}
	tt.Key(0, ui.KeyDown)
	if app.chart.replayOrderQuantity != .25 {
		t.Fatal("Down did not step the focused quantity input")
	}
	tt.Key(0, ui.KeySpace)
	tt.Key(0, ui.KeyRight)
	if app.chart.actionPlaying || app.chart.replayEnd != end || len(app.chart.replayAccount.Trades) != 0 {
		t.Fatal("quantity editing triggered replay shortcuts")
	}

	app.chart.Start = 100
	beginAnnotationTextEdit(&app.chart, 0, annotationAnchor{Time: app.candles[120].Time, Price: app.candles[120].Close}, true)
	tt.Frame()
	tt.Type("draft")
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyDown)
	tt.TypeKey(0, ui.KeySpace, " ")
	tt.Key(0, ui.KeyRight)
	if app.chart.textDraft != "draft " || app.chart.actionPlaying || app.chart.replayEnd != end || len(app.chart.replayAccount.Trades) != 0 {
		t.Fatalf("annotation editing triggered replay shortcuts or lost its draft: %q", app.chart.textDraft)
	}
	finishAnnotationTextEdit(&app.chart, false)
	tt.Frame()

	clickReplaySetting(t, tt, app.text().Play())
	tt.Key(0, ui.KeySpace)
	// From play, Tab focuses the speed selector without opening it.
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeySpace)
	if !tt.HasText("0.5×") || app.chart.actionPlaying || app.chart.replayEnd != end {
		t.Fatal("speed selector keys changed playback instead of opening the menu")
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyRight)
	if len(app.chart.replayAccount.Trades) != 0 || app.chart.replayEnd != end {
		t.Fatal("speed menu navigation submitted an order or advanced playback")
	}
}

func TestReplayShortcutsDoNotActBehindSummary(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.selectReplayStart(120, len(app.candles))
	app.candles[120].Close = 100
	tt.Frame()
	clickReplaySetting(t, tt, app.text().Buy())
	clickReplaySetting(t, tt, app.text().End())
	before := app.chart.replayAccount
	end := app.chart.replayEnd
	for _, key := range []ui.Key{ui.KeySpace, ui.KeyUp, ui.KeyDown, ui.KeyRight} {
		tt.Key(0, key)
	}
	if !app.chart.replaySummaryOpen || app.chart.actionPlaying || app.chart.replayEnd != end || !reflect.DeepEqual(before, app.chart.replayAccount) {
		t.Fatal("replay shortcuts changed the session behind its summary")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Key(0, ui.KeyUp) // Focus returns to End in the action bar.
	if len(app.chart.replayAccount.Trades) != 2 {
		t.Fatal("trading shortcut did not return after dismissing the summary")
	}
	tt.Key(0, ui.KeySpace)
	if !app.chart.actionPlaying || app.chart.replaySummaryOpen {
		t.Fatal("Space on the restored End button should resume playback, not reopen the summary")
	}
}

func TestReplayShortcutTooltipsAreLocalized(t *testing.T) {
	for _, language := range []locales.Language{locales.Chinese, locales.English} {
		t.Run(string(language), func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			app.language = language
			app.chart.selectReplayStart(120, len(app.candles))
			tt.Frame()
			for _, tc := range []struct{ label, key string }{
				{app.text().Play(), "Space"},
				{app.text().Pause(), "Space"},
				{app.text().Buy(), "↑"},
				{app.text().Sell(), "↓"},
			} {
				if tc.label == app.text().Pause() {
					tt.Key(0, ui.KeySpace)
				}
				want := tc.label + " (" + tc.key + ")"
				for i := 0; i < 40 && !tt.HasText(want); i++ {
					tt.Key(0, ui.KeyTab)
				}
				if !tt.HasText(want) || !tt.Focused(tc.label) {
					t.Fatalf("missing focused tooltip %q; texts=%q", want, tt.Texts())
				}
			}
		})
	}
}
