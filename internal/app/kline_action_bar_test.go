package app

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestKlineActionBarGroupSpacing(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.selectReplayStart(app.chart.Start+app.chart.Visible/2, len(app.candles))
	tt.Frame()

	for _, tc := range []struct {
		left, right string
		gap         float32
	}{
		{app.text().Play(), app.text().PlaybackSpeed(), 4},
		{app.text().PlaybackSpeed(), app.text().OrderQuantity(), 33},
		{app.text().Buy(), app.text().Sell(), 4},
		{app.text().Sell(), app.text().Reset(), 33},
		{app.text().Reset(), app.text().End(), 4},
	} {
		left, leftOK := tt.Find(tc.left)
		right, rightOK := tt.Find(tc.right)
		if !leftOK || !rightOK {
			t.Fatalf("missing action bar controls: %s=%v, %s=%v", tc.left, leftOK, tc.right, rightOK)
		}
		if gap := right.X - (left.X + left.W); gap != tc.gap {
			t.Errorf("gap between %s and %s = %v, want %v", tc.left, tc.right, gap, tc.gap)
		}
	}
}

func TestKlineActionBarFollowsChartResize(t *testing.T) {
	for _, replay := range []bool{false, true} {
		t.Run(fmt.Sprintf("replay=%v", replay), func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			if replay {
				app.chart.selectReplayStart(app.chart.Start+app.chart.Visible/2, len(app.candles))
				tt.Frame()
			}
			assertKlineActionBarAnchored(t, app, tt, 0, 0)
			for _, size := range [][2]int{{1200, 740}, {760, 520}, {1001, 701}, {900, 560}} {
				tt.SetSize(size[0], size[1])
				assertKlineActionBarAnchored(t, app, tt, 0, 0)
				tt.Frame()
				assertKlineActionBarAnchored(t, app, tt, 0, 0)
			}
		})
	}
}

func TestKlineActionBarFollowsSidebarChanges(t *testing.T) {
	for _, tc := range []struct{ replay, dragged bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		t.Run(fmt.Sprintf("replay=%v/dragged=%v", tc.replay, tc.dragged), func(t *testing.T) {
			app := &application{activePage: "market", candles: demoCandles(), chart: newKlineState(demoCandleCount)}
			sidebarWidth := float32(165)
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).Fill().Children(func() {
					if sidebarWidth > 0 {
						ui.Box(c.Key("test-sidebar")).Width(sidebarWidth).FillHeight()
					}
					ui.Box(c.Key("test-content")).Fill().Children(func() { app.content(c) })
				})
			}, 1120, 760)
			if tc.replay {
				app.chart.selectReplayStart(app.chart.Start+app.chart.Visible/2, len(app.candles))
				tt.Frame()
			}
			bar := assertKlineActionBarAnchored(t, app, tt, 0, 0)
			var offsetX, offsetY float32
			if tc.dragged {
				offsetX, offsetY = 40, -20
				x, y := bar.X+3, bar.Y+bar.H/2
				tt.Press(x, y)
				tt.Move(x+offsetX, y+offsetY)
				tt.Release(x+offsetX, y+offsetY)
			}
			for _, width := range []float32{0, 165, 227, 133, 0, 165} {
				sidebarWidth = width
				tt.Frame()
				assertKlineActionBarAnchored(t, app, tt, offsetX, offsetY)
			}
			if !tc.replay {
				if err := tt.Click(app.text().ReplayStart()); err != nil {
					t.Fatal(err)
				}
				if !app.chart.replaySelecting || app.chart.replayHasStart {
					t.Fatal("action bar click after sidebar changes did not enter replay-start selection")
				}
			}
		})
	}
}

func TestKlineActionBarPreservesDragOffsetAcrossResize(t *testing.T) {
	app, tt := newKlineToolbarTester()
	bar := assertKlineActionBarAnchored(t, app, tt, 0, 0)
	const offsetX, offsetY = float32(40), float32(-20)
	x, y := bar.X+3, bar.Y+bar.H/2
	initialStart := app.chart.Start
	tt.Press(x, y)
	tt.Move(x+offsetX, y+offsetY)
	tt.Release(x+offsetX, y+offsetY)
	assertKlineActionBarAnchored(t, app, tt, offsetX, offsetY)

	for _, size := range [][2]int{{1200, 740}, {760, 520}, {1001, 701}, {900, 560}} {
		tt.SetSize(size[0], size[1])
		bar = assertKlineActionBarAnchored(t, app, tt, offsetX, offsetY)
		if !pointerOverActionBar(&app.chart, bar.X+3, bar.Y+bar.H/2) || pointerOverActionBar(&app.chart, bar.X-1, bar.Y+bar.H/2) {
			t.Fatal("action bar hit region did not follow the resized bar")
		}
	}
	if app.chart.Start != initialStart || app.chart.dragging {
		t.Fatal("dragging or resizing the action bar also panned the chart")
	}

	app.chart.selectReplayStart(app.chart.Start+app.chart.Visible/2, len(app.candles))
	tt.Frame()
	assertKlineActionBarAnchored(t, app, tt, offsetX, offsetY)
	app.chart.endReplay(len(app.candles))
	tt.Frame()
	assertKlineActionBarAnchored(t, app, tt, offsetX, offsetY)
}

func TestKlineActionBarRestoresAnchorAfterClamping(t *testing.T) {
	for _, dragged := range []bool{false, true} {
		t.Run(fmt.Sprintf("dragged=%v", dragged), func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			app.chart.selectReplayStart(app.chart.Start+app.chart.Visible/2, len(app.candles))
			tt.Frame()
			bar := assertKlineActionBarAnchored(t, app, tt, 0, 0)
			var offsetX, offsetY float32
			if dragged {
				offsetX, offsetY = 40, -20
				x, y := bar.X+3, bar.Y+bar.H/2
				tt.Press(x, y)
				tt.Move(x+offsetX, y+offsetY)
				tt.Release(x+offsetX, y+offsetY)
			}
			for _, size := range [][2]int{{320, 80}, {1200, 740}, {900, 560}} {
				tt.SetSize(size[0], size[1])
				assertKlineActionBarAnchored(t, app, tt, offsetX, offsetY)
			}
		})
	}
}

func assertKlineActionBarAnchored(t *testing.T, app *application, tt *ui.Tester, offsetX, offsetY float32) ui.Rect {
	t.Helper()
	chart, chartOK := tt.Find(app.text().CandleChart(app.displayInstrument().Symbol))
	bar, barOK := tt.Find(app.text().ChartActions())
	if !chartOK || !barOK {
		t.Fatalf("chart or action bar is missing: chart=%v, bar=%v", chartOK, barOK)
	}
	wantX := chart.X + max(0, min(max(0, (chart.W-bar.W)/2)+offsetX, max(0, chart.W-bar.W)))
	wantY := chart.Y + max(0, min(max(0, chart.H-bar.H-klineActionBarInset)+offsetY, max(0, chart.H-bar.H)))
	if bar.X != wantX || bar.Y != wantY {
		t.Fatalf("action bar position = (%v, %v), want (%v, %v); chart=%+v, bar=%+v", bar.X, bar.Y, wantX, wantY, chart, bar)
	}
	return bar
}
