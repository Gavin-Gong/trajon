package app

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

const maxIndicatorPeriod = 10000

type emaLineSettings struct {
	ID      uint64
	Enabled bool
	Period  float64
	Color   ui.Color
	Width   float64
}

type indicatorSettings struct {
	EMAEnabled                 bool
	EMALines                   []emaLineSettings
	NextEMAID                  uint64
	VolumeUp, VolumeDown       ui.Color
	MACDEnabled                bool
	MACDFast, MACDSlow         float64
	MACDSignal                 float64
	MACDDIF, MACDDEA           ui.Color
	MACDPositive, MACDNegative ui.Color
}

type indicatorDraft struct {
	indicatorSettings
	ShowVolume bool
}

func defaultIndicatorSettings() indicatorSettings {
	return indicatorSettings{
		EMAEnabled: true, NextEMAID: 1,
		EMALines: []emaLineSettings{{ID: 1, Enabled: true, Period: 20, Color: ui.Hex("#e4a83a"), Width: 1.5}},
		VolumeUp: ui.Hex("#20a36a"), VolumeDown: ui.Hex("#e5535d"),
		MACDFast: 12, MACDSlow: 26, MACDSignal: 9,
		MACDDIF: ui.Hex("#e4a83a"), MACDDEA: ui.Hex("#7196ef"),
		MACDPositive: ui.Hex("#20a36a"), MACDNegative: ui.Hex("#e5535d"),
	}
}

func (s indicatorSettings) clone() indicatorSettings {
	s.EMALines = slices.Clone(s.EMALines)
	return s
}

func validIndicatorPeriod(value float64) bool {
	return value >= 1 && value <= maxIndicatorPeriod && math.Trunc(value) == value
}

func (s indicatorSettings) valid() bool {
	for _, line := range s.EMALines {
		if !validIndicatorPeriod(line.Period) || math.IsNaN(line.Width) || line.Width < .5 || line.Width > 4 {
			return false
		}
	}
	return validIndicatorPeriod(s.MACDFast) && validIndicatorPeriod(s.MACDSlow) &&
		validIndicatorPeriod(s.MACDSignal) && s.MACDFast < s.MACDSlow
}

func (a *application) openIndicators() {
	a.indicatorDraft = indicatorDraft{indicatorSettings: a.chart.indicators.clone(), ShowVolume: a.chart.showVolume}
	a.indicatorsOpen = true
}

func (a *application) closeIndicators() {
	a.indicatorsOpen = false
	a.indicatorsButton.Focus()
}

func (a *application) saveIndicators() {
	if !a.indicatorsOpen || !a.indicatorDraft.valid() {
		return
	}
	a.chart.indicators = a.indicatorDraft.indicatorSettings.clone()
	a.chart.showVolume = a.indicatorDraft.ShowVolume
	a.chart.hover = -1
	a.closeIndicators()
}

func (a *application) addEMALine() {
	s := &a.indicatorDraft.indicatorSettings
	s.NextEMAID++
	periods := []float64{20, 50, 100, 200}
	colors := []ui.Color{ui.Hex("#e4a83a"), ui.Hex("#7196ef"), ui.Hex("#bc83e7"), ui.Hex("#35b9ba")}
	index := len(s.EMALines) % len(periods)
	s.EMALines = append(s.EMALines, emaLineSettings{ID: s.NextEMAID, Enabled: true, Period: periods[index], Color: colors[index], Width: 1.5})
}

func (a *application) removeEMALine(id uint64) {
	s := &a.indicatorDraft.indicatorSettings
	for i, line := range s.EMALines {
		if line.ID == id {
			s.EMALines = slices.Delete(s.EMALines, i, i+1)
			return
		}
	}
}

type indicatorValues struct {
	EMA  map[int][]float64
	MACD market.MACDSeries
}

type indicatorCalculation struct {
	Periods            []int
	Fast, Slow, Signal int
}

func (s indicatorSettings) calculation() indicatorCalculation {
	result := indicatorCalculation{Fast: int(s.MACDFast), Slow: int(s.MACDSlow), Signal: int(s.MACDSignal)}
	for _, line := range s.EMALines {
		result.Periods = append(result.Periods, int(line.Period))
	}
	slices.Sort(result.Periods)
	result.Periods = slices.Compact(result.Periods)
	return result
}

func (r indicatorCalculation) key() string {
	return fmt.Sprintf("%v/%d/%d/%d", r.Periods, r.Fast, r.Slow, r.Signal)
}

func calculateIndicators(ctx context.Context, candles []Candle, request indicatorCalculation) *indicatorValues {
	result := &indicatorValues{EMA: make(map[int][]float64, len(request.Periods))}
	for _, period := range request.Periods {
		if ctx.Err() != nil {
			return nil
		}
		result.EMA[period] = market.ComputeEMA(candles, period)
	}
	if ctx.Err() != nil {
		return nil
	}
	result.MACD = market.ComputeMACD(candles, request.Fast, request.Slow, request.Signal)
	return result
}

type indicatorCache struct {
	First      *Candle
	Count      int
	Key        string
	Values     *indicatorValues
	Generation uint64
	Cancel     context.CancelFunc
	Closed     bool
}

// ensureIndicators runs outside painting. Candle snapshots are replaced, not
// mutated, by the market loader; their backing-array identity invalidates this
// cache on source/timeframe changes and history prepends. Style/visibility edits
// reuse the same calculations. Headless views compute synchronously for tests.
func (a *application) ensureIndicators() {
	cache := &a.indicatorCache
	if cache.Closed {
		return
	}
	var first *Candle
	if len(a.candles) > 0 {
		first = &a.candles[0]
	}
	request := a.chart.indicators.calculation()
	key := request.key()
	if cache.First == first && cache.Count == len(a.candles) && cache.Key == key {
		a.chart.indicatorValues = cache.Values
		return
	}
	if cache.Cancel != nil {
		cache.Cancel()
	}
	cache.First, cache.Count, cache.Key = first, len(a.candles), key
	cache.Values, a.chart.indicatorValues = nil, nil
	cache.Generation++
	generation := cache.Generation
	ctx, cancel := context.WithCancel(context.Background())
	cache.Cancel = cancel
	if a.window == nil {
		a.applyIndicatorValues(generation, calculateIndicators(ctx, a.candles, request))
		return
	}
	candles, window := cloneCandles(a.candles), a.window
	go func() {
		values := calculateIndicators(ctx, candles, request)
		if ctx.Err() == nil {
			window.Update(func() { a.applyIndicatorValues(generation, values) })
		}
	}()
}

func (a *application) applyIndicatorValues(generation uint64, values *indicatorValues) {
	if a.indicatorCache.Closed || generation != a.indicatorCache.Generation {
		return
	}
	a.indicatorCache.Values = values
	a.chart.indicatorValues = values
}

func (a *application) closeIndicatorCalculations() {
	a.indicatorCache.Closed = true
	if a.indicatorCache.Cancel != nil {
		a.indicatorCache.Cancel()
	}
}

func (s *klineState) emaValues(period int) []float64 {
	if s.indicatorValues != nil {
		return s.indicatorValues.EMA[period]
	}
	return nil
}

// emaValue preserves the precomputed default line while an asynchronous
// calculation is pending. Other periods wait for their own result.
func (s *klineState) emaValue(period, index int, candles []Candle) (float64, bool) {
	if values := s.emaValues(period); index < len(values) {
		return values[index], true
	}
	if period == 20 && index < len(candles) {
		return candles[index].EMA20, true
	}
	return 0, false
}
