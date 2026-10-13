package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"trajon/internal/chart"
	"trajon/internal/market"
)

const (
	initialVisibleCandles = chart.InitialVisibleCandles
	minVisibleCandles     = chart.MinVisibleCandles
)

type Candle = market.Candle

type klineState struct {
	chart.Viewport
	hover         int
	hoverY        float32
	showVolume    bool
	dragging      bool
	dragX         float32
	dragRemainder float32
	plotWidth     float32
	chartWidth    float32
	chartHeight   float32
	chartHandle   ui.Handle

	tool                  klineTool
	annotations           []klineAnnotation
	nextAnnotationID      uint64
	selectedAnnotation    uint64
	annotationGesture     annotationGesture
	gestureTarget         annotationTarget
	gestureID             uint64
	gestureDownX          float32
	gestureDownY          float32
	gestureOrigin         annotationAnchor
	gestureBase           klineAnnotation
	gestureStart          annotationAnchor
	gestureCurrent        annotationAnchor
	textEditing           bool
	textEditingNew        bool
	textEditingID         uint64
	textEditingAnchor     annotationAnchor
	textDraft             string
	textEditorX           float32
	textEditorY           float32
	lastAnnotationClick   time.Time
	lastAnnotationClickID uint64
	lastAnnotationClickX  float32
	lastAnnotationClickY  float32

	actionPlaying       bool
	actionBarPositioned bool
	actionBarX          float32
	actionBarY          float32
	actionBarWidth      float32
	timeframe           string

	replaySelecting         bool
	replayDragging          bool
	replayDragMoved         bool
	replayDownX             float32
	replayDownY             float32
	replayPointerX          float32
	replayHasStart          bool
	replayStart             int
	replayEnd               int
	replaySessionStarted    bool
	replayComplete          bool
	replayNextStep          time.Time
	replaySpeed             float64
	replayOrderQuantity     float64
	replayAccount           market.ReplayAccount
	replaySummaryOpen       bool
	replaySummary           market.ReplayStatistics
	replayTradeMessage      string
	replayTradeMessageUntil time.Time
	replayLineHandles       [4]ui.Handle
	replayLineTagHandles    [4]ui.Handle
	replayLineCloseHandles  [4]ui.Handle
	replayLineDrag          replayLineDrag
	replayExitError         error
}

func newKlineState(count int) klineState {
	return klineState{
		Viewport:            chart.NewViewport(count),
		hover:               -1,
		showVolume:          true,
		timeframe:           "1h",
		replaySpeed:         1,
		replayOrderQuantity: 1,
	}
}

func (s *klineState) clamp(count int) {
	s.Viewport.Clamp(count)
	if count <= 0 {
		s.hover = -1
		if s.replayHasStart {
			s.replayStart, s.replayEnd = 0, 0
			s.replayHasStart = false
			s.replaySessionStarted = false
			s.replayComplete = false
			s.replaySelecting = false
			s.replayDragging = false
			s.actionPlaying = false
			s.replayNextStep = time.Time{}
			s.clearReplayTrading()
		}
		return
	}
	if s.replayHasStart {
		s.replayStart = max(0, min(s.replayStart, count-1))
		s.replayEnd = max(s.replayStart+1, min(s.replayEnd, count))
	}
	hoverEnd := min(count, s.Start+s.Visible)
	if s.replayHasStart && !s.replayComplete && !s.replaySelecting {
		hoverEnd = min(hoverEnd, s.replayEnd)
	}
	if s.hover < s.Start || s.hover >= hoverEnd {
		s.hover = -1
	}
}

func (s *klineState) visibleEnd(count int) int {
	end := min(count, s.Start+s.Visible)
	if s.replayHasStart && !s.replayComplete && !s.replaySelecting {
		end = min(end, s.replayEnd)
	}
	return end
}

func (s *klineState) replayLocked() bool {
	return s.replayHasStart && !s.replayComplete
}

func (s *klineState) panBy(bars, count int) {
	s.Viewport.PanBy(bars, count)
	s.clamp(count)
}

func (s *klineState) zoomAt(zoomIn bool, anchor float32, count int) {
	s.Viewport.ZoomAt(zoomIn, anchor, count)
	s.clamp(count)
}
