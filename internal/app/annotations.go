package app

import (
	"math"
	"sort"
	"time"

	"github.com/egoist/mygo/ui"
)

type klineTool uint8

const (
	klineToolSelect klineTool = iota
	klineToolTrend
	klineToolText
)

type annotationKind uint8

const (
	annotationTrend annotationKind = iota
	annotationText
)

type annotationAnchor struct {
	Time  time.Time
	Price float64
}

type klineAnnotation struct {
	ID    uint64
	Kind  annotationKind
	Start annotationAnchor
	End   annotationAnchor
	Text  string
}

type annotationTarget uint8

const (
	annotationTargetNone annotationTarget = iota
	annotationTargetBody
	annotationTargetStart
	annotationTargetEnd
)

type annotationGesture uint8

const (
	annotationGestureNone annotationGesture = iota
	annotationGesturePan
	annotationGestureCreateTrend
	annotationGestureMove
	annotationGestureMoveStart
	annotationGestureMoveEnd
)

const (
	annotationHandleHitRadius = float32(9)
	annotationLineHitRadius   = float32(7)
	annotationTextFontSize    = float32(14)
	annotationTextHeight      = float32(20)
)

// candlePositionAtTime maps an arbitrary timestamp to a fractional candle
// index. Candle centers map to their timestamps; times between candles remain
// continuous instead of snapping to a bar.
func candlePositionAtTime(candles []Candle, timestamp time.Time) float64 {
	if len(candles) == 0 {
		return 0
	}
	if len(candles) == 1 {
		return 0
	}
	upper := sort.Search(len(candles), func(i int) bool {
		return !candles[i].Time.Before(timestamp)
	})
	if upper == 0 {
		interval := candles[1].Time.Sub(candles[0].Time)
		if interval <= 0 {
			return 0
		}
		return float64(timestamp.Sub(candles[0].Time)) / float64(interval)
	}
	if upper == len(candles) {
		last := len(candles) - 1
		interval := candles[last].Time.Sub(candles[last-1].Time)
		if interval <= 0 {
			return float64(last)
		}
		return float64(last) + float64(timestamp.Sub(candles[last].Time))/float64(interval)
	}
	before := upper - 1
	interval := candles[upper].Time.Sub(candles[before].Time)
	if interval <= 0 {
		return float64(before)
	}
	return float64(before) + float64(timestamp.Sub(candles[before].Time))/float64(interval)
}

// candleTimeAtPosition is the inverse of candlePositionAtTime, including
// interpolation and extrapolation so annotations can sit between candle
// centers and at the edges of the plot.
func candleTimeAtPosition(candles []Candle, position float64) time.Time {
	if len(candles) == 0 {
		return time.Time{}
	}
	if len(candles) == 1 {
		return candles[0].Time
	}
	left := int(math.Floor(position))
	fraction := position - float64(left)
	if left < 0 {
		interval := candles[1].Time.Sub(candles[0].Time)
		if interval <= 0 {
			return candles[0].Time
		}
		return candles[0].Time.Add(time.Duration(float64(interval) * position))
	}
	last := len(candles) - 1
	if left >= last {
		interval := candles[last].Time.Sub(candles[last-1].Time)
		if interval <= 0 {
			return candles[last].Time
		}
		return candles[last].Time.Add(time.Duration(float64(interval) * (position - float64(last))))
	}
	interval := candles[left+1].Time.Sub(candles[left].Time)
	if interval <= 0 {
		return candles[left].Time
	}
	return candles[left].Time.Add(time.Duration(float64(interval) * fraction))
}

func annotationAnchorAt(x, y float32, g klineGeometry, candles []Candle, state *klineState) (annotationAnchor, bool) {
	if !insideKlinePricePlot(x, y, g) || len(candles) == 0 || state.Visible <= 0 || g.width <= 0 {
		return annotationAnchor{}, false
	}
	low, high, ok := visibleKlinePriceRange(candles, state)
	if !ok {
		return annotationAnchor{}, false
	}
	position := float64(state.Start) + float64((x-g.left)/g.width)*float64(state.Visible) - .5
	return annotationAnchor{
		Time:  candleTimeAtPosition(candles, position),
		Price: priceForY(y, g, low, high),
	}, true
}

func annotationPoint(anchor annotationAnchor, g klineGeometry, candles []Candle, state *klineState, low, high float64) (float32, float32, bool) {
	if len(candles) == 0 || state.Visible <= 0 || g.width <= 0 || g.priceHeight <= 0 {
		return 0, 0, false
	}
	position := candlePositionAtTime(candles, anchor.Time)
	x := g.left + (float32(position-float64(state.Start)+.5)/float32(state.Visible))*g.width
	y := g.top + float32((high-anchor.Price)/(high-low))*g.priceHeight
	return x, y, true
}

func visibleKlinePriceRange(candles []Candle, state *klineState) (float64, float64, bool) {
	if drag := state.replayLineDrag; drag.Source != 0 && drag.High > drag.Low {
		return drag.Low, drag.High, true
	}
	if len(candles) == 0 || state.Visible <= 0 {
		return 0, 0, false
	}
	start := max(0, min(state.Start, len(candles)-1))
	end := state.visibleEnd(len(candles))
	if start >= end && state.replayAccount.Quantity == 0 {
		return 0, 0, false
	}
	low, high := math.MaxFloat64, -math.MaxFloat64
	if start < end {
		for _, candle := range candles[start:end] {
			low = math.Min(low, math.Min(candle.Low, candle.EMA20))
			high = math.Max(high, math.Max(candle.High, candle.EMA20))
		}
	}
	if state.replayAccount.Quantity != 0 {
		for _, price := range []float64{state.replayAccount.AveragePrice, state.replayAccount.TakeProfit.Price, state.replayAccount.StopLoss.Price} {
			if price > 0 {
				low = math.Min(low, price)
				high = math.Max(high, price)
			}
		}
	}
	if low == high {
		low--
		high++
	}
	padding := (high - low) * .06
	return low - padding, high + padding, true
}

func insideKlinePricePlot(x, y float32, g klineGeometry) bool {
	return x >= g.left && x <= g.left+g.width && y >= g.top && y <= g.priceBottom
}

func distanceToSegment(x, y, x1, y1, x2, y2 float32) float32 {
	dx, dy := x2-x1, y2-y1
	lengthSquared := dx*dx + dy*dy
	if lengthSquared == 0 {
		return float32(math.Hypot(float64(x-x1), float64(y-y1)))
	}
	t := max(0, min(1, ((x-x1)*dx+(y-y1)*dy)/lengthSquared))
	px, py := x1+t*dx, y1+t*dy
	return float32(math.Hypot(float64(x-px), float64(y-py)))
}

func annotationTextHitWidth(text string) float32 {
	return max(float32(18), float32(len([]rune(text)))*annotationTextFontSize*.62+8)
}

func annotationWithID(state *klineState, id uint64) *klineAnnotation {
	for i := range state.annotations {
		if state.annotations[i].ID == id {
			return &state.annotations[i]
		}
	}
	return nil
}

func addKlineAnnotation(state *klineState, annotation klineAnnotation) uint64 {
	state.nextAnnotationID++
	annotation.ID = state.nextAnnotationID
	state.annotations = append(state.annotations, annotation)
	state.selectedAnnotation = annotation.ID
	return annotation.ID
}

func deleteSelectedAnnotation(state *klineState) {
	if state.selectedAnnotation == 0 {
		return
	}
	for i := range state.annotations {
		if state.annotations[i].ID == state.selectedAnnotation {
			state.annotations = append(state.annotations[:i], state.annotations[i+1:]...)
			break
		}
	}
	if state.textEditingID == state.selectedAnnotation {
		finishAnnotationTextEdit(state, false)
	}
	state.selectedAnnotation = 0
}

func beginAnnotationTextEdit(state *klineState, id uint64, anchor annotationAnchor, isNew bool) {
	state.textEditing = true
	state.textEditingNew = isNew
	state.textEditingID = id
	state.textEditingAnchor = anchor
	state.textDraft = ""
	if annotation := annotationWithID(state, id); annotation != nil {
		state.textDraft = annotation.Text
		state.selectedAnnotation = id
	}
}

func finishAnnotationTextEdit(state *klineState, save bool) {
	if !state.textEditing {
		return
	}
	text := state.textDraft
	if save && text != "" {
		if state.textEditingNew {
			addKlineAnnotation(state, klineAnnotation{
				Kind:  annotationText,
				Start: state.textEditingAnchor,
				Text:  text,
			})
		} else if annotation := annotationWithID(state, state.textEditingID); annotation != nil {
			annotation.Text = text
			state.selectedAnnotation = annotation.ID
		}
	}
	state.textEditing = false
	state.textEditingNew = false
	state.textEditingID = 0
	state.textDraft = ""
}

func beginAnnotationGesture(state *klineState, id uint64, target annotationTarget, anchor annotationAnchor, x, y float32) {
	annotation := annotationWithID(state, id)
	if annotation == nil {
		return
	}
	state.gestureID = id
	state.gestureTarget = target
	state.gestureOrigin = anchor
	state.gestureBase = *annotation
	state.gestureDownX, state.gestureDownY = x, y
	state.gestureCurrent = anchor
	state.annotationGesture = annotationGestureMove
	if annotation.Kind == annotationTrend {
		switch target {
		case annotationTargetStart:
			state.annotationGesture = annotationGestureMoveStart
		case annotationTargetEnd:
			state.annotationGesture = annotationGestureMoveEnd
		}
	}
}

func updateAnnotationGesture(state *klineState, anchor annotationAnchor) {
	annotation := annotationWithID(state, state.gestureID)
	if annotation == nil {
		return
	}
	deltaTime := anchor.Time.Sub(state.gestureOrigin.Time)
	deltaPrice := anchor.Price - state.gestureOrigin.Price
	switch state.annotationGesture {
	case annotationGestureMove:
		annotation.Start = shiftedAnchor(state.gestureBase.Start, deltaTime, deltaPrice)
		if annotation.Kind == annotationTrend {
			annotation.End = shiftedAnchor(state.gestureBase.End, deltaTime, deltaPrice)
		}
	case annotationGestureMoveStart:
		annotation.Start = anchor
	case annotationGestureMoveEnd:
		annotation.End = anchor
	}
}

func shiftedAnchor(anchor annotationAnchor, deltaTime time.Duration, deltaPrice float64) annotationAnchor {
	anchor.Time = anchor.Time.Add(deltaTime)
	anchor.Price += deltaPrice
	return anchor
}

func isKlineAnnotationDoubleClick(state *klineState, id uint64, x, y float32) bool {
	now := time.Now()
	double := state.lastAnnotationClickID == id && !state.lastAnnotationClick.IsZero() &&
		now.Sub(state.lastAnnotationClick) < 500*time.Millisecond &&
		absFloat32(x-state.lastAnnotationClickX) < 5 && absFloat32(y-state.lastAnnotationClickY) < 5
	if double {
		resetAnnotationClickHistory(state)
		return true
	}
	state.lastAnnotationClick = now
	state.lastAnnotationClickID = id
	state.lastAnnotationClickX, state.lastAnnotationClickY = x, y
	return false
}

func resetAnnotationClickHistory(state *klineState) {
	state.lastAnnotationClick = time.Time{}
	state.lastAnnotationClickID = 0
}

func hitKlineAnnotation(state *klineState, candles []Candle, g klineGeometry, x, y float32) (uint64, annotationTarget) {
	low, high, ok := visibleKlinePriceRange(candles, state)
	if !ok {
		return 0, annotationTargetNone
	}
	for i := len(state.annotations) - 1; i >= 0; i-- {
		annotation := state.annotations[i]
		x1, y1, ok := annotationPoint(annotation.Start, g, candles, state, low, high)
		if !ok {
			continue
		}
		if annotation.Kind == annotationText {
			if x >= x1-4 && x <= x1+annotationTextHitWidth(annotation.Text) && y >= y1-4 && y <= y1+annotationTextHeight {
				return annotation.ID, annotationTargetBody
			}
			continue
		}
		x2, y2, ok := annotationPoint(annotation.End, g, candles, state, low, high)
		if !ok {
			continue
		}
		if math.Hypot(float64(x-x1), float64(y-y1)) <= float64(annotationHandleHitRadius) {
			return annotation.ID, annotationTargetStart
		}
		if math.Hypot(float64(x-x2), float64(y-y2)) <= float64(annotationHandleHitRadius) {
			return annotation.ID, annotationTargetEnd
		}
		if distanceToSegment(x, y, x1, y1, x2, y2) <= annotationLineHitRadius {
			return annotation.ID, annotationTargetBody
		}
	}
	return 0, annotationTargetNone
}

func pointerOverTextEditor(state *klineState, x, y float32) bool {
	if !state.textEditing {
		return false
	}
	return x >= state.textEditorX+1 && x <= state.textEditorX+181 &&
		y >= state.textEditorY+1 && y <= state.textEditorY+33
}

func drawKlineAnnotations(p *ui.Painter, r ui.Rect, g klineGeometry, candles []Candle, state *klineState, t *ui.Theme, low, high float64) {
	plot := ui.Rect{X: r.X + g.left, Y: r.Y + g.top, W: g.width, H: g.priceHeight}
	p.Clip(plot, 0, func() {
		for _, annotation := range state.annotations {
			if state.textEditing && !state.textEditingNew && state.textEditingID == annotation.ID {
				continue
			}
			x1, y1, ok := annotationPoint(annotation.Start, g, candles, state, low, high)
			if !ok {
				continue
			}
			x1, y1 = r.X+x1, r.Y+y1
			selected := state.selectedAnnotation == annotation.ID
			if annotation.Kind == annotationText {
				width, height := p.MeasureText(0, ui.Span{Text: annotation.Text, Size: annotationTextFontSize})
				label := ui.Rect{X: x1 - 4, Y: y1 - 2, W: width + 8, H: max(annotationTextHeight, height+4)}
				if selected {
					p.Fill(label, t.Accent.Alpha(.16), t.Radius/2)
					p.Stroke(label, t.Accent, t.Radius/2, 1)
				}
				p.Text(x1, y1, annotation.Text, annotationTextFontSize, t.Text)
				continue
			}
			x2, y2, ok := annotationPoint(annotation.End, g, candles, state, low, high)
			if !ok {
				continue
			}
			x2, y2 = r.X+x2, r.Y+y2
			if selected {
				drawAnnotationSegment(p, x1, y1, x2, y2, 5, t.Accent.Alpha(.18))
			}
			drawAnnotationSegment(p, x1, y1, x2, y2, 2, t.Accent)
			if selected {
				drawAnnotationHandle(p, x1, y1, t)
				drawAnnotationHandle(p, x2, y2, t)
			}
		}
		if state.annotationGesture == annotationGestureCreateTrend {
			x1, y1, ok1 := annotationPoint(state.gestureStart, g, candles, state, low, high)
			x2, y2, ok2 := annotationPoint(state.gestureCurrent, g, candles, state, low, high)
			if ok1 && ok2 {
				drawAnnotationSegment(p, r.X+x1, r.Y+y1, r.X+x2, r.Y+y2, 2, t.Accent.Alpha(.72))
				drawAnnotationHandle(p, r.X+x1, r.Y+y1, t)
			}
		}
	})
}

func drawAnnotationSegment(p *ui.Painter, x1, y1, x2, y2, width float32, color ui.Color) {
	var path ui.Path
	path.MoveTo(x1, y1).LineTo(x2, y2)
	p.StrokePath(&path, width, color)
}

func drawAnnotationHandle(p *ui.Painter, x, y float32, t *ui.Theme) {
	handle := ui.Rect{X: x - 4, Y: y - 4, W: 8, H: 8}
	p.Fill(handle, t.Background, 4)
	p.Stroke(handle, t.Accent, 4, 1.5)
}
