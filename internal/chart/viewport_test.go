package chart

import "testing"

func TestViewportZoomAndPanStayWithinData(t *testing.T) {
	const count = 3000
	viewport := NewViewport(count)
	if viewport.Visible != InitialVisibleCandles || viewport.Start != count-InitialVisibleCandles {
		t.Fatalf("initial viewport = start %d, visible %d", viewport.Start, viewport.Visible)
	}

	viewport.ZoomAt(true, .5, count)
	if viewport.Visible >= InitialVisibleCandles {
		t.Fatalf("zoom in left %d candles visible", viewport.Visible)
	}
	viewport.ZoomAt(false, .5, count)
	if viewport.Visible < InitialVisibleCandles-1 || viewport.Visible > InitialVisibleCandles+1 {
		t.Fatalf("zoom out did not approximately restore visible range: %d", viewport.Visible)
	}

	for range 100 {
		viewport.ZoomAt(true, .5, count)
	}
	if viewport.Visible != MinVisibleCandles {
		t.Fatalf("minimum zoom range = %d, want %d", viewport.Visible, MinVisibleCandles)
	}
	for range 100 {
		viewport.ZoomAt(false, .5, count)
	}
	if viewport.Visible != count {
		t.Fatalf("maximum zoom range = %d, want %d", viewport.Visible, count)
	}

	viewport.PanBy(-count*2, count)
	if viewport.Start != 0 {
		t.Errorf("panned before data start: %d", viewport.Start)
	}
	viewport.PanBy(count*2, count)
	if viewport.Start != 0 {
		t.Errorf("all-data range should remain at start: %d", viewport.Start)
	}
}
