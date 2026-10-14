package market

import (
	"math"
	"reflect"
	"testing"
)

func indicatorTestCandles() []Candle {
	return []Candle{{Close: 10}, {Close: 12}, {Close: 11}, {Close: 15}}
}

func assertIndicatorSeries(t *testing.T, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-10 {
			t.Errorf("value[%d] = %.12f, want %.12f", i, got[i], want[i])
		}
	}
}

func TestComputeEMA(t *testing.T) {
	candles := indicatorTestCandles()
	before := CloneCandles(candles)
	assertIndicatorSeries(t, ComputeEMA(candles, 3), []float64{10, 11, 11, 13})
	assertIndicatorSeries(t, ComputeEMA(candles, 1), []float64{10, 12, 11, 15})
	if !reflect.DeepEqual(candles, before) {
		t.Fatal("EMA mutated candles")
	}
	for _, period := range []int{0, -1} {
		if ComputeEMA(candles, period) != nil {
			t.Fatalf("invalid period %d accepted", period)
		}
	}
	if ComputeEMA(nil, 20) != nil {
		t.Fatal("empty EMA should be nil")
	}
	assertIndicatorSeries(t, ComputeEMA(candles[:1], 10000), []float64{10})
	AddEMA20(candles)
	for i, value := range ComputeEMA(candles, 20) {
		if candles[i].EMA20 != value {
			t.Fatal("default EMA differs from configurable EMA20")
		}
	}
}

func TestComputeMACD(t *testing.T) {
	candles := indicatorTestCandles()
	before := CloneCandles(candles)
	values := ComputeMACD(candles, 2, 3, 2)
	assertIndicatorSeries(t, values.DIF, []float64{0, 1.0 / 3, 1.0 / 9, 19.0 / 27})
	assertIndicatorSeries(t, values.DEA, []float64{0, 2.0 / 9, 4.0 / 27, 14.0 / 27})
	assertIndicatorSeries(t, values.Histogram, []float64{0, 2.0 / 9, -2.0 / 27, 10.0 / 27})
	if !reflect.DeepEqual(candles, before) {
		t.Fatal("MACD mutated candles")
	}
	constant := ComputeMACD([]Candle{{Close: 42}, {Close: 42}, {Close: 42}}, 12, 26, 9)
	assertIndicatorSeries(t, constant.DIF, []float64{0, 0, 0})
	assertIndicatorSeries(t, constant.DEA, []float64{0, 0, 0})
	assertIndicatorSeries(t, constant.Histogram, []float64{0, 0, 0})
	for _, periods := range [][3]int{{0, 26, 9}, {12, 12, 9}, {26, 12, 9}, {12, 26, 0}, {12, -1, 9}} {
		if got := ComputeMACD(candles, periods[0], periods[1], periods[2]); got.DIF != nil || got.DEA != nil || got.Histogram != nil {
			t.Fatal("invalid MACD periods accepted", periods)
		}
	}
	if ComputeMACD(nil, 12, 26, 9).DIF != nil {
		t.Fatal("empty MACD should be empty")
	}
}

func TestIndicatorsDoNotDependOnFutureCandles(t *testing.T) {
	candles := indicatorTestCandles()
	prefix := candles[:3]
	assertIndicatorSeries(t, ComputeEMA(candles, 3)[:3], ComputeEMA(prefix, 3))
	all, partial := ComputeMACD(candles, 2, 3, 2), ComputeMACD(prefix, 2, 3, 2)
	assertIndicatorSeries(t, all.DIF[:3], partial.DIF)
	assertIndicatorSeries(t, all.DEA[:3], partial.DEA)
	assertIndicatorSeries(t, all.Histogram[:3], partial.Histogram)
}
