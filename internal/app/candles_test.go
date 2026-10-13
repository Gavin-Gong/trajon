package app

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

const demoCandleCount = 3000

func generateCandles(start time.Time, count int) []Candle {
	if count <= 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(20250101))
	candles := make([]Candle, count)
	price := 100.0
	ema := price
	alpha := 2.0 / 21.0
	for i := range candles {
		open := price
		change := (rng.Float64() - 0.5) * 0.018
		close := math.Max(1, open*(1+change))
		high := math.Max(open, close) * (1 + rng.Float64()*0.004)
		low := math.Min(open, close) * (1 - rng.Float64()*0.004)
		volume := 500 + rng.Float64()*9500
		ema += alpha * (close - ema)
		candles[i] = Candle{
			Time: start.Add(time.Duration(i) * time.Hour), Open: open, High: high, Low: low,
			Close: close, Volume: volume, EMA20: ema,
		}
		price = close
	}
	return candles
}

func demoCandles() []Candle {
	return generateCandles(time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC), demoCandleCount)
}

func TestDemoCandlesAreDeterministicAndValid(t *testing.T) {
	first, second := demoCandles(), demoCandles()
	if len(first) != demoCandleCount {
		t.Fatalf("got %d candles, want %d", len(first), demoCandleCount)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("demo candles changed between generations")
	}

	wantStart := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !first[0].Time.Equal(wantStart) {
		t.Errorf("first candle time = %s, want %s", first[0].Time, wantStart)
	}
	for i, candle := range first {
		if candle.High < candle.Open || candle.High < candle.Close {
			t.Errorf("candle %d high is below open/close: %+v", i, candle)
		}
		if candle.Low > candle.Open || candle.Low > candle.Close {
			t.Errorf("candle %d low is above open/close: %+v", i, candle)
		}
		if candle.Volume <= 0 {
			t.Errorf("candle %d has non-positive volume: %v", i, candle.Volume)
		}
		if i > 0 && !candle.Time.Equal(first[i-1].Time.Add(time.Hour)) {
			t.Errorf("candle %d is not one hour after the previous candle", i)
		}
	}
}

func TestKlineStateZoomAndPanStayWithinData(t *testing.T) {
	const count = 3000
	state := newKlineState(count)
	if state.Visible != initialVisibleCandles || state.Start != count-initialVisibleCandles {
		t.Fatalf("initial viewport = start %d, visible %d", state.Start, state.Visible)
	}

	state.zoomAt(true, .5, count)
	if state.Visible >= initialVisibleCandles {
		t.Fatalf("zoom in left %d candles visible", state.Visible)
	}
	state.zoomAt(false, .5, count)
	if state.Visible < initialVisibleCandles-1 || state.Visible > initialVisibleCandles+1 {
		t.Fatalf("zoom out did not approximately restore visible range: %d", state.Visible)
	}

	for range 100 {
		state.zoomAt(true, .5, count)
	}
	if state.Visible != minVisibleCandles {
		t.Fatalf("minimum zoom range = %d, want %d", state.Visible, minVisibleCandles)
	}
	for range 100 {
		state.zoomAt(false, .5, count)
	}
	if state.Visible != count {
		t.Fatalf("maximum zoom range = %d, want %d", state.Visible, count)
	}

	state.panBy(-count*2, count)
	if state.Start != 0 {
		t.Errorf("panned before data start: %d", state.Start)
	}
	state.panBy(count*2, count)
	if state.Start != 0 {
		t.Errorf("all-data range should remain at start: %d", state.Start)
	}
}
