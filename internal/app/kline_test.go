package app

import (
	"bytes"
	"image"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestKlineChartWheelZoomAndDragPan(t *testing.T) {
	candles := demoCandles()
	app := &application{candles: candles, chart: newKlineState(len(candles))}
	tt := ui.NewTester(func(c *ui.Context) { app.klineChart(c) }, 900, 560)

	initialVisible := app.chart.Visible
	tt.Scroll(320, 220, 0, -120)
	if app.chart.Visible >= initialVisible {
		t.Fatalf("wheel up did not zoom in: visible %d, started at %d", app.chart.Visible, initialVisible)
	}

	start := app.chart.Start
	tt.Press(320, 220)
	tt.Move(370, 220)
	tt.Release(370, 220)
	if app.chart.Start >= start {
		t.Fatalf("dragging right did not pan toward older data: start %d, was %d", app.chart.Start, start)
	}
}

func TestKlineHoverHighlightsCandleAndAxisReadings(t *testing.T) {
	candles := demoCandles()
	app := &application{candles: candles, chart: newKlineState(len(candles))}
	tt := ui.NewTester(func(c *ui.Context) { app.klineChart(c) }, 900, 560)
	before := bytes.Clone(tt.Image().Pix)

	g := currentKlineGeometry(app)
	fraction := float32(.62)
	x := g.left + g.width*fraction
	y := g.top + g.priceHeight*.45
	tt.Move(x, y)

	wantIndex := app.chart.Start + int(fraction*float32(app.chart.Visible))
	if app.chart.hover != wantIndex {
		t.Fatalf("hover selected candle %d, want %d", app.chart.hover, wantIndex)
	}
	if app.chart.hoverY != y {
		t.Fatalf("hover price cursor y = %v, want %v", app.chart.hoverY, y)
	}
	if bytes.Equal(before, tt.Image().Pix) {
		t.Fatal("hover did not render a highlighted candle and axis readouts")
	}
}

func TestKlineToolbarRendersAboveTheChart(t *testing.T) {
	app, tt := newKlineToolbarTester()
	bounds, ok := tt.Find("K 线图工具栏")
	if !ok {
		t.Fatal("chart toolbar was not rendered")
	}
	g := currentKlineGeometry(app)
	if bounds.Y >= g.top {
		t.Fatalf("toolbar y = %v, should appear above chart plot at y=%v", bounds.Y, g.top)
	}
	pair, pairOK := tt.Find("DEMO/USDT")
	period, periodOK := tt.Find("15m")
	trend, trendOK := tt.Find("趋势线")
	deleteButton, deleteOK := tt.Find("删除标注")
	if !pairOK || !periodOK || !trendOK || !deleteOK {
		t.Fatal("toolbar is missing the trading pair, period selector, drawing tools, or delete action")
	}
	if _, selectOK := tt.Find("选择/平移"); selectOK {
		t.Fatal("Select/Pan icon button should not be rendered")
	}
	toolbar, toolbarOK := tt.Find("K 线图工具栏")
	if !toolbarOK || deleteButton.Y < toolbar.Y || deleteButton.Y+deleteButton.H > toolbar.Y+toolbar.H {
		t.Fatalf("delete action is not inside the chart toolbar: toolbar=%+v, delete=%+v", toolbar, deleteButton)
	}
	if !(pair.X < period.X && period.X < trend.X && trend.X < deleteButton.X) {
		t.Fatalf("toolbar order is wrong: pair x=%v, period x=%v, trend x=%v, delete x=%v", pair.X, period.X, trend.X, deleteButton.X)
	}
	bar, barOK := tt.Find("K 线图操作栏")
	replay, replayOK := tt.Find("回放起点")
	if !barOK || !replayOK || replay.X < bar.X || replay.Y < bar.Y || replay.X+replay.W > bar.X+bar.W || replay.Y+replay.H > bar.Y+bar.H {
		t.Fatalf("replay start is not inside the action bar: bar=%+v, replay=%+v", bar, replay)
	}
}

func TestKlineActionBarFloatsAndCanBeDragged(t *testing.T) {
	app, tt := newKlineToolbarTester()
	if !app.chart.actionBarPositioned {
		t.Fatal("floating action bar position was not initialized")
	}
	wantX, wantY := defaultKlineActionBarPosition(ui.Rect{W: 900, H: app.chart.chartHeight}, app.chart.actionBarWidth)
	if app.chart.actionBarX != wantX || app.chart.actionBarY != wantY {
		t.Fatalf("action bar position = (%v, %v), want (%v, %v)", app.chart.actionBarX, app.chart.actionBarY, wantX, wantY)
	}
	bar, ok := tt.Find("K 线图操作栏")
	if !ok {
		t.Fatal("floating action bar was not rendered")
	}
	if bar.X != wantX || bar.Y != wantY {
		t.Fatalf("action bar bounds = (%v, %v), want (%v, %v)", bar.X, bar.Y, wantX, wantY)
	}

	initialStart := app.chart.Start
	startX, startY := bar.X+3, bar.Y+bar.H/2 // blank area before the first button
	tt.Press(startX, startY)
	tt.Move(startX+40, startY+8)
	tt.Release(startX+40, startY+8)
	if app.chart.actionBarX != wantX+40 || app.chart.actionBarY != wantY+8 {
		t.Fatalf("dragged action bar position = (%v, %v), want (%v, %v)", app.chart.actionBarX, app.chart.actionBarY, wantX+40, wantY+8)
	}
	if app.chart.Start != initialStart || app.chart.dragging {
		t.Fatal("dragging the floating action bar also panned the chart")
	}

	bar, ok = tt.Find("K 线图操作栏")
	if !ok {
		t.Fatal("floating action bar disappeared after dragging")
	}
	startX, startY = bar.X+3, bar.Y+bar.H/2
	tt.Press(startX, startY)
	tt.Move(startX+450, startY+80)
	tt.Release(startX+450, startY+80)
	maxX := max(0, 900-int(klineActionBarWidth))
	maxY := max(0, int(app.chart.chartHeight)-int(klineActionBarHeight))
	if app.chart.actionBarX != float32(maxX) || app.chart.actionBarY != float32(maxY) {
		t.Fatalf("action bar was not clamped to chart: position = (%v, %v), want (%d, %d)", app.chart.actionBarX, app.chart.actionBarY, maxX, maxY)
	}
	if app.chart.Start != initialStart || app.chart.dragging {
		t.Fatal("dragging the floating action bar also panned the chart")
	}
}

func TestKlineReplayButtonsPauseResumeAndReset(t *testing.T) {
	app, tt := newKlineToolbarTester()
	for _, label := range []string{"播放", "暂停", "重置", "结束"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("%s was rendered without a selected replay start", label)
		}
	}

	startIndex := app.chart.Start + app.chart.Visible/2
	app.chart.selectReplayStart(startIndex, len(app.candles))
	tt.Frame()
	if err := tt.Click("播放"); err != nil {
		t.Fatal(err)
	}
	if !app.chart.actionPlaying || !app.chart.replaySessionStarted || app.chart.replaySelecting {
		t.Fatal("Play did not start playback and exit start-selection mode")
	}
	if _, ok := tt.Find("暂停"); !ok {
		t.Fatal("Pause button was not rendered during playback")
	}

	visibleBeforePause := app.chart.replayEnd
	if err := tt.Click("暂停"); err != nil {
		t.Fatal(err)
	}
	if app.chart.actionPlaying || app.chart.replayEnd != visibleBeforePause {
		t.Fatal("Pause did not hold playback at the current candle")
	}
	if err := tt.Click("播放"); err != nil {
		t.Fatal(err)
	}
	if !app.chart.actionPlaying {
		t.Fatal("Play did not resume paused playback")
	}
	if !app.advanceReplayStep() {
		if app.chart.replayEnd != visibleBeforePause+1 {
			t.Fatalf("resumed replay end = %d, want %d", app.chart.replayEnd, visibleBeforePause+1)
		}
	}

	if err := tt.Click("暂停"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("重置"); err != nil {
		t.Fatal(err)
	}
	if app.chart.actionPlaying || !app.chart.replaySessionStarted || app.chart.replayEnd != startIndex+1 {
		t.Fatalf("Reset did not return to the selected start: playing=%v started=%v end=%d start=%d", app.chart.actionPlaying, app.chart.replaySessionStarted, app.chart.replayEnd, startIndex)
	}
}

func TestKlinePeriodSelectorChangesSelectedPeriod(t *testing.T) {
	app, tt := newKlineToolbarTester()
	tt.SetPreferences(ui.Preferences{Accent: ui.Hex("#ff00ff"), TextScale: 1})
	if app.chart.timeframe != "1h" {
		t.Fatalf("initial period = %q, want 1h", app.chart.timeframe)
	}
	if err := tt.Click("4h"); err != nil {
		t.Fatal(err)
	}
	if app.chart.timeframe != "4h" {
		t.Fatalf("selected period = %q, want 4h", app.chart.timeframe)
	}
	bounds, ok := tt.Find("4h")
	if !ok {
		t.Fatal("selected timeframe button was not rendered")
	}
	if bounds.W >= 36 || bounds.H >= 30 {
		t.Fatalf("timeframe button size = %vx%v, want smaller than 36x30", bounds.W, bounds.H)
	}
	assertNeutralButtonBackground(t, tt.Image(), bounds, 3)
}

func TestReplaySelectionDragsThroughHistoryAndHidesFutureCandles(t *testing.T) {
	app, tt := newKlineToolbarTester()
	if err := tt.Click("回放起点"); err != nil {
		t.Fatal(err)
	}
	if !app.chart.replaySelecting {
		t.Fatal("replay start mode did not activate")
	}
	tt.Key(0, ui.KeyEscape)
	if app.chart.replaySelecting {
		t.Fatal("Escape did not exit selection mode while the action bar button had focus")
	}
	if err := tt.Click("回放起点"); err != nil {
		t.Fatal(err)
	}

	initialStart, visible := app.chart.Start, app.chart.Visible
	g := currentKlineGeometry(app)
	x, y := g.left+g.width*.45, g.top+g.priceHeight*.5
	endX := x + 40
	tt.Press(x, y)
	tt.Move(endX, y)
	tt.Release(endX, y)

	if app.chart.Start >= initialStart {
		t.Fatalf("dragging right did not pan to older candles: start %d, was %d", app.chart.Start, initialStart)
	}
	if app.chart.replayHasStart || !app.chart.replaySelecting || app.chart.replayLocked() {
		t.Fatal("dragging should browse history without committing a replay start")
	}
	if app.chart.Visible != visible {
		t.Fatalf("selection changed viewport scale: visible=%d, want %d", app.chart.Visible, visible)
	}
	if got, want := app.chart.visibleEnd(len(app.candles)), app.chart.Start+visible; got != want {
		t.Fatalf("selection preview hid future candles: visible end = %d, want %d", got, want)
	}

	wantStart := replayCandleIndexAtX(&app.chart, app.candles, endX)
	startBeforeClick := app.chart.Start
	tt.ClickAt(endX, y)
	if app.chart.replayStart != wantStart || app.chart.replayEnd != wantStart+1 {
		t.Fatalf("selected replay range = [%d,%d), want start %d", app.chart.replayStart, app.chart.replayEnd, wantStart)
	}
	if app.chart.replaySelecting || !app.chart.replayLocked() || app.chart.dragging || app.chart.replayDragging {
		t.Fatal("click did not finish selection and lock the selected series")
	}
	if app.chart.Start != startBeforeClick {
		t.Fatal("clicking a replay start also panned the chart")
	}
	if got := app.chart.visibleEnd(len(app.candles)); got != min(app.chart.Start+visible, wantStart+1) {
		t.Fatalf("visible candle end = %d, want cutoff at %d", got, wantStart+1)
	}

	tt.Key(0, ui.KeyEscape)
	if app.chart.replaySelecting || app.chart.visibleEnd(len(app.candles)) != min(app.chart.Start+visible, wantStart+1) {
		t.Fatal("Escape changed the confirmed replay start")
	}
	if _, ok := tt.Find("回放起点"); ok {
		t.Fatal("replay start button remained visible after confirming the start")
	}
}

func TestReplayLocksInstrumentAndPeriodUntilCompletion(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.selectReplayStart(len(app.candles)-2, len(app.candles))
	tt.Frame()

	if err := tt.Click("4h"); err != nil {
		t.Fatal(err)
	}
	if app.chart.timeframe != "1h" {
		t.Fatal("period changed after selecting a replay start")
	}
	if err := tt.Click("选择标的 DEMO/USDT"); err != nil {
		t.Fatal(err)
	}
	if app.instrumentPickerOpen {
		t.Fatal("instrument picker opened before the replay completed")
	}

	if err := tt.Click("播放"); err != nil {
		t.Fatal(err)
	}
	if !app.chart.actionPlaying {
		t.Fatal("replay did not start")
	}
	if err := tt.Click("暂停"); err != nil {
		t.Fatal(err)
	}
	if app.chart.actionPlaying {
		t.Fatal("replay did not pause")
	}
	if err := tt.Click("4h"); err != nil {
		t.Fatal(err)
	}
	if app.chart.timeframe != "1h" {
		t.Fatal("period changed while replay was paused")
	}
	if _, ok := tt.Find("回放起点"); ok {
		t.Fatal("replay start tool remained visible while playback was paused")
	}

	app.chart.actionPlaying = true
	app.chart.replayNextStep = time.Now().Add(-time.Second)
	tt.Frame()
	if !app.chart.replayComplete || app.chart.actionPlaying || app.chart.replayLocked() {
		t.Fatal("replay did not stop and unlock controls after the final candle")
	}
	if !tt.HasText("回放结束") {
		t.Fatal("replay completion toast was not shown")
	}
	wantVisibleEnd := min(len(app.candles), app.chart.Start+app.chart.Visible)
	if app.chart.visibleEnd(len(app.candles)) != wantVisibleEnd {
		t.Fatal("replay completion kept the future-candle cutoff")
	}
	if app.chart.Start > len(app.candles)-1 || app.chart.Start+app.chart.Visible <= len(app.candles)-1 {
		t.Fatal("playback did not follow the latest candle into the viewport")
	}
	if err := tt.Click("4h"); err != nil {
		t.Fatal(err)
	}
	if app.chart.timeframe != "4h" {
		t.Fatal("period remained locked after replay completion")
	}
	if err := tt.Click("选择标的 DEMO/USDT"); err != nil {
		t.Fatal(err)
	}
	if !app.instrumentPickerOpen {
		t.Fatal("instrument picker remained locked after replay completion")
	}
	app.instrumentPickerOpen = false
	tt.Frame()
	tt2 := ui.NewTester(func(c *ui.Context) { app.klineChart(c) }, 900, 560)
	if err := tt2.Click("重置"); err != nil {
		t.Fatal(err)
	}
	if app.chart.replayComplete || !app.chart.replayLocked() || !app.chart.replaySessionStarted || app.chart.replayEnd != app.chart.replayStart+1 {
		t.Fatalf("Reset after completion did not re-enter replay at the selected start: complete=%v locked=%v started=%v end=%d start=%d", app.chart.replayComplete, app.chart.replayLocked(), app.chart.replaySessionStarted, app.chart.replayEnd, app.chart.replayStart)
	}
}

func assertNeutralButtonBackground(t *testing.T, img *image.RGBA, bounds ui.Rect, insetY float32) {
	t.Helper()
	pixel := img.RGBAAt(int(bounds.X+bounds.W/2), int(bounds.Y+insetY))
	channels := []uint8{pixel.R, pixel.G, pixel.B}
	minChannel, maxChannel := channels[0], channels[0]
	for _, channel := range channels[1:] {
		minChannel = min(minChannel, channel)
		maxChannel = max(maxChannel, channel)
	}
	if maxChannel-minChannel > 20 {
		t.Fatalf("selected button background is not neutral gray: RGB(%d, %d, %d)", pixel.R, pixel.G, pixel.B)
	}
}

func newKlineToolbarTester() (*application, *ui.Tester) {
	candles := demoCandles()
	app := &application{candles: candles, chart: newKlineState(len(candles)), replaySettings: defaultReplaySettings()}
	tt := ui.NewTester(func(c *ui.Context) { app.klineChart(c) }, 900, 560)
	tt.Frame() // settle chart dimensions so the toolbar can fill the chart width
	return app, tt
}

func TestCrosshairPriceFollowsPointerHeight(t *testing.T) {
	g := geometryFor(900, klineChartHeight, true)
	const low, high = 80.0, 120.0
	for _, tc := range []struct {
		y    float32
		want float64
	}{
		{g.top, high},
		{g.top + g.priceHeight/2, 100},
		{g.priceBottom, low},
	} {
		if got := priceForY(tc.y, g, low, high); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("price at y=%v = %v, want %v", tc.y, got, tc.want)
		}
	}
}

func TestHoverTimeUsesLocalTimezone(t *testing.T) {
	original := time.Local
	time.Local = time.FixedZone("test local", 8*60*60)
	t.Cleanup(func() { time.Local = original })

	stamp := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	if got, want := formatCandleTime(stamp), "2025-01-01 08:00"; got != want {
		t.Fatalf("hover time = %q, want local time %q", got, want)
	}
}

func TestKlineChartFillsAndTracksAvailableSize(t *testing.T) {
	app := &application{candles: demoCandles(), chart: newKlineState(demoCandleCount)}
	tt := ui.NewTester(func(c *ui.Context) { app.marketPage(c) }, 1000, 700)
	if app.chart.chartWidth != 1000 || app.chart.chartHeight != 700 {
		t.Fatalf("initial chart size = %vx%v, want 1000x700", app.chart.chartWidth, app.chart.chartHeight)
	}

	tt.SetSize(760, 440)
	if app.chart.chartWidth != 760 || app.chart.chartHeight != 440 {
		t.Fatalf("resized chart size = %vx%v, want 760x440", app.chart.chartWidth, app.chart.chartHeight)
	}
}

func TestMarketContentKeepsChartResponsive(t *testing.T) {
	app := &application{activePage: "market", candles: demoCandles(), chart: newKlineState(demoCandleCount)}
	tt := ui.NewTester(func(c *ui.Context) { app.content(c) }, 1000, 700)
	if app.chart.chartWidth != 1000 || app.chart.chartHeight <= klineChartHeight {
		t.Fatalf("content chart size = %vx%v, want full content height", app.chart.chartWidth, app.chart.chartHeight)
	}
	initialHeight := app.chart.chartHeight

	tt.SetSize(1000, 520)
	if app.chart.chartHeight >= initialHeight || app.chart.chartHeight <= 0 {
		t.Fatalf("content chart height after resize = %v, want less than %v and positive", app.chart.chartHeight, initialHeight)
	}
}

func TestKlineVolumeSwitchCanBeTurnedOff(t *testing.T) {
	candles := demoCandles()
	app := &application{candles: candles, chart: newKlineState(len(candles))}
	tt := ui.NewTester(func(c *ui.Context) { app.marketPage(c) }, 1000, 700)
	if !app.chart.showVolume {
		t.Fatal("volume should be visible by default")
	}
	if _, ok := tt.Find("成交量"); !ok {
		t.Fatal("volume switch was not rendered")
	}
	tt.Click("成交量")
	if app.chart.showVolume {
		t.Fatal("clicking the volume switch did not turn the volume panel off")
	}
}

func TestTrendLineUsesTwoClickRubberBandAndSupportsArbitraryAngles(t *testing.T) {
	app, tt := newAnnotationTester()
	if err := tt.Click("趋势线"); err != nil {
		t.Fatal(err)
	}
	g := currentKlineGeometry(app)
	x1, y1 := g.left+g.width*.27, g.top+g.priceHeight*.24
	x2, y2 := g.left+g.width*.73, g.top+g.priceHeight*.76

	// Put the pointer at the initial location and remove the crosshair so the
	// image diff below isolates the newly displayed anchor marker.
	tt.Move(x1, y1)
	app.chart.hover = -1
	tt.Frame()
	beforeAnchor := bytes.Clone(tt.Image().Pix)
	tt.ClickAt(x1, y1)
	if app.chart.annotationGesture != annotationGestureCreateTrend {
		t.Fatal("first click did not leave a pending trend-line anchor")
	}
	if len(app.chart.annotations) != 0 {
		t.Fatal("trend line was committed on the anchor click")
	}
	if bytes.Equal(beforeAnchor, tt.Image().Pix) {
		t.Fatal("first click did not visibly render the anchor point")
	}

	beforePreview := bytes.Clone(tt.Image().Pix)
	tt.Move(x2, y2) // Move after release: a rubber-band preview must still update.
	if app.chart.gestureCurrent.Time.Equal(app.chart.gestureStart.Time) || app.chart.gestureCurrent.Price == app.chart.gestureStart.Price {
		t.Fatal("pointer movement did not update both time and price of the preview endpoint")
	}
	previewImage := tt.Image()
	if bytes.Equal(beforePreview, previewImage.Pix) {
		t.Fatal("moving the pointer did not render the anchor-to-pointer preview line")
	}
	low, high, _ := visibleKlinePriceRange(app.candles, &app.chart)
	previewX1, previewY1, _ := annotationPoint(app.chart.gestureStart, g, app.candles, &app.chart, low, high)
	previewX2, previewY2, _ := annotationPoint(app.chart.gestureCurrent, g, app.candles, &app.chart, low, high)
	diagonalX := previewX1 + (previewX2-previewX1)*.25
	diagonalY := previewY1 + (previewY2-previewY1)*.25
	if !rgbaRegionChangedNear(beforePreview, previewImage, diagonalX, diagonalY) {
		t.Fatal("rubber-band preview is not rendered along its diagonal")
	}

	tt.ClickAt(x2, y2)
	if len(app.chart.annotations) != 1 || app.chart.annotationGesture != annotationGestureNone {
		t.Fatal("second click did not complete the trend line")
	}
	if !rgbaRegionChangedNear(beforePreview, tt.Image(), diagonalX, diagonalY) {
		t.Fatal("completed trend line is not rendered along its diagonal")
	}
	line := app.chart.annotations[0]
	if line.Start.Time.Equal(line.End.Time) || line.Start.Price == line.End.Price {
		t.Fatalf("completed trend line is not at an arbitrary angle: %+v", line)
	}
}

func TestTrendLineEscapeCancelsPendingAnchor(t *testing.T) {
	app, tt := newAnnotationTester()
	if err := tt.Click("趋势线"); err != nil {
		t.Fatal(err)
	}
	g := currentKlineGeometry(app)
	tt.ClickAt(g.left+g.width*.4, g.top+g.priceHeight*.4)
	if app.chart.annotationGesture != annotationGestureCreateTrend {
		t.Fatal("first click did not establish a pending trend-line anchor")
	}
	tt.Key(0, ui.KeyEscape)
	if app.chart.annotationGesture != annotationGestureNone || len(app.chart.annotations) != 0 {
		t.Fatal("Escape did not cancel the pending trend-line anchor")
	}
	if app.chart.tool != klineToolSelect {
		t.Fatalf("cancelled trend-line action left tool = %v, want pan mode", app.chart.tool)
	}
}

func TestTrendLineToolCreatesEditsAndDeletesAnnotation(t *testing.T) {
	app, tt := newAnnotationTester()
	if err := tt.Click("趋势线"); err != nil {
		t.Fatal(err)
	}
	if app.chart.tool != klineToolTrend {
		t.Fatalf("active tool = %v, want trend line", app.chart.tool)
	}
	g := currentKlineGeometry(app)
	x1, y1 := g.left+g.width*.32, g.top+g.priceHeight*.32
	x2, y2 := g.left+g.width*.68, g.top+g.priceHeight*.72
	tt.ClickAt(x1, y1)
	tt.Move(x2, y2)
	tt.ClickAt(x2, y2)
	if len(app.chart.annotations) != 1 {
		t.Fatalf("created %d annotations, want 1", len(app.chart.annotations))
	}
	line := app.chart.annotations[0]
	if line.Kind != annotationTrend || line.Start.Time.Equal(line.End.Time) || line.Start.Price == line.End.Price {
		t.Fatalf("trend line was not anchored at both drag endpoints: %+v", line)
	}
	if app.chart.selectedAnnotation != line.ID {
		t.Fatalf("selected annotation = %d, want newly created id %d", app.chart.selectedAnnotation, line.ID)
	}
	if app.chart.tool != klineToolSelect {
		t.Fatalf("completed trend-line action left tool = %v, want pan mode", app.chart.tool)
	}

	low, high, ok := visibleKlinePriceRange(app.candles, &app.chart)
	if !ok {
		t.Fatal("visible price range was unavailable")
	}
	startX, startY, _ := annotationPoint(line.Start, g, app.candles, &app.chart, low, high)
	originalStart, originalEnd := app.chart.annotations[0].Start, app.chart.annotations[0].End
	tt.Press(startX, startY)
	tt.Move(startX+18, startY+10)
	tt.Release(startX+18, startY+10)
	line = app.chart.annotations[0]
	if line.Start.Time.Equal(originalStart.Time) && line.Start.Price == originalStart.Price {
		t.Fatal("dragging the trend line start handle did not update it")
	}
	if !line.End.Time.Equal(originalEnd.Time) || line.End.Price != originalEnd.Price {
		t.Fatal("dragging the start handle also changed the end handle")
	}

	if err := tt.Click("删除标注"); err != nil {
		t.Fatal(err)
	}
	if len(app.chart.annotations) != 0 || app.chart.selectedAnnotation != 0 {
		t.Fatal("delete toolbar action did not remove the selected trend line")
	}
}

func TestCompletedTrendLineReturnsToPanUntilToolIsSelectedAgain(t *testing.T) {
	app, tt := newAnnotationTester()
	g := currentKlineGeometry(app)
	x1, y1 := g.left+g.width*.2, g.top+g.priceHeight*.25
	x2, y2 := g.left+g.width*.4, g.top+g.priceHeight*.55

	if err := tt.Click("趋势线"); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(x1, y1)
	tt.ClickAt(x2, y2)
	if len(app.chart.annotations) != 1 || app.chart.tool != klineToolSelect {
		t.Fatalf("completed trend-line action = annotations:%d tool:%v, want one annotation and pan mode", len(app.chart.annotations), app.chart.tool)
	}

	start := app.chart.Start
	tt.Press(g.left+g.width*.05, g.top+g.priceHeight*.8)
	tt.Move(g.left+g.width*.05+30, g.top+g.priceHeight*.8)
	tt.Release(g.left+g.width*.05+30, g.top+g.priceHeight*.8)
	if len(app.chart.annotations) != 1 {
		t.Fatal("pan-mode drag started another trend line without selecting the tool")
	}
	if app.chart.Start == start {
		t.Fatal("completed trend-line action did not return to pan behavior")
	}

	if err := tt.Click("趋势线"); err != nil {
		t.Fatal(err)
	}
	if app.chart.tool != klineToolTrend {
		t.Fatalf("reselected trend-line tool = %v, want trend mode", app.chart.tool)
	}
}

func TestTextToolCreatesEditsAndMovesTextAnnotation(t *testing.T) {
	app, tt := newAnnotationTester()
	before := bytes.Clone(tt.Image().Pix)
	if err := tt.Click("文本"); err != nil {
		t.Fatal(err)
	}
	g := currentKlineGeometry(app)
	x, y := g.left+g.width*.44, g.top+g.priceHeight*.38
	tt.Press(x, y)
	tt.Release(x, y)
	if !app.chart.textEditing {
		t.Fatal("clicking in text mode did not open the inline editor")
	}
	if _, ok := tt.Find("标注文字"); !ok {
		t.Fatal("inline text editor is not rendered or labeled")
	}
	tt.Type("关键价格")
	tt.Key(0, ui.KeyEnter)
	if app.chart.textEditing || len(app.chart.annotations) != 1 {
		t.Fatal("Enter did not commit the new text annotation")
	}
	annotation := app.chart.annotations[0]
	if annotation.Kind != annotationText || annotation.Text != "关键价格" {
		t.Fatalf("created annotation = %+v, want text '关键价格'", annotation)
	}
	if bytes.Equal(before, tt.Image().Pix) {
		t.Fatal("committed text annotation did not change the rendered chart")
	}
	if app.chart.tool != klineToolSelect {
		t.Fatalf("completed text action left tool = %v, want pan mode", app.chart.tool)
	}

	low, high, ok := visibleKlinePriceRange(app.candles, &app.chart)
	if !ok {
		t.Fatal("visible price range was unavailable")
	}
	labelX, labelY, _ := annotationPoint(annotation.Start, g, app.candles, &app.chart, low, high)
	tt.ClickAt(labelX+2, labelY+5)
	tt.ClickAt(labelX+2, labelY+5)
	if !app.chart.textEditing || app.chart.textEditingNew {
		t.Fatal("double-clicking text in Select mode did not enter edit mode")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("阻力位")
	tt.Key(0, ui.KeyEnter)
	if got := app.chart.annotations[0].Text; got != "阻力位" {
		t.Fatalf("edited text = %q, want 阻力位", got)
	}

	annotation = app.chart.annotations[0]
	labelX, labelY, _ = annotationPoint(annotation.Start, g, app.candles, &app.chart, low, high)
	oldAnchor := annotation.Start
	tt.Press(labelX+2, labelY+5)
	tt.Move(labelX+24, labelY+15)
	tt.Release(labelX+24, labelY+15)
	annotation = app.chart.annotations[0]
	if annotation.Start.Time.Equal(oldAnchor.Time) && annotation.Start.Price == oldAnchor.Price {
		t.Fatal("dragging text did not move its time/price anchor")
	}
}

func TestTextToolEscapeCancelsEmptyDraft(t *testing.T) {
	app, tt := newAnnotationTester()
	if err := tt.Click("文本"); err != nil {
		t.Fatal(err)
	}
	g := currentKlineGeometry(app)
	tt.Press(g.left+g.width*.5, g.top+g.priceHeight*.5)
	tt.Release(g.left+g.width*.5, g.top+g.priceHeight*.5)
	tt.Key(0, ui.KeyEscape)
	if app.chart.textEditing || len(app.chart.annotations) != 0 {
		t.Fatal("Escape did not cancel the empty text draft")
	}
	if app.chart.tool != klineToolSelect {
		t.Fatalf("cancelled text action left tool = %v, want pan mode", app.chart.tool)
	}
}

func TestTrendLineBodyDragMovesBothEndpointsTogether(t *testing.T) {
	app, tt := newAnnotationTester()
	g := currentKlineGeometry(app)
	createTestTrendLine(t, app, tt, g.left+g.width*.25, g.top+g.priceHeight*.3, g.left+g.width*.75, g.top+g.priceHeight*.65)
	if app.chart.tool != klineToolSelect {
		t.Fatalf("completed trend-line action left tool = %v, want pan mode", app.chart.tool)
	}
	line := app.chart.annotations[0]
	low, high, _ := visibleKlinePriceRange(app.candles, &app.chart)
	x1, y1, _ := annotationPoint(line.Start, g, app.candles, &app.chart, low, high)
	x2, y2, _ := annotationPoint(line.End, g, app.candles, &app.chart, low, high)
	originalStart, originalEnd := line.Start, line.End
	midX, midY := (x1+x2)/2, (y1+y2)/2
	tt.Press(midX, midY)
	tt.Move(midX+20, midY+12)
	tt.Release(midX+20, midY+12)
	line = app.chart.annotations[0]
	startShift := line.Start.Time.Sub(originalStart.Time)
	endShift := line.End.Time.Sub(originalEnd.Time)
	if startShift == 0 || startShift != endShift {
		t.Fatalf("whole-line drag time shifts = (%v, %v), want same nonzero shift", startShift, endShift)
	}
	startPriceShift := line.Start.Price - originalStart.Price
	endPriceShift := line.End.Price - originalEnd.Price
	if startPriceShift == 0 || math.Abs(startPriceShift-endPriceShift) > 1e-9 {
		t.Fatalf("whole-line drag price shifts = (%v, %v), want same nonzero shift", startPriceShift, endPriceShift)
	}
}

func TestAnnotationsStayAnchoredWhileChartZoomsAndPans(t *testing.T) {
	app, tt := newAnnotationTester()
	g := currentKlineGeometry(app)
	createTestTrendLine(t, app, tt, g.left+g.width*.35, g.top+g.priceHeight*.3, g.left+g.width*.7, g.top+g.priceHeight*.6)
	if app.chart.tool != klineToolSelect {
		t.Fatalf("completed trend-line action left tool = %v, want pan mode", app.chart.tool)
	}
	before := app.chart.annotations[0]
	low, high, _ := visibleKlinePriceRange(app.candles, &app.chart)
	beforeX, _, _ := annotationPoint(before.Start, g, app.candles, &app.chart, low, high)

	tt.Scroll(g.left+g.width*.9, g.top+30, 0, -120)
	if !app.chart.annotations[0].Start.Time.Equal(before.Start.Time) || app.chart.annotations[0].Start.Price != before.Start.Price {
		t.Fatal("zoom changed the stored annotation time/price anchor")
	}
	low, high, _ = visibleKlinePriceRange(app.candles, &app.chart)
	zoomedX, _, _ := annotationPoint(app.chart.annotations[0].Start, g, app.candles, &app.chart, low, high)
	if math.Abs(float64(zoomedX-beforeX)) < 1 {
		t.Fatal("annotation did not move with its anchored candle during zoom")
	}

	startBeforePan := app.chart.Start
	tt.Press(g.left+g.width*.1, g.top+10)
	tt.Move(g.left+g.width*.1+30, g.top+10)
	tt.Release(g.left+g.width*.1+30, g.top+10)
	if app.chart.Start == startBeforePan {
		t.Fatal("chart drag did not pan during Select/Pan mode")
	}
	if !app.chart.annotations[0].Start.Time.Equal(before.Start.Time) || app.chart.annotations[0].Start.Price != before.Start.Price {
		t.Fatal("pan changed the stored annotation time/price anchor")
	}
}

func TestSelectedAnnotationCanBeDeletedWithKeyboard(t *testing.T) {
	app, tt := newAnnotationTester()
	g := currentKlineGeometry(app)
	createTestTrendLine(t, app, tt, g.left+g.width*.2, g.top+g.priceHeight*.3, g.left+g.width*.6, g.top+g.priceHeight*.5)
	tt.Key(0, ui.KeyDelete)
	if len(app.chart.annotations) != 0 || app.chart.selectedAnnotation != 0 {
		t.Fatal("Delete key did not remove the selected annotation")
	}
}

func createTestTrendLine(t *testing.T, app *application, tt *ui.Tester, x1, y1, x2, y2 float32) {
	t.Helper()
	if err := tt.Click("趋势线"); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(x1, y1)
	tt.Move(x2, y2)
	tt.ClickAt(x2, y2)
	if len(app.chart.annotations) != 1 {
		t.Fatalf("created %d trend lines, want 1", len(app.chart.annotations))
	}
}

func rgbaRegionChangedNear(before []byte, after *image.RGBA, x, y float32) bool {
	centerX, centerY := int(math.Round(float64(x))), int(math.Round(float64(y)))
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			px, py := centerX+dx, centerY+dy
			if px < 0 || py < 0 || px >= after.Rect.Dx() || py >= after.Rect.Dy() {
				continue
			}
			offset := py*after.Stride + px*4
			if !bytes.Equal(before[offset:offset+4], after.Pix[offset:offset+4]) {
				return true
			}
		}
	}
	return false
}

func currentKlineGeometry(app *application) klineGeometry {
	return geometryFor(app.chart.plotWidth+klineRightGutter, app.chart.chartHeight, app.chart.showVolume)
}

func newAnnotationTester() (*application, *ui.Tester) {
	candles := demoCandles()
	app := &application{candles: candles, chart: newKlineState(len(candles))}
	tt := ui.NewTester(func(c *ui.Context) { app.klineChart(c) }, 900, 560)
	tt.Frame()
	return app, tt
}
