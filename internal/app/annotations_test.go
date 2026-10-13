package app

import (
	"math"
	"testing"
	"time"
)

func TestCandleTimePositionRoundTripKeepsContinuousTime(t *testing.T) {
	start := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: start},
		{Time: start.Add(time.Hour)},
		{Time: start.Add(2 * time.Hour)},
		{Time: start.Add(3 * time.Hour)},
	}
	want := start.Add(90 * time.Minute)
	position := candlePositionAtTime(candles, want)
	if math.Abs(position-1.5) > 1e-9 {
		t.Fatalf("fractional candle position = %v, want 1.5", position)
	}
	if got := candleTimeAtPosition(candles, position); !got.Equal(want) {
		t.Fatalf("round-trip time = %v, want %v", got, want)
	}
}

func TestAnnotationAnchorRoundTripsThroughViewport(t *testing.T) {
	candles := generateCandles(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 20)
	state := newKlineState(len(candles))
	state.Start, state.Visible = 4, 8
	g := geometryFor(900, klineChartHeight, true)
	wantX := g.left + g.width*.37
	wantY := g.top + g.priceHeight*.43

	anchor, ok := annotationAnchorAt(wantX, wantY, g, candles, &state)
	if !ok {
		t.Fatal("pointer inside price plot did not produce an anchor")
	}
	low, high, ok := visibleKlinePriceRange(candles, &state)
	if !ok {
		t.Fatal("visible price range was unavailable")
	}
	gotX, gotY, ok := annotationPoint(anchor, g, candles, &state, low, high)
	if !ok {
		t.Fatal("anchor did not map back to viewport")
	}
	if math.Abs(float64(gotX-wantX)) > .01 || math.Abs(float64(gotY-wantY)) > .01 {
		t.Fatalf("anchor maps to (%v, %v), want (%v, %v)", gotX, gotY, wantX, wantY)
	}
	if anchor.Time.Equal(candles[4].Time) || anchor.Time.Equal(candles[5].Time) {
		t.Fatalf("anchor time %v snapped to a candle; want continuous time", anchor.Time)
	}
}

func TestAnnotationAnchorMustBeInsidePriceArea(t *testing.T) {
	candles := demoCandles()
	state := newKlineState(len(candles))
	g := geometryFor(900, klineChartHeight, true)
	if _, ok := annotationAnchorAt(g.left+10, g.volumeTop+1, g, candles, &state); ok {
		t.Fatal("volume panel should not accept drawing anchors")
	}
}

func TestDistanceToSegment(t *testing.T) {
	if got := distanceToSegment(5, 2, 0, 0, 10, 0); math.Abs(float64(got-2)) > 1e-6 {
		t.Fatalf("distance above segment = %v, want 2", got)
	}
	if got := distanceToSegment(12, 0, 0, 0, 10, 0); math.Abs(float64(got-2)) > 1e-6 {
		t.Fatalf("distance beyond segment = %v, want 2", got)
	}
}
