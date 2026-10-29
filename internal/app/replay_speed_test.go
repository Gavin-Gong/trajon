package app

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

func TestReplaySpeedIntervals(t *testing.T) {
	for _, tc := range []struct {
		speed float64
		want  time.Duration
	}{
		{.5, 2 * time.Second},
		{1, time.Second},
		{2, 500 * time.Millisecond},
		{5, 200 * time.Millisecond},
		{10, 100 * time.Millisecond},
		{0, time.Second},
		{-1, time.Second},
		{3, time.Second},
		{math.NaN(), time.Second},
		{math.Inf(1), time.Second},
	} {
		if got := replayInterval(tc.speed); got != tc.want {
			t.Errorf("speed %v: interval = %v, want %v", tc.speed, got, tc.want)
		}
	}
}

func TestReplaySpeedChangeReschedulesOnlyPlayingReplay(t *testing.T) {
	now := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	for _, playing := range []bool{false, true} {
		for _, speed := range replaySpeeds {
			state := newKlineState(200)
			state.replaySpeed = 2
			state.actionPlaying = playing
			state.replayEnd = 121
			if playing {
				state.replayNextStep = now.Add(time.Hour)
			}
			before := state.replayNextStep
			state.setReplaySpeed(speed, now)
			want := before
			if playing && speed != 2 {
				want = now.Add(replayInterval(speed))
			}
			if state.replaySpeed != speed || state.actionPlaying != playing || state.replayNextStep != want || state.replayEnd != 121 {
				t.Fatalf("playing=%v speed=%v: got speed=%v playing=%v deadline=%v end=%d", playing, speed, state.replaySpeed, state.actionPlaying, state.replayNextStep, state.replayEnd)
			}
		}
	}

	state := newKlineState(200)
	state.actionPlaying = true
	state.replayNextStep = now.Add(time.Second)
	for _, speed := range []float64{0, -1, 3, math.NaN(), math.Inf(1)} {
		state.setReplaySpeed(speed, now)
		if state.replaySpeed != 1 || state.replayNextStep != now.Add(time.Second) {
			t.Fatalf("invalid speed %v changed playback", speed)
		}
	}
}

func TestReplaySpeedSchedulesStartAndEveryFrame(t *testing.T) {
	for _, speed := range replaySpeeds {
		app := &application{
			candles:        demoCandles(),
			chart:          newKlineState(200),
			replaySettings: defaultReplaySettings(),
		}
		app.chart.selectReplayStart(120, len(app.candles))
		app.chart.replaySpeed = speed
		start := true
		var now time.Time
		tt := ui.NewTester(func(c *ui.Context) {
			now = c.Now()
			if start {
				start = false
				app.startReplay(c)
			}
			app.advanceReplayFrame(c)
		}, 100, 100)
		interval := replayInterval(speed)
		if app.chart.replayNextStep != now.Add(interval) || app.chart.replayEnd != 121 || !app.chart.actionPlaying {
			t.Fatalf("speed %v: start did not schedule %v", speed, interval)
		}

		app.chart.replayNextStep = time.Time{}
		tt.Frame()
		if app.chart.replayNextStep != now.Add(interval) || app.chart.replayEnd != 121 {
			t.Fatalf("speed %v: missing deadline was not initialized with %v", speed, interval)
		}

		app.chart.replayNextStep = now.Add(-time.Hour)
		tt.Frame()
		if app.chart.replayNextStep != now.Add(interval) || app.chart.replayEnd != 122 {
			t.Fatalf("speed %v: due frame did not reveal exactly one candle and schedule %v", speed, interval)
		}

		app.chart.pauseReplay()
		tt.Frame()
		if !app.chart.replayNextStep.IsZero() || app.chart.replayEnd != 122 || app.chart.replaySpeed != speed {
			t.Fatalf("speed %v: pause changed speed or advanced replay", speed)
		}

		start = true
		tt.Frame()
		if app.chart.replayNextStep != now.Add(interval) || app.chart.replayEnd != 122 {
			t.Fatalf("speed %v: resume did not schedule %v", speed, interval)
		}
	}
}

func TestReplaySpeedSelectorVisibilityLayoutAndLifecycle(t *testing.T) {
	app, tt := newReplaySettingsTester()
	if app.chart.replaySpeed != 1 {
		t.Fatal("playback speed did not default to 1×")
	}
	if _, ok := tt.Find("播放速度"); ok {
		t.Fatal("speed selector was shown before a replay start")
	}
	clickReplaySetting(t, tt, "回放起点")
	if _, ok := tt.Find("播放速度"); ok {
		t.Fatal("speed selector was shown during start selection")
	}
	app.chart.selectReplayStart(120, len(app.candles))
	tt.Frame()
	bar, barOK := tt.Find("K 线图操作栏")
	selector, selectorOK := tt.Find("播放速度")
	if !barOK || !selectorOK || selector.W != klineReplaySpeedWidth || selector.H != 28 ||
		selector.X < bar.X || selector.Y < bar.Y || selector.X+selector.W > bar.X+bar.W || selector.Y+selector.H > bar.Y+bar.H {
		t.Fatalf("speed selector does not fit the action bar: bar=%+v selector=%+v", bar, selector)
	}
	play, playOK := tt.Find("播放")
	quantity, quantityOK := tt.Find("下单数量")
	if !playOK || !quantityOK || play.X+play.W > selector.X || selector.X+selector.W > quantity.X {
		t.Fatal("speed selector is not between play and quantity")
	}

	for i, label := range []string{"0.5×", "1×", "2×", "5×", "10×"} {
		clickReplaySetting(t, tt, "播放速度")
		for _, option := range []string{"0.5×", "1×", "2×", "5×", "10×"} {
			if !tt.HasText(option) {
				t.Fatalf("speed menu is missing %s", option)
			}
		}
		clickReplaySetting(t, tt, label)
		tt.Frame()
		if app.chart.replaySpeed != replaySpeeds[i] || !tt.HasText(label) || app.chart.actionPlaying || !app.chart.replayNextStep.IsZero() || app.chart.replayEnd != 121 || app.chart.dragging {
			t.Fatalf("selecting %s did not update speed without changing replay", label)
		}
		if !tt.Focused("播放速度") {
			t.Fatal("choosing speed lost trigger focus")
		}
	}

	clickReplaySetting(t, tt, "播放速度")
	tt.Key(0, ui.KeyEscape)
	if tt.HasText("0.5×") || app.chart.replaySpeed != 10 || !tt.Focused("播放速度") {
		t.Fatal("Escape did not dismiss the speed menu without changing speed")
	}
	clickReplaySetting(t, tt, "播放速度")
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyEnter)
	if app.chart.replaySpeed != .5 || !tt.HasText("0.5×") || tt.HasText("10×") {
		t.Fatal("keyboard selection did not choose 0.5× and close the menu")
	}

	clickReplaySetting(t, tt, "播放速度")
	app.chart.endReplay(len(app.candles))
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("播放速度"); ok {
		t.Fatal("ending the session retained the speed selector")
	}
	app.chart.selectReplayStart(130, len(app.candles))
	tt.Frame()
	if app.chart.replaySpeed != .5 || !tt.HasText("0.5×") || tt.HasText("10×") {
		t.Fatal("new replay lost speed or reopened the old menu")
	}
}

func TestReplaySpeedChangesDuringPlaybackWithoutResettingSession(t *testing.T) {
	app, tt := newReplaySettingsTester()
	app.chart.selectReplayStart(120, len(app.candles))
	tt.Frame()
	clickReplaySetting(t, tt, "买入")
	clickReplaySetting(t, tt, "播放")
	app.chart.replayNextStep = tt.Now().Add(time.Hour)
	viewport := app.chart.Viewport
	clickReplaySetting(t, tt, "播放速度")
	clickReplaySetting(t, tt, "5×")
	deadline := app.chart.replayNextStep
	if !deadline.Equal(tt.Now().Add(200 * time.Millisecond)) {
		t.Fatalf("speed change did not reschedule the next candle at 5×: %v", deadline)
	}
	if !app.chart.actionPlaying || app.chart.replayEnd != 121 || app.chart.Viewport != viewport || app.chart.replayAccount.Quantity != 1 || len(app.chart.replayAccount.Trades) != 1 || app.chart.dragging {
		t.Fatal("changing speed paused, advanced, panned, or cleared the replay session")
	}

	clickReplaySetting(t, tt, "暂停")
	clickReplaySetting(t, tt, "播放速度")
	clickReplaySetting(t, tt, "0.5×")
	clickReplaySetting(t, tt, "播放")
	deadline = app.chart.replayNextStep
	if !deadline.Equal(tt.Now().Add(2 * time.Second)) {
		t.Fatalf("resume did not use the speed selected while paused: %v", deadline)
	}
}

func TestCompletedReplayDisablesSpeedSelector(t *testing.T) {
	app, tt := newReplaySettingsTester()
	app.chart.selectReplayStart(120, len(app.candles))
	app.chart.replaySpeed = 5
	app.completeReplay()
	tt.Frame()
	clickReplaySetting(t, tt, "播放速度")
	if tt.HasText("0.5×") || app.chart.replaySpeed != 5 || app.chart.actionPlaying {
		t.Fatal("completed replay allowed speed changes")
	}
	clickReplaySetting(t, tt, "重置")
	clickReplaySetting(t, tt, "播放速度")
	clickReplaySetting(t, tt, "2×")
	if app.chart.replaySpeed != 2 || app.chart.replayComplete {
		t.Fatal("reset did not re-enable speed changes")
	}

	clickReplaySetting(t, tt, "播放速度")
	app.completeReplay()
	tt.Frame()
	clickReplaySetting(t, tt, "0.5×")
	if app.chart.replaySpeed != 2 || app.chart.actionPlaying {
		t.Fatal("an already-open menu changed speed after replay completed")
	}
	tt.Key(0, ui.KeyEscape)
	if tt.HasText("0.5×") {
		t.Fatal("completed replay trapped the open speed menu")
	}
}

func TestReplaySpeedSurvivesSessionAndMarketChanges(t *testing.T) {
	app, tt := newReplaySettingsTester()
	app.chart.selectReplayStart(120, len(app.candles))
	tt.Frame()
	clickReplaySetting(t, tt, "播放速度")
	clickReplaySetting(t, tt, "5×")
	clickReplaySetting(t, tt, "播放")
	clickReplaySetting(t, tt, "重置")
	if app.chart.replaySpeed != 5 {
		t.Fatal("reset lost playback speed")
	}
	clickReplaySetting(t, tt, "结束")
	clickReplaySetting(t, tt, "放弃")
	if app.chart.replaySpeed != 5 {
		t.Fatal("discarding the session lost playback speed")
	}

	for _, interval := range []string{"4h", "1d"} {
		app.applyMarketSnapshot(market.Snapshot{
			Instrument: app.currentInstrument,
			Interval:   interval,
			Candles:    demoCandles(),
		})
		if app.chart.replaySpeed != 5 || app.chart.timeframe != interval {
			t.Fatal("committed timeframe change lost playback speed")
		}
	}
	item := app.currentInstrument
	item.Symbol = "ETH/USDT"
	app.applyMarketSnapshot(market.Snapshot{Instrument: item, Interval: "1d", Candles: demoCandles()})
	if app.chart.replaySpeed != 5 || app.currentInstrument != item {
		t.Fatal("committed instrument change lost playback speed")
	}
	app.chart.selectReplayStart(130, len(app.candles))
	tt.Frame()
	if !tt.HasText("5×") || tt.HasText("10×") {
		t.Fatal("new replay did not display the retained speed with a closed menu")
	}
}
