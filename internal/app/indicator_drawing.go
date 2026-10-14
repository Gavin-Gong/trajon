package app

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

func drawEMAIndicators(p *ui.Painter, r ui.Rect, g klineGeometry, candles []Candle, state *klineState, priceAt func(float64) float32, fontSize float32) {
	if !state.indicators.EMAEnabled {
		return
	}
	start, end := state.Start, state.visibleEnd(len(candles))
	barWidth := g.width / float32(state.Visible)
	legendX := r.X + g.left + 4
	for _, line := range state.indicators.EMALines {
		if !line.Enabled {
			continue
		}
		var path ui.Path
		started := false
		for index := start; index < end; index++ {
			value, ok := state.emaValue(int(line.Period), index, candles)
			if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
				continue
			}
			x, y := r.X+g.left+(float32(index-start)+.5)*barWidth, priceAt(value)
			if !started {
				path.MoveTo(x, y)
				started = true
			} else {
				path.LineTo(x, y)
			}
		}
		if started {
			p.StrokePath(&path, float32(line.Width), line.Color)
		}
		label := fmt.Sprintf("EMA %.0f", line.Period)
		width, _ := p.MeasureText(0, ui.Span{Text: label, Size: fontSize})
		if legendX+width < r.X+g.left+g.width {
			p.Text(legendX, r.Y+g.top+3, label, fontSize, line.Color)
			legendX += width + 14
		}
	}
}

// macdVisibleRange only inspects revealed, visible candles, never future replay
// values. Zero is always included, and flat series retain a usable scale.
func macdVisibleRange(values *indicatorValues, start, end int) (float64, float64, bool) {
	if values == nil || start < 0 || start >= end || end > len(values.MACD.DIF) || end > len(values.MACD.DEA) || end > len(values.MACD.Histogram) {
		return 0, 0, false
	}
	low, high := 0.0, 0.0
	for _, series := range [][]float64{values.MACD.DIF, values.MACD.DEA, values.MACD.Histogram} {
		for _, value := range series[start:end] {
			if !math.IsNaN(value) && !math.IsInf(value, 0) {
				low, high = math.Min(low, value), math.Max(high, value)
			}
		}
	}
	if low == high {
		return -1, 1, true
	}
	padding := (high - low) * .08
	return low - padding, high + padding, true
}

func drawMACDIndicator(p *ui.Painter, r ui.Rect, g klineGeometry, state *klineState, start, end int, fontSize float32, t *ui.Theme) {
	if !state.indicators.MACDEnabled {
		return
	}
	settings := &state.indicators
	left, right := r.X+g.left, r.X+g.left+g.width
	grid := t.Border.Alpha(.55)
	p.Line(left, r.Y+g.macdTop, right, r.Y+g.macdTop, 1, grid)
	p.Line(left, r.Y+g.macdBottom, right, r.Y+g.macdBottom, 1, grid)
	label := fmt.Sprintf("MACD (%.0f, %.0f, %.0f)", settings.MACDFast, settings.MACDSlow, settings.MACDSignal)
	p.Text(left, r.Y+g.macdTop+4, label, fontSize, t.TextMuted)
	labelWidth, _ := p.MeasureText(0, ui.Span{Text: label, Size: fontSize})
	p.Text(left+labelWidth+14, r.Y+g.macdTop+4, "DIF", fontSize, settings.MACDDIF)
	p.Text(left+labelWidth+48, r.Y+g.macdTop+4, "DEA", fontSize, settings.MACDDEA)
	low, high, ok := macdVisibleRange(state.indicatorValues, start, end)
	if !ok {
		return
	}
	plotTop, plotHeight := r.Y+g.macdTop+22, g.macdBottom-g.macdTop-26
	valueAt := func(value float64) float32 { return plotTop + float32((high-value)/(high-low))*plotHeight }
	zeroY := valueAt(0)
	p.Line(left, zeroY, right, zeroY, 1, grid)
	p.Text(right+8, zeroY-fontSize*.55, "0", fontSize, t.TextMuted)
	p.Text(right+8, plotTop, fmt.Sprintf("%.2f", high), fontSize, t.TextMuted)
	p.Text(right+8, r.Y+g.macdBottom-fontSize, fmt.Sprintf("%.2f", low), fontSize, t.TextMuted)
	barWidth := g.width / float32(state.Visible)
	bodyWidth := max(float32(1), min(float32(10), barWidth*.72))
	var difPath, deaPath ui.Path
	for index := start; index < end; index++ {
		x := left + (float32(index-start)+.5)*barWidth
		histogram := state.indicatorValues.MACD.Histogram[index]
		color := settings.MACDPositive
		if histogram < 0 {
			color = settings.MACDNegative
		}
		y := valueAt(histogram)
		p.Fill(ui.Rect{X: x - bodyWidth/2, Y: min(y, zeroY), W: bodyWidth, H: max(1, absFloat32(y-zeroY))}, color.Alpha(.65), 0)
		difY, deaY := valueAt(state.indicatorValues.MACD.DIF[index]), valueAt(state.indicatorValues.MACD.DEA[index])
		if index == start {
			difPath.MoveTo(x, difY)
			deaPath.MoveTo(x, deaY)
		} else {
			difPath.LineTo(x, difY)
			deaPath.LineTo(x, deaY)
		}
	}
	p.StrokePath(&difPath, 1.5, settings.MACDDIF)
	p.StrokePath(&deaPath, 1.5, settings.MACDDEA)
}
