package chart

import "math"

const (
	InitialVisibleCandles = 240
	MinVisibleCandles     = 20
)

type Viewport struct {
	Start   int
	Visible int
}

func NewViewport(count int) Viewport {
	visible := min(InitialVisibleCandles, count)
	return Viewport{
		Start:   max(0, count-visible),
		Visible: visible,
	}
}

func (v *Viewport) Clamp(count int) {
	if count <= 0 {
		v.Start, v.Visible = 0, 0
		return
	}
	v.Visible = min(count, max(MinVisibleCandles, v.Visible))
	v.Start = max(0, min(v.Start, count-v.Visible))
}

func (v *Viewport) PanBy(bars, count int) {
	v.Start += bars
	v.Clamp(count)
}

func (v *Viewport) ZoomAt(zoomIn bool, anchor float32, count int) {
	if count <= 0 {
		return
	}
	v.Clamp(count)
	oldVisible := v.Visible
	factor := 1.2
	newVisible := oldVisible
	if zoomIn {
		newVisible = int(math.Round(float64(oldVisible) / factor))
	} else {
		newVisible = int(math.Round(float64(oldVisible) * factor))
	}
	newVisible = max(MinVisibleCandles, min(newVisible, count))
	if newVisible == oldVisible {
		return
	}
	if anchor < 0 {
		anchor = 0
	} else if anchor > 1 {
		anchor = 1
	}
	absolute := float64(v.Start) + float64(anchor)*float64(oldVisible)
	v.Visible = newVisible
	v.Start = int(math.Round(absolute - float64(anchor)*float64(newVisible)))
	v.Clamp(count)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
