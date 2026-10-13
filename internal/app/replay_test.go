package app

import (
	"bytes"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestReplaySelectionMaskFollowsPointerBeforeAndDuringDrag(t *testing.T) {
	app, tt := newKlineToolbarTester()
	g := currentKlineGeometry(app)
	x1, x2 := g.left+g.width*.3, g.left+g.width*.65
	y := g.top + g.priceHeight*.5
	probeY := int(g.top + 5) // Empty space above the candles and below the top grid line.
	leftX, middleX, rightX := int(g.left+g.width*.1), int(g.left+g.width*.45), int(g.left+g.width*.9)
	tt.Move(x1, y)
	before := tt.Image()
	left, middle, right := before.RGBAAt(leftX, probeY), before.RGBAAt(middleX, probeY), before.RGBAAt(rightX, probeY)

	if err := tt.Click("回放起点"); err != nil {
		t.Fatal(err)
	}
	if tt.Image().RGBAAt(rightX, probeY) == right {
		t.Fatal("clicking the replay start button did not immediately show the gray mask")
	}
	tt.Move(x1, y)
	preview := tt.Image()
	if preview.RGBAAt(leftX, probeY) != left {
		t.Fatal("replay selection mask covered the left side of the pointer")
	}
	if preview.RGBAAt(middleX, probeY) == middle || preview.RGBAAt(rightX, probeY) == right {
		t.Fatal("replay selection did not shade the right side before dragging")
	}
	marker := preview.RGBAAt(int(x1), probeY)
	if marker == left || marker == preview.RGBAAt(rightX, probeY) {
		t.Fatal("selection preview did not draw a distinct vertical boundary at the pointer")
	}

	tt.Move(x2, y)
	preview = tt.Image()
	if preview.RGBAAt(middleX, probeY) != middle || preview.RGBAAt(rightX, probeY) == right {
		t.Fatal("moving the pointer did not move the gray mask boundary")
	}

	tt.Press(x2, y)
	tt.Move(x2+40, y)
	preview = tt.Image()
	unmaskedX := int(x2 + 20)
	if preview.RGBAAt(unmaskedX, probeY) != left || preview.RGBAAt(rightX, probeY) == right {
		t.Fatal("dragging did not move the gray mask boundary with the pointer")
	}
	tt.Release(x2+40, y)
	if !app.chart.replaySelecting || app.chart.replayHasStart {
		t.Fatal("drag release committed the preview instead of remaining in selection mode")
	}
}

func TestReplaySelectionOnlyCommitsAnUndraggedClick(t *testing.T) {
	for _, tc := range []struct {
		name       string
		dx, dy     float32
		returnBack bool
		oldest     bool
		wantSelect bool
	}{
		{name: "click", wantSelect: true},
		{name: "small pointer jitter", dx: 2, dy: 1, wantSelect: true},
		{name: "horizontal drag", dx: 40},
		{name: "vertical drag", dy: 20},
		{name: "drag returning to press location", dx: 40, returnBack: true},
		{name: "drag at oldest history boundary", dx: 40, oldest: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			if tc.oldest {
				app.chart.Start = 0
				tt.Frame()
			}
			if err := tt.Click("回放起点"); err != nil {
				t.Fatal(err)
			}
			g := currentKlineGeometry(app)
			x, y := g.left+g.width*.4, g.top+g.priceHeight*.5
			initialStart := app.chart.Start
			tt.Press(x, y)
			tt.Move(x+tc.dx, y+tc.dy)
			if tc.returnBack {
				tt.Move(x, y)
			}
			endX, endY := x+tc.dx, y+tc.dy
			if tc.returnBack {
				endX, endY = x, y
			}
			wantIndex := replayCandleIndexAtX(&app.chart, app.candles, endX)
			tt.Release(endX, endY)
			if app.chart.replayHasStart != tc.wantSelect || app.chart.replaySelecting == tc.wantSelect {
				t.Fatalf("after release: selected=%v selecting=%v, want selected=%v", app.chart.replayHasStart, app.chart.replaySelecting, tc.wantSelect)
			}
			if tc.wantSelect && (app.chart.replayStart != wantIndex || app.chart.Start != initialStart) {
				t.Fatalf("click selected candle %d at viewport %d, want candle %d at viewport %d", app.chart.replayStart, app.chart.Start, wantIndex, initialStart)
			}
			if app.chart.dragging || app.chart.replayDragging {
				t.Fatal("releasing left the chart in a drag gesture")
			}
		})
	}
}

func TestReplaySelectionCanRestartAfterEndingReplay(t *testing.T) {
	app, tt := newKlineToolbarTester()
	oldStart := app.chart.Start + app.chart.Visible/4
	app.chart.selectReplayStart(oldStart, len(app.candles))
	tt.Frame()
	if _, ok := tt.Find("回放起点"); ok {
		t.Fatal("confirmed replay still offers start selection")
	}
	if err := tt.Click("结束"); err != nil {
		t.Fatal(err)
	}
	clickReplaySetting(t, tt, "放弃")
	if err := tt.Click("回放起点"); err != nil {
		t.Fatal(err)
	}
	if got, want := app.chart.visibleEnd(len(app.candles)), app.chart.Start+app.chart.Visible; got != want {
		t.Fatalf("new selection retained the previous future-candle cutoff: %d, want %d", got, want)
	}
	g := currentKlineGeometry(app)
	x, y := g.left+g.width*.75, g.top+g.priceHeight*.5
	tt.Press(x, y)
	tt.Move(x+30, y)
	tt.Release(x+30, y)
	if app.chart.replayHasStart || !app.chart.replaySelecting {
		t.Fatal("browsing history committed a new replay start")
	}

	tt.Key(0, ui.KeyEscape)
	if app.chart.replayHasStart || app.chart.visibleEnd(len(app.candles)) != app.chart.Start+app.chart.Visible {
		t.Fatal("cancelling selection restored the ended replay cutoff")
	}
	if err := tt.Click("回放起点"); err != nil {
		t.Fatal(err)
	}
	wantIndex := replayCandleIndexAtX(&app.chart, app.candles, x)
	tt.ClickAt(x, y)
	if app.chart.replaySelecting || app.chart.replayStart != wantIndex || wantIndex <= oldStart {
		t.Fatal("clicking a future candle did not confirm a new replay start")
	}
}

func TestReplaySelectionEscapeCancelsAnActiveDrag(t *testing.T) {
	app, tt := newKlineToolbarTester()
	if err := tt.Click("回放起点"); err != nil {
		t.Fatal(err)
	}
	g := currentKlineGeometry(app)
	x, y := g.left+g.width*.4, g.top+g.priceHeight*.5
	tt.Press(x, y)
	tt.Move(x+30, y)
	tt.Key(0, ui.KeyEscape)
	startAfterEscape := app.chart.Start
	tt.Move(x+60, y)
	tt.Release(x+60, y)
	if app.chart.replaySelecting || app.chart.replayDragging || app.chart.dragging || app.chart.replayHasStart || app.chart.Start != startAfterEscape {
		t.Fatal("Escape did not cancel the selection gesture before the remaining move and release")
	}
}

func TestReplaySelectionSmallMovesAccumulatePanAtWideCandleScale(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.Visible = minVisibleCandles
	tt.Frame()
	if err := tt.Click("回放起点"); err != nil {
		t.Fatal(err)
	}
	g := currentKlineGeometry(app)
	x, y := g.left+g.width*.4, g.top+g.priceHeight*.5
	initialStart := app.chart.Start
	tt.Press(x, y)
	for dx := float32(5); dx <= 50; dx += 5 {
		tt.Move(x+dx, y)
	}
	tt.Release(x+50, y)
	if app.chart.Start >= initialStart {
		t.Fatal("small pointer moves did not accumulate into a candle of horizontal pan")
	}
	if app.chart.replayHasStart || !app.chart.replaySelecting {
		t.Fatal("dragging at a wide candle scale committed a replay start")
	}
}

func TestReplayStartDoesNotLeaveAPersistentMarker(t *testing.T) {
	app, tt := newKlineToolbarTester()
	g := currentKlineGeometry(app)
	before := bytes.Clone(tt.Image().Pix)
	index := app.chart.Start + app.chart.Visible/2
	app.chart.selectReplayStart(index, len(app.candles))
	tt.Move(0, 0) // Keep the ordinary hover crosshair away from the selected candle.
	tt.Frame()
	markerX := g.left + float32(index-app.chart.Start+1)*g.width/float32(app.chart.Visible)
	if rgbaRegionChangedNear(before, tt.Image(), markerX, g.top+5) {
		t.Fatal("a replay start marker remained visible after confirming selection")
	}
}

func TestReplaySelectionRejectsNonChartClicksAndOutsideRelease(t *testing.T) {
	app, tt := newKlineToolbarTester()
	if err := tt.Click("回放起点"); err != nil {
		t.Fatal(err)
	}
	g := currentKlineGeometry(app)
	y := g.top + g.priceHeight*.5
	initialStart := app.chart.Start
	tt.RightClickAt(g.left+g.width*.4, y)
	tt.ClickAt(g.left+g.width+10, y)           // Price axis.
	tt.ClickAt(g.left+g.width*.4, g.bottom+10) // Time axis.
	bar, ok := tt.Find("K 线图操作栏")
	if !ok {
		t.Fatal("action bar was not rendered during selection")
	}
	tt.ClickAt(bar.X+3, bar.Y+bar.H/2) // Blank area owned by the action bar.
	if app.chart.replayHasStart || !app.chart.replaySelecting || app.chart.Start != initialStart || app.chart.dragging {
		t.Fatal("a non-primary chart click or child control committed a replay start")
	}

	// This leaves the plot without crossing the drag threshold.
	tt.Press(g.left+1, y)
	tt.Release(g.left-2, y)
	if app.chart.replayHasStart || !app.chart.replaySelecting || app.chart.dragging || app.chart.replayDragging {
		t.Fatal("releasing outside the plot committed a replay start or kept dragging")
	}
}

func TestReplaySelectionBelowTitleToolbarUsesChartLocalPointer(t *testing.T) {
	candles := demoCandles()
	app := &application{activePage: "market", candles: candles, chart: newKlineState(len(candles))}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Box(c).FillWidth().Height(40).Children(func() { app.contentTitle(c) })
			app.klineChart(c)
		})
	}, 900, 600)
	tt.Frame()
	toolbar, toolbarOK := tt.Find("K 线图工具栏")
	bar, barOK := tt.Find("K 线图操作栏")
	start, startOK := tt.Find("回放起点")
	if !toolbarOK || !barOK || !startOK || start.Y <= toolbar.Y+toolbar.H || start.X < bar.X || start.X+start.W > bar.X+bar.W || start.Y < bar.Y || start.Y+start.H > bar.Y+bar.H {
		t.Fatalf("replay start is not in the action bar below the title toolbar: toolbar=%+v, bar=%+v, start=%+v", toolbar, bar, start)
	}
	if err := tt.Click("回放起点"); err != nil {
		t.Fatal(err)
	}
	bounds, ok := tt.Find("DEMO/USDT K 线图")
	if !ok {
		t.Fatal("chart was not rendered below the title toolbar")
	}
	g := geometryForMode(bounds.W, bounds.H, app.chart.showVolume, false)
	x, y := bounds.X+g.left+g.width*.4, bounds.Y+g.top+g.priceHeight*.5
	tt.Move(x, y)
	if app.chart.replayPointerX != x-bounds.X || app.chart.hover != -1 {
		t.Fatal("selection boundary did not use the pointer's chart-local position")
	}
	tt.Scroll(x, y, 0, -120)
	if app.chart.replayHasStart || !app.chart.replaySelecting {
		t.Fatal("zooming confirmed or cancelled the replay start selection")
	}
	initialStart := app.chart.Start
	tt.Press(x, y)
	tt.Move(x+40, y)
	if app.chart.replayPointerX != x+40-bounds.X {
		t.Fatal("dragging stopped updating the chart-local selection boundary")
	}
	tt.Release(x+40, y)
	if app.chart.Start >= initialStart || app.chart.replayHasStart || !app.chart.replaySelecting {
		t.Fatal("drag below the title toolbar did not browse history without committing")
	}
	wantIndex := replayCandleIndexAtX(&app.chart, app.candles, x-bounds.X)
	tt.ClickAt(x, y)
	if app.chart.replaySelecting || app.chart.replayStart != wantIndex {
		t.Fatal("selection below the title toolbar did not finish on the clicked candle")
	}
}

func TestReplayActionBarControlsFollowSession(t *testing.T) {
	app, tt := newKlineToolbarTester()
	assertControls := func(shown, hidden []string) {
		t.Helper()
		for range 3 {
			tt.Frame() // Controls must retain identity across extra build passes.
		}
		for _, label := range shown {
			if _, ok := tt.Find(label); !ok {
				t.Fatalf("%s is missing from the action bar", label)
			}
		}
		for _, label := range hidden {
			if _, ok := tt.Find(label); ok {
				t.Fatalf("%s should not be rendered in this replay state", label)
			}
		}
	}
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		assertControls([]string{"回放起点", "回放设置"}, []string{"播放", "暂停", "重置", "结束"})
		click("回放起点")
		assertControls([]string{"回放起点", "回放设置"}, []string{"播放", "暂停", "重置", "结束"})
		click("回放起点")
		if app.chart.replaySelecting {
			t.Fatal("clicking the active start button did not cancel selection")
		}
		click("回放起点")
		g := currentKlineGeometry(app)
		tt.ClickAt(g.left+g.width*.4, g.top+g.priceHeight*.5)
		assertControls([]string{"播放", "重置", "结束"}, []string{"回放起点", "暂停", "回放设置"})
		click("播放")
		assertControls([]string{"暂停", "重置", "结束"}, []string{"回放起点", "播放", "回放设置"})
		tt.Key(0, ui.KeySpace) // The same keyed button remains focused after switching to Pause.
		if app.chart.actionPlaying {
			t.Fatal("keyboard activation did not pause the focused playback button")
		}
		assertControls([]string{"播放", "重置", "结束"}, []string{"回放起点", "暂停", "回放设置"})
		click("重置")
		assertControls([]string{"播放", "重置", "结束"}, []string{"回放起点", "暂停", "回放设置"})
		click("结束")
		click("放弃")
		assertControls([]string{"回放起点", "回放设置"}, []string{"播放", "暂停", "重置", "结束"})
		if app.chart.replayHasStart || app.chart.replaySelecting || app.chart.actionPlaying {
			t.Fatal("ending replay accidentally activated the returned start button")
		}
	}
}

func TestReplayEndReturnsToLatestAndUnlocksControls(t *testing.T) {
	for _, mode := range []string{"selected", "playing", "paused", "reset", "completed"} {
		t.Run(mode, func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			app.chart.Visible, app.chart.Start = 80, 100
			app.chart.showVolume = false
			annotation := klineAnnotation{Kind: annotationText, Start: annotationAnchor{Time: app.candles[120].Time, Price: 100}, Text: "标注"}
			addKlineAnnotation(&app.chart, annotation)
			annotations := append([]klineAnnotation(nil), app.chart.annotations...)
			candles := cloneCandles(app.candles)
			app.chart.selectReplayStart(120, len(app.candles))
			tt.Frame()
			if mode != "selected" {
				if err := tt.Click("播放"); err != nil {
					t.Fatal(err)
				}
				app.advanceReplayStep()
			}
			switch mode {
			case "paused":
				if err := tt.Click("暂停"); err != nil {
					t.Fatal(err)
				}
			case "reset":
				if err := tt.Click("重置"); err != nil {
					t.Fatal(err)
				}
			case "completed":
				app.completeReplay()
				tt.Frame()
			}

			if err := tt.Click("结束"); err != nil {
				t.Fatal(err)
			}
			clickReplaySetting(t, tt, "放弃")
			if app.chart.replayHasStart || app.chart.replayComplete || app.chart.replaySessionStarted || app.chart.replayLocked() || app.chart.actionPlaying || !app.chart.replayNextStep.IsZero() {
				t.Fatal("ending replay left an active session or scheduled playback step")
			}
			if got, want := app.chart.Start, len(app.candles)-80; got != want {
				t.Fatalf("viewport start after ending = %d, want latest start %d", got, want)
			}
			if app.chart.Visible != 80 || app.chart.visibleEnd(len(app.candles)) != len(app.candles) {
				t.Fatal("ending replay changed zoom or kept future candles hidden")
			}
			if app.chart.showVolume || app.chart.timeframe != "1h" || !reflect.DeepEqual(app.chart.annotations, annotations) || !reflect.DeepEqual(app.candles, candles) {
				t.Fatal("ending replay changed chart settings, annotations, or market data")
			}
			if app.advanceReplayStep() || app.chart.replayEnd != 0 {
				t.Fatal("playback continued after ending replay")
			}
			if err := tt.Click("4h"); err != nil {
				t.Fatal(err)
			}
			if app.chart.timeframe != "4h" {
				t.Fatal("ending replay left the period selector locked")
			}
			if err := tt.Click("选择标的 DEMO/USDT"); err != nil {
				t.Fatal(err)
			}
			if !app.instrumentPickerOpen {
				t.Fatal("ending replay left the instrument picker locked")
			}
		})
	}
}

func TestEndReplayClearsGesturesAndClampsLatestViewport(t *testing.T) {
	for _, count := range []int{0, 5, 3000} {
		state := newKlineState(count)
		state.Visible, state.Start, state.hover = 80, 100, 120
		state.replayHasStart, state.replaySessionStarted = true, true
		state.replayStart, state.replayEnd = 120, 130
		state.actionPlaying, state.replaySelecting, state.replayDragging = true, true, true
		state.replayDragMoved, state.dragging, state.dragRemainder = true, true, .5
		state.replayNextStep = time.Now().Add(time.Second)
		state.endReplay(count)
		if state.replaySelecting || state.replayDragging || state.replayDragMoved || state.dragging || state.dragRemainder != 0 || state.hover != -1 || !state.replayNextStep.IsZero() {
			t.Fatalf("ending replay with %d candles left a gesture, hover, or timer active", count)
		}
		wantVisible := min(count, 80)
		if state.Visible != wantVisible || state.Start != max(0, count-wantVisible) || state.visibleEnd(count) != count {
			t.Fatalf("ending replay with %d candles produced viewport %+v", count, state.Viewport)
		}
		if state.replayHasStart || state.replaySessionStarted || state.replayComplete || state.replayStart != 0 || state.replayEnd != 0 || state.actionPlaying {
			t.Fatalf("ending replay with %d candles retained session state", count)
		}
	}
}

func TestReplayActionBarResizesAndClampsWithControls(t *testing.T) {
	for _, edge := range []string{"center", "left", "right"} {
		t.Run(edge, func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			switch edge {
			case "left":
				app.chart.actionBarX = 0
			case "right":
				app.chart.actionBarX = app.chart.chartWidth - klineActionBarWidth
			}
			tt.Frame()
			oldCenter := app.chart.actionBarX + klineActionBarWidth/2
			app.chart.selectReplayStart(app.chart.Start+app.chart.Visible/2, len(app.candles))
			tt.Frame()
			bar, ok := tt.Find("K 线图操作栏")
			if !ok || bar.W != klineReplayActionBarWidth || bar.X < 0 || bar.X+bar.W > app.chart.chartWidth {
				t.Fatalf("expanded replay action bar is not contained in the chart: %+v", bar)
			}
			if edge == "center" && bar.X+bar.W/2 != oldCenter {
				t.Fatal("expanding controls moved the action bar's center")
			}
			previousX := float32(-1)
			for _, label := range []string{"播放", "播放速度", "下单数量", "买入", "卖出", "重置", "结束"} {
				button, ok := tt.Find(label)
				if !ok || button.X < bar.X || button.X+button.W > bar.X+bar.W || button.Y < bar.Y || button.Y+button.H > bar.Y+bar.H || button.X <= previousX {
					t.Fatalf("%s is outside the expanded bar or out of order: bar=%+v, button=%+v", label, bar, button)
				}
				previousX = button.X + button.W
			}
			if !pointerOverActionBar(&app.chart, bar.X+bar.W-3, bar.Y+bar.H/2) || pointerOverActionBar(&app.chart, bar.X+bar.W+1, bar.Y+bar.H/2) {
				t.Fatal("expanded action bar hit region does not match its visible width")
			}
			initialStart := app.chart.Start
			x, y := bar.X+bar.W-3, bar.Y+bar.H/2
			tt.Press(x, y)
			tt.Move(x-20, y-8)
			tt.Release(x-20, y-8)
			if app.chart.Start != initialStart || app.chart.dragging {
				t.Fatal("dragging the expanded action bar also panned the chart")
			}
			center := app.chart.actionBarX + klineReplayActionBarWidth/2
			if err := tt.Click("结束"); err != nil {
				t.Fatal(err)
			}
			clickReplaySetting(t, tt, "放弃")
			bar, ok = tt.Find("K 线图操作栏")
			if !ok || bar.W != klineActionBarWidth || bar.X+bar.W/2 != center {
				t.Fatalf("ending replay did not shrink the action bar around its center: %+v", bar)
			}
		})
	}
}

func TestReplayCandleIndexAtPlotEdges(t *testing.T) {
	candles := demoCandles()
	state := newKlineState(len(candles))
	state.Start = 10
	state.plotWidth, state.chartHeight = 800, 500
	for _, tc := range []struct {
		x    float32
		want int
	}{
		{x: -100, want: state.Start},
		{x: 8, want: state.Start},
		{x: 8 + state.plotWidth, want: state.Start + state.Visible - 1},
		{x: 10000, want: state.Start + state.Visible - 1},
	} {
		if got := replayCandleIndexAtX(&state, candles, tc.x); got != tc.want {
			t.Errorf("index at x=%v = %d, want %d", tc.x, got, tc.want)
		}
	}
	barWidth := state.plotWidth / float32(state.Visible)
	for i := 0; i < state.Visible; i++ {
		x := 8 + (float32(i)+.5)*barWidth
		if got, want := replayCandleIndexAtX(&state, candles, x), state.Start+i; got != want {
			t.Fatalf("index at candle center x=%v = %d, want %d", math.Round(float64(x)), got, want)
		}
	}
}
