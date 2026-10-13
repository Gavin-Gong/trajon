package app

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	replayStepInterval  = time.Second
	replayDragThreshold = 5
)

func (s *klineState) selectReplayStart(index, count int) {
	if count <= 0 || index < 0 {
		return
	}
	index = max(0, min(index, count-1))
	s.exitReplaySelection()
	s.clearReplayTrading()
	s.replayHasStart = true
	s.replayStart = index
	s.replayEnd = index + 1
	s.replaySessionStarted = false
	s.replayComplete = false
	s.actionPlaying = false
	s.replayNextStep = time.Time{}
	s.hover = -1
	s.clamp(count)
}

func (s *klineState) exitReplaySelection() {
	s.replaySelecting = false
	s.replayDragMoved = false
	if s.replayDragging {
		s.replayDragging = false
		s.dragging = false
		s.dragRemainder = 0
	}
}

func (s *klineState) pauseReplay() {
	s.actionPlaying = false
	s.replayNextStep = time.Time{}
}

func (s *klineState) resetReplay() {
	if !s.replayHasStart {
		return
	}
	s.actionPlaying = false
	s.replayNextStep = time.Time{}
	s.replayEnd = s.replayStart + 1
	s.clearReplayTrading()
	s.replaySessionStarted = true
	s.replayComplete = false
	s.exitReplaySelection()
	s.hover = -1
}

func (s *klineState) endReplay(count int) {
	s.pauseReplay()
	s.exitReplaySelection()
	s.replayHasStart = false
	s.replayStart, s.replayEnd = 0, 0
	s.clearReplayTrading()
	s.replaySessionStarted = false
	s.replayComplete = false
	s.dragging = false
	s.dragRemainder = 0
	s.hover = -1
	s.clamp(count)
	s.Start = max(0, count-s.Visible)
}

func (a *application) startReplay(c *ui.Context) {
	state := &a.chart
	if !state.replayHasStart || state.replayComplete || state.replaySummaryOpen {
		return
	}
	state.exitReplaySelection()
	state.replayExitError = nil
	state.replaySessionStarted = true
	if state.replayEnd >= len(a.candles) {
		a.finishReplay(c)
		return
	}
	state.actionPlaying = true
	state.replayNextStep = c.Now().Add(replayInterval(state.replaySpeed))
}

func (a *application) advanceReplayFrame(c *ui.Context) {
	state := &a.chart
	if !state.actionPlaying {
		return
	}
	if state.replayNextStep.IsZero() {
		state.replayNextStep = c.Now().Add(replayInterval(state.replaySpeed))
	}
	if !c.Now().Before(state.replayNextStep) {
		beforeTrades := len(state.replayAccount.Trades)
		completed := a.advanceReplayStep()
		if state.replayExitError != nil {
			a.showReplayTradeMessage(c, "止盈止损执行失败")
			return
		}
		if len(state.replayAccount.Trades) > beforeTrades {
			trade := state.replayAccount.Trades[len(state.replayAccount.Trades)-1]
			a.showReplayTradeMessage(c, replayFillMessage(replayExitLabel(trade.ExitKind), trade))
		}
		if completed {
			c.Toast("回放结束")
			return
		}
		state.replayNextStep = c.Now().Add(replayInterval(state.replaySpeed))
	}
	c.After(max(time.Millisecond, state.replayNextStep.Sub(c.Now())))
}

func (a *application) advanceReplayStep() bool {
	state := &a.chart
	if !state.actionPlaying {
		return false
	}
	nextEnd := min(len(a.candles), state.replayEnd+1)
	if nextEnd > state.replayEnd {
		if _, err := state.replayAccount.CheckExits(a.replaySettings, a.candles[nextEnd-1]); err != nil {
			state.replayExitError = err
			state.pauseReplay()
			state.replayLineDrag = replayLineDrag{}
			return false
		}
		if state.replayAccount.Quantity == 0 {
			state.replayLineDrag = replayLineDrag{}
		}
	}
	state.replayEnd = nextEnd
	a.followReplay()
	if state.replayEnd >= len(a.candles) {
		a.completeReplay()
		return true
	}
	return false
}

func (a *application) completeReplay() {
	state := &a.chart
	state.actionPlaying = false
	state.replayNextStep = time.Time{}
	state.replayEnd = len(a.candles)
	state.replayComplete = true
	state.replayLineDrag = replayLineDrag{}
	state.replaySessionStarted = false
	state.exitReplaySelection()
	state.hover = -1
}

func (a *application) finishReplay(c *ui.Context) {
	a.completeReplay()
	c.Toast("回放结束")
}

func (a *application) followReplay() {
	state := &a.chart
	if len(a.candles) == 0 || state.Visible <= 0 || state.replayEnd <= 0 {
		return
	}
	lastVisible := state.replayEnd - 1
	if lastVisible < state.Start || lastVisible >= state.Start+state.Visible {
		state.Start = lastVisible - state.Visible + 1
		state.clamp(len(a.candles))
	}
}

func (a *application) panReplaySelectionDrag(state *klineState, x, y float32) {
	state.replayPointerX = x
	if !state.replayDragMoved {
		state.replayDragMoved = math.Hypot(float64(x-state.replayDownX), float64(y-state.replayDownY)) >= replayDragThreshold
	}
	if state.replayDragMoved {
		a.panKlineDrag(state, x)
	}
}

func (a *application) panKlineDrag(state *klineState, x float32) {
	if state.plotWidth <= 0 {
		return
	}
	dx := x - state.dragX
	state.dragX = x
	bars := -dx/state.plotWidth*float32(state.Visible) + state.dragRemainder
	whole := int(math.Trunc(float64(bars)))
	state.dragRemainder = bars - float32(whole)
	before := state.Start
	state.panBy(whole, len(a.candles))
	if whole != 0 && before == state.Start {
		state.dragRemainder = 0
	}
	if state.Start == 0 {
		a.loadOlderCandles()
	}
	state.hover = -1
}

func replayCandleIndexAtX(state *klineState, candles []Candle, x float32) int {
	if len(candles) == 0 || state.Visible <= 0 || state.plotWidth <= 0 {
		return -1
	}
	g := geometryForMode(state.plotWidth+klineRightGutter, state.chartHeight, state.showVolume, true)
	fraction := max(0, min(1, (x-g.left)/g.width))
	index := state.Start + min(state.Visible-1, int(fraction*float32(state.Visible)))
	return max(0, min(index, len(candles)-1))
}
