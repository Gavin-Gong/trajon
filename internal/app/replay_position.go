package app

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"trajon/internal/locales"
	"trajon/internal/market"
)

type replayLineKind uint8

const (
	replayPositionLine replayLineKind = iota + 1
	replayTakeProfitLine
	replayStopLossLine
)

type replayLineDrag struct {
	Source, Target replayLineKind
	DownY          float32
	Low, High      float64
	Price          float64
	Moved          bool
}

func replayPositionCaption(quantity float64, text locales.Messages) string {
	direction := text.Long()
	if quantity < 0 {
		direction = text.Short()
	}
	return direction + " " + strconv.FormatFloat(math.Abs(quantity), 'f', -1, 64)
}

func replayExitKind(kind replayLineKind) market.ReplayExitKind {
	if kind == replayTakeProfitLine {
		return market.ReplayTakeProfit
	}
	return market.ReplayStopLoss
}

func (a *application) closeReplayPosition() (market.ReplayTrade, error) {
	if !a.canTradeReplay() {
		return market.ReplayTrade{}, errReplayTradingUnavailable
	}
	candle := a.candles[a.chart.replayEnd-1]
	trade, err := a.chart.replayAccount.Close(a.replaySettings, candle.Close, candle.Time)
	if err == nil {
		a.chart.replayLineDrag = replayLineDrag{}
	}
	return trade, err
}

func replayFillMessage(label string, trade market.ReplayTrade) string {
	return label + " " + strconv.FormatFloat(trade.Quantity, 'f', -1, 64) +
		" @ " + strconv.FormatFloat(trade.Price, 'g', 8, 64)
}

func (a *application) submitReplayClose(c *ui.Context) {
	trade, err := a.closeReplayPosition()
	if err != nil {
		a.showReplayTradeMessage(c, a.text().CloseUnavailable())
		return
	}
	a.showReplayTradeMessage(c, replayFillMessage(a.text().ClosePosition(), trade))
}

func (a *application) pointerOverReplayLines(c *ui.Context, x, y float32) bool {
	state := &a.chart
	if !state.replayHasStart || state.replaySelecting || state.replayAccount.Quantity == 0 {
		return false
	}
	chartBounds := state.chartHandle.Bounds(c)
	x, y = x+chartBounds.X, y+chartBounds.Y
	for kind := replayPositionLine; kind <= replayStopLossLine; kind++ {
		if state.replayLineHandles[kind].Bounds(c).Contains(x, y) || state.replayLineTagHandles[kind].Bounds(c).Contains(x, y) {
			return true
		}
	}
	return false
}

func (a *application) beginReplayLineDrag(kind replayLineKind, y float32) bool {
	state := &a.chart
	if !a.canTradeReplay() || state.replayAccount.Quantity == 0 || state.textEditing || state.annotationGesture != annotationGestureNone {
		return false
	}
	low, high, ok := visibleKlinePriceRange(a.candles, state)
	if !ok {
		return false
	}
	state.replayLineDrag = replayLineDrag{Source: kind, Target: kind, DownY: y, Low: low, High: high}
	state.hover = -1
	return true
}

func (a *application) updateReplayLineDrag(y float32) {
	state := &a.chart
	drag := &state.replayLineDrag
	if drag.Source == 0 {
		return
	}
	if !a.canTradeReplay() || state.replayAccount.Quantity == 0 {
		state.replayLineDrag = replayLineDrag{}
		return
	}
	if !drag.Moved && math.Abs(float64(y-drag.DownY)) < replayDragThreshold {
		return
	}
	drag.Moved = true
	g := state.geometry(state.chartWidth, state.chartHeight, !a.marketToolbarInTitle)
	drag.Price = priceForY(y, g, drag.Low, drag.High)
	if drag.Source == replayPositionLine {
		profitable := (drag.Price > state.replayAccount.AveragePrice) == (state.replayAccount.Quantity > 0)
		drag.Target = replayStopLossLine
		if profitable {
			drag.Target = replayTakeProfitLine
		}
	}
	state.hover = -1
}

func (a *application) finishReplayLineDrag(c *ui.Context, y float32) {
	a.updateReplayLineDrag(y)
	drag := a.chart.replayLineDrag
	a.chart.replayLineDrag = replayLineDrag{}
	if !drag.Moved || !a.canTradeReplay() {
		return
	}
	stamp := a.candles[a.chart.replayEnd-1].Time
	if err := a.chart.replayAccount.SetExit(replayExitKind(drag.Target), drag.Price, stamp); err != nil {
		a.showReplayTradeMessage(c, a.text().InvalidExitPrice())
	}
}

func (a *application) replayLineInput(c *ui.Context, kind replayLineKind, ev ui.InputEvent) bool {
	state := &a.chart
	bounds := state.replayLineHandles[kind].Bounds(c)
	x, y := bounds.X+ev.X, bounds.Y+ev.Y
	chartY := y - state.chartHandle.Bounds(c).Y
	switch ev.Kind {
	case ui.InputPointerDown:
		// Leave native buttons in charge of focus, keyboard and activation.
		if ev.Button != 0 || state.replayLineCloseHandles[kind].Bounds(c).Contains(x, y) {
			return false
		}
		return a.beginReplayLineDrag(kind, chartY)
	case ui.InputPointerMove:
		if state.replayLineDrag.Source == kind {
			a.updateReplayLineDrag(chartY)
			return true
		}
	case ui.InputPointerUp:
		if state.replayLineDrag.Source == kind {
			a.finishReplayLineDrag(c, chartY)
			return true
		}
	case ui.InputKeyDown:
		if ev.Key == ui.KeyEscape && state.replayLineDrag.Source != 0 {
			state.replayLineDrag = replayLineDrag{}
			return true
		}
	case ui.InputScroll:
		return state.replayLineDrag.Source != 0
	}
	return false
}

func (a *application) replayLinePrices() [4]float64 {
	state := &a.chart
	prices := [4]float64{0, state.replayAccount.AveragePrice, state.replayAccount.TakeProfit.Price, state.replayAccount.StopLoss.Price}
	if drag := state.replayLineDrag; drag.Moved {
		prices[drag.Target] = drag.Price
	}
	return prices
}

func (a *application) replayPositionLines(c *ui.Context) {
	state := &a.chart
	if !state.replayHasStart || state.replaySelecting || state.replayAccount.Quantity == 0 {
		return
	}
	g := state.geometry(state.chartWidth, state.chartHeight, !a.marketToolbarInTitle)
	low, high, ok := visibleKlinePriceRange(a.candles, state)
	if !ok || g.width <= 0 || g.priceHeight <= 0 {
		return
	}
	prices := a.replayLinePrices()
	for kind := replayPositionLine; kind <= replayStopLossLine; kind++ {
		price := prices[kind]
		if price <= 0 {
			continue
		}
		label, caption, color := a.text().PositionLine(), a.replayPositionCaption(state.replayAccount.Quantity), c.Theme().Accent
		closeLabel := a.text().ClosePosition()
		if kind != replayPositionLine {
			label = a.replayExitLabel(replayExitKind(kind))
			pnl := (price - state.replayAccount.AveragePrice) * state.replayAccount.Quantity
			caption = fmt.Sprintf("%s %+.2f", label, pnl)
			closeLabel = a.text().CancelTakeProfit()
			if kind == replayStopLossLine {
				closeLabel = a.text().CancelStopLoss()
			}
			color = ui.Hex("#20a36a")
			if kind == replayStopLossLine {
				color = ui.Hex("#e5535d")
			}
		}
		textWidth, _ := c.MeasureText(0, ui.Span{Text: caption, Size: 10})
		width := min(float32(180), textWidth+30)
		y := g.top + float32((high-price)/(high-low))*g.priceHeight
		line := ui.Box(c.Key(kind)).Absolute().Left(g.left).Top(y-6).Size(g.width, 12).
			Bind(&state.replayLineHandles[kind]).Focusable().Role(ui.RoleGroup).Label(label).
			Cursor(ui.CursorResizeNS).Disabled(!a.canTradeReplay()).
			HandleInput(func(ev ui.InputEvent) bool { return a.replayLineInput(c, kind, ev) })
		stroke := float32(1)
		if line.Hovered() || state.replayLineDrag.Source == kind || (state.replayLineDrag.Moved && state.replayLineDrag.Target == kind) {
			stroke = 2
		}
		line.Draw(func(p *ui.Painter, r ui.Rect) {
			if kind == replayPositionLine {
				p.Line(r.X, r.Y+r.H/2, r.X+r.W, r.Y+r.H/2, stroke, color)
			} else {
				for x := r.X; x < r.X+r.W; x += 10 {
					p.Line(x, r.Y+r.H/2, min(x+6, r.X+r.W), r.Y+r.H/2, stroke, color)
				}
			}
		})
		line.Children(func() {
			ui.Row(c.Key("tag")).Absolute().Right(12).Top(-4).Size(width, 20).
				Bind(&state.replayLineTagHandles[kind]).Padding(0, 2, 0, 6).Gap(2).AlignItems(ui.Center).
				Radius(3).Border(1, color).Background(c.Theme().Background).Children(func() {
				components.Text(c, caption).FontSize(10).TextColor(color).SingleLine().Grow(1).MinWidth(0)
				components.Button(c.Key("close"), "", components.ButtonGhost).Size(18, 18).Padding(0).Radius(2).
					Bind(&state.replayLineCloseHandles[kind]).Label(closeLabel).TextColor(color).
					OnClick(func() {
						if kind == replayPositionLine {
							a.submitReplayClose(c)
						} else if a.canTradeReplay() {
							state.replayAccount.CancelExit(replayExitKind(kind))
							state.replayLineDrag = replayLineDrag{}
						}
					}).Children(func() {
					components.Icon(c, lucide.X).Size(10, 10)
				})
			})
		})
	}
}
