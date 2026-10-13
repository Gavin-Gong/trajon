package app

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
)

const (
	klineChartHeight          = 500
	klineToolbarHeight        = float32(40)
	klineRightGutter          = float32(72)
	klineActionBarWidth       = float32(72)
	klineReplayQuantityWidth  = float32(88)
	klineReplayTradeBarWidth  = klineReplayQuantityWidth + 2*28 + 4*4 + 1
	klineReplayActionBarWidth = float32(104) + klineReplayTradeBarWidth
	klineActionBarHeight      = float32(40)
	klineActionBarInset       = float32(12)
)

type klineGeometry struct {
	left, top, width, priceHeight float32
	priceBottom                   float32
	volumeTop, volumeBottom       float32
	bottom                        float32
}

func geometryFor(width, height float32, showVolume bool) klineGeometry {
	return geometryForMode(width, height, showVolume, true)
}

func geometryForMode(width, height float32, showVolume, toolbarInChart bool) klineGeometry {
	top := float32(8)
	if toolbarInChart {
		top = 58
	}
	g := klineGeometry{left: 8, top: top, width: max(0, width-klineRightGutter), bottom: height - 28}
	if showVolume {
		volumeHeight := min(float32(82), height*.2)
		gap := float32(26)
		g.volumeBottom = g.bottom
		g.volumeTop = g.volumeBottom - volumeHeight
		g.priceBottom = g.volumeTop - gap
	} else {
		g.priceBottom = g.bottom
	}
	g.priceHeight = max(48, g.priceBottom-g.top)
	return g
}

func (a *application) marketPage(c *ui.Context) {
	a.ensureMarketLoaded()
	ui.Column(c).Fill().Children(func() {
		a.klineChart(c)
	})
}

func (a *application) klineChart(c *ui.Context) {
	a.advanceReplayFrame(c)
	t := c.Theme()
	state := &a.chart
	cursor := ui.CursorGrab
	if state.tool != klineToolSelect || state.replaySelecting {
		cursor = ui.CursorCrosshair
	}
	currentInstrument := a.displayInstrument()
	description := "滚轮缩放，拖动浏览历史；趋势线点击起点后移动预览，再点击终点完成"
	if state.replaySelecting {
		description = "拖动浏览历史，点击 K 线选择回放起点，按 Esc 退出"
	}
	chart := ui.Box(c).Key("kline-chart").Fill().
		Cursor(cursor).Focusable().Label(currentInstrument.Symbol + " K 线图").
		Description(description).Bind(&state.chartHandle)
	if state.replaySelecting {
		chart.OnShortcut(0, ui.KeyEscape, state.exitReplaySelection)
	}

	if x, y, over := chart.PointerPosition(); state.replaySelecting {
		state.replayPointerX = x
		state.hover = -1
	} else if state.replayLineDrag.Source != 0 || (over && a.pointerOverReplayLines(c, x, y)) {
		state.hover = -1
	} else if over && !state.dragging && state.annotationGesture == annotationGestureNone && (a.marketToolbarInTitle || !pointerOverKlineToolbar(state, x, y)) && !pointerOverActionBar(state, x, y) && !pointerOverTextEditor(state, x, y) && !a.pointerOverReplayLines(c, x, y) {
		updateKlineHover(state, a.candles, x, y, a.marketToolbarInTitle)
	} else if !over && !state.dragging && state.annotationGesture == annotationGestureNone {
		state.hover = -1
	}

	chart.HandleInput(func(ev ui.InputEvent) bool {
		// Child controls own pointer input over the toolbar, action bar, and editor.
		blockedByChild := (!a.marketToolbarInTitle && pointerOverKlineToolbar(state, ev.X, ev.Y)) || pointerOverActionBar(state, ev.X, ev.Y) || pointerOverTextEditor(state, ev.X, ev.Y) || state.replayLineDrag.Source != 0 || a.pointerOverReplayLines(c, ev.X, ev.Y)
		if blockedByChild && !(state.replayDragging && (ev.Kind == ui.InputPointerMove || ev.Kind == ui.InputPointerUp)) {
			if ev.Kind == ui.InputPointerDown {
				resetAnnotationClickHistory(state)
			}
			return false
		}
		g := geometryForMode(state.plotWidth+klineRightGutter, state.chartHeight, state.showVolume, !a.marketToolbarInTitle)
		if state.plotWidth <= 0 || state.chartHeight <= 0 {
			return false
		}
		switch ev.Kind {
		case ui.InputScroll:
			if ev.DY == 0 || state.textEditing {
				return false
			}
			anchor := float32((float64(ev.X) - float64(g.left)) / float64(state.plotWidth))
			state.zoomAt(ev.DY < 0, anchor, len(a.candles))
			state.hover = -1
			return true
		case ui.InputKeyDown:
			if state.textEditing {
				return false
			}
			if ev.Key == ui.KeyEscape && state.replaySelecting {
				state.exitReplaySelection()
				return true
			}
			if ev.Key == ui.KeyEscape && state.annotationGesture == annotationGestureCreateTrend {
				state.annotationGesture = annotationGestureNone
				state.tool = klineToolSelect
				state.hover = -1
				return true
			}
			if state.selectedAnnotation != 0 && (ev.Key == ui.KeyDelete || ev.Key == ui.KeyBackspace) {
				deleteSelectedAnnotation(state)
				return true
			}
		case ui.InputPointerDown:
			if ev.Button != 0 || !insideKlinePlot(ev.X, ev.Y, g) {
				return false
			}
			if state.replaySelecting {
				state.dragging = true
				state.replayDragging = true
				state.replayDragMoved = false
				state.replayDownX, state.replayDownY = ev.X, ev.Y
				state.replayPointerX = ev.X
				state.dragX = ev.X
				state.dragRemainder = 0
				state.hover = -1
				return true
			}
			if state.textEditing {
				return true
			}
			if state.tool == klineToolTrend || state.tool == klineToolText {
				anchor, ok := annotationAnchorAt(ev.X, ev.Y, g, a.candles, state)
				if !ok {
					return false
				}
				state.selectedAnnotation = 0
				state.hover = -1
				if state.tool == klineToolText {
					beginAnnotationTextEdit(state, 0, anchor, true)
					return true
				}
				if state.annotationGesture == annotationGestureCreateTrend {
					state.gestureCurrent = anchor
					if math.Hypot(float64(ev.X-state.gestureDownX), float64(ev.Y-state.gestureDownY)) >= 4 {
						addKlineAnnotation(state, klineAnnotation{
							Kind:  annotationTrend,
							Start: state.gestureStart,
							End:   anchor,
						})
					}
					state.annotationGesture = annotationGestureNone
					state.tool = klineToolSelect
					return true
				}
				state.annotationGesture = annotationGestureCreateTrend
				state.gestureStart, state.gestureCurrent = anchor, anchor
				state.gestureDownX, state.gestureDownY = ev.X, ev.Y
				return true
			}

			if insideKlinePricePlot(ev.X, ev.Y, g) {
				if id, target := hitKlineAnnotation(state, a.candles, g, ev.X, ev.Y); id != 0 {
					state.selectedAnnotation = id
					annotation := annotationWithID(state, id)
					doubleClick := ev.Clicks >= 2 || isKlineAnnotationDoubleClick(state, id, ev.X, ev.Y)
					if annotation != nil && annotation.Kind == annotationText && doubleClick {
						beginAnnotationTextEdit(state, id, annotation.Start, false)
						return true
					}
					anchor, ok := annotationAnchorAt(ev.X, ev.Y, g, a.candles, state)
					if !ok {
						return true
					}
					beginAnnotationGesture(state, id, target, anchor, ev.X, ev.Y)
					return true
				}
			}
			resetAnnotationClickHistory(state)
			state.selectedAnnotation = 0
			state.dragging = true
			state.replayDragging = false
			state.dragX = ev.X
			state.dragRemainder = 0
			return true
		case ui.InputPointerMove:
			if state.annotationGesture != annotationGestureNone {
				if state.annotationGesture != annotationGestureCreateTrend &&
					math.Hypot(float64(ev.X-state.gestureDownX), float64(ev.Y-state.gestureDownY)) >= 5 {
					resetAnnotationClickHistory(state)
				}
				anchor, ok := annotationAnchorAt(ev.X, ev.Y, g, a.candles, state)
				if ok {
					state.gestureCurrent = anchor
					if state.annotationGesture != annotationGestureCreateTrend {
						updateAnnotationGesture(state, anchor)
					}
				}
				return true
			}
			if state.dragging {
				if state.replayDragging {
					a.panReplaySelectionDrag(state, ev.X, ev.Y)
				} else {
					a.panKlineDrag(state, ev.X)
				}
				return true
			}
			return updateKlineHover(state, a.candles, ev.X, ev.Y, a.marketToolbarInTitle)
		case ui.InputPointerUp:
			if state.annotationGesture == annotationGestureCreateTrend {
				// Keep the first click's anchor armed after releasing the button.
				// Pointer movement now drives a rubber-band preview until the next click.
				return true
			}
			if state.annotationGesture != annotationGestureNone {
				state.annotationGesture = annotationGestureNone
				state.gestureID = 0
				return true
			}
			if state.dragging {
				if state.replayDragging {
					a.panReplaySelectionDrag(state, ev.X, ev.Y)
					if !state.replayDragMoved && insideKlinePlot(ev.X, ev.Y, g) {
						index := replayCandleIndexAtX(state, a.candles, ev.X)
						state.selectReplayStart(index, len(a.candles))
					}
					state.replayDragging = false
					state.replayDragMoved = false
				}
				state.dragging = false
				state.dragRemainder = 0
				return true
			}
		}
		return false
	})
	if state.dragging || state.annotationGesture == annotationGestureMove || state.annotationGesture == annotationGestureMoveStart || state.annotationGesture == annotationGestureMoveEnd {
		chart.Cursor(ui.CursorGrabbing)
	}
	chart.Draw(func(p *ui.Painter, r ui.Rect) {
		g := geometryForMode(r.W, r.H, state.showVolume, !a.marketToolbarInTitle)
		if state.chartWidth != r.W || state.chartHeight != r.H {
			state.chartWidth = r.W
			state.chartHeight = r.H
			c.Invalidate()
		}
		state.plotWidth = g.width
		drawKline(p, r, g, a.candles, state, t)
		drawReplaySelection(p, r, g, state, t)
	})
	chart.Children(func() {
		a.replayPositionLines(c)
		if !a.marketToolbarInTitle {
			a.klineToolbar(c)
		}
		a.klineActionBar(c)
		a.replayTradeFeedback(c)
		if state.textEditing {
			a.klineTextEditor(c)
		}
		a.marketStatus(c)
	})
}

func (a *application) klineActionBar(c *ui.Context) {
	state := &a.chart
	chartBounds := ui.Rect{W: state.chartWidth, H: state.chartHeight}
	width := klineActionBarWidth
	if state.replayHasStart {
		width = klineReplayActionBarWidth
	}
	if state.actionBarPositioned {
		// Keep the bar's center when the replay controls appear or disappear.
		state.actionBarX += (state.actionBarWidth - width) / 2
	}
	state.actionBarWidth = width
	bar := ui.Row(c).Key("kline-action-bar").Size(width, klineActionBarHeight).
		Padding(6).Gap(4).AlignItems(ui.Center).Justify(ui.Center).
		Radius(klineActionBarHeight/2).Background(c.Theme().Background.Alpha(.78)).
		Border(1, c.Theme().Border.Alpha(.55)).
		Shadow(0, 2, 10, 0, c.Theme().Text.Alpha(.1)).
		Role(ui.RoleGroup).Label("K 线图操作栏").Cursor(ui.CursorGrab)

	if chartBounds.W > 0 && chartBounds.H > 0 {
		if !state.actionBarPositioned {
			state.actionBarX, state.actionBarY = defaultKlineActionBarPosition(chartBounds, width)
			state.actionBarPositioned = true
		}
		clampKlineActionBarPosition(state, chartBounds)
	}
	if dx, dy, dragged := bar.Dragged(); dragged && state.actionBarPositioned {
		state.actionBarX += dx
		state.actionBarY += dy
		clampKlineActionBarPosition(state, chartBounds)
	}
	if state.actionBarPositioned {
		bar.Absolute().Left(state.actionBarX).Top(state.actionBarY)
	} else {
		bar.Absolute().Attach(ui.AnchorBottom, ui.AnchorBottom).Bottom(klineActionBarInset)
	}
	if bar.Pressed() {
		bar.Cursor(ui.CursorGrabbing)
	}

	bar.Children(func() {
		if !state.replayHasStart {
			replayTool := klineActionIconButton(c.Key("replay-start"), lucide.ClockArrowLeft, "回放起点", state.replaySelecting)
			if state.replaySelecting {
				replayTool.OnShortcut(0, ui.KeyEscape, state.exitReplaySelection)
			}
			replayTool.OnClick(func() {
				if state.replaySelecting {
					state.exitReplaySelection()
				} else {
					finishAnnotationTextEdit(state, true)
					state.annotationGesture = annotationGestureNone
					state.tool = klineToolSelect
					state.replaySelecting = true
					state.hover = -1
				}
			})
		} else {
			playLabel, playIcon := "播放", lucide.Play
			if state.actionPlaying {
				playLabel, playIcon = "暂停", lucide.Pause
			}
			klineActionIconButton(c.Key("replay-play"), playIcon, playLabel, false).
				Disabled(state.replayComplete).OnClick(func() {
				if state.actionPlaying {
					state.pauseReplay()
				} else {
					a.startReplay(c)
				}
			})
			klineActionIconButton(c.Key("replay-reset"), lucide.RotateCcw, "重置", false).
				Disabled(!state.replaySessionStarted && !state.replayComplete && len(state.replayAccount.Trades) == 0).
				OnClick(state.resetReplay)
			klineActionIconButton(c.Key("replay-end"), lucide.Square, "结束", false).
				OnClick(func() { state.endReplay(len(a.candles)) })
		}
		if state.canEditReplaySettings() {
			klineActionIconButton(c.Key("replay-settings"), lucide.Settings2, "回放设置", false).
				Bind(&a.replaySettingsButton).OnClick(a.openReplaySettings)
		}
		if state.replayHasStart {
			a.replayTradingControls(c)
		}
	})
}

func defaultKlineActionBarPosition(chartBounds ui.Rect, width float32) (float32, float32) {
	innerWidth := max(0, chartBounds.W)
	innerHeight := max(0, chartBounds.H)
	x := max(0, (innerWidth-width)/2)
	y := max(0, innerHeight-klineActionBarHeight-klineActionBarInset)
	return x, y
}

func clampKlineActionBarPosition(state *klineState, chartBounds ui.Rect) {
	innerWidth := max(0, chartBounds.W)
	innerHeight := max(0, chartBounds.H)
	state.actionBarX = max(0, min(state.actionBarX, max(0, innerWidth-state.actionBarWidth)))
	state.actionBarY = max(0, min(state.actionBarY, max(0, innerHeight-klineActionBarHeight)))
}

func pointerOverActionBar(state *klineState, x, y float32) bool {
	if !state.actionBarPositioned {
		return false
	}
	return x >= state.actionBarX && x <= state.actionBarX+state.actionBarWidth &&
		y >= state.actionBarY && y <= state.actionBarY+klineActionBarHeight
}

func (a *application) klineToolbar(c *ui.Context) {
	a.klineToolbarLayout(c, false)
}

func (a *application) klineToolbarInTitle(c *ui.Context) {
	a.klineToolbarLayout(c, true)
}

func (a *application) klineToolbarLayout(c *ui.Context, inTitle bool) {
	state := &a.chart
	t := c.Theme()
	toolbar := ui.Row(c).Key("kline-toolbar").Role(ui.RoleToolbar).Label("K 线图工具栏")
	if inTitle {
		toolbar.FillWidth().FillHeight().Padding(0, 0).AlignItems(ui.Center)
	} else {
		toolbar.Absolute().Left(12).Top(4).
			Size(max(0, state.chartWidth-24), klineToolbarHeight).
			AlignItems(ui.Center)
	}
	toolbar.Gap(6)

	toolbar.Children(func() {
		currentInstrument := a.displayInstrument()
		instrumentButton := components.Button(c.Key("kline-instrument"), "", components.ButtonGhost).
			Padding(4, 8).Radius(t.Radius).MinWidth(0).
			Label("选择标的 " + currentInstrument.Symbol).
			Disabled(state.replayLocked())
		instrumentButton.Children(func() {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				components.Text(c, currentInstrument.Symbol).FontSize(17).Bold().SingleLine()
				components.Icon(c, lucide.ChevronDown).Size(13, 13).TextColor(t.TextMuted)
			})
		})
		if instrumentButton.Clicked() {
			a.openInstrumentPicker()
		}
		klineToolbarDivider(c)

		ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
			for _, period := range []string{"15m", "1h", "4h", "1D"} {
				selected := state.timeframe == period
				button := components.Button(c, period, components.ButtonGhost).
					Size(32, 26).Padding(0).Radius(t.Radius).Selected(selected).
					Disabled(state.replayLocked())
				if selected {
					button.Background(t.Text.Alpha(.12)).TextColor(t.Text)
				}
				if button.Clicked() {
					previousTimeframe := state.timeframe
					state.timeframe = period
					state.hover = -1
					if a.marketLoader != nil && a.marketLoadStarted && currentInstrument.SourceID != "" {
						a.pendingPreviousTimeframe = previousTimeframe
						a.loadMarketData(currentInstrument, period, false)
					}
				}
			}
		})
		klineToolbarDivider(c)

		trendTool := klineActionIconButton(c, lucide.TrendingUp, "趋势线", state.tool == klineToolTrend).Disabled(state.replaySelecting)
		if trendTool.Clicked() {
			resetAnnotationClickHistory(state)
			finishAnnotationTextEdit(state, true)
			state.annotationGesture = annotationGestureNone
			state.tool = klineToolTrend
		}
		textTool := klineActionIconButton(c, lucide.Type, "文本", state.tool == klineToolText).Disabled(state.replaySelecting)
		if textTool.Clicked() {
			resetAnnotationClickHistory(state)
			finishAnnotationTextEdit(state, true)
			state.annotationGesture = annotationGestureNone
			state.tool = klineToolText
		}

		klineToolbarDivider(c)
		remove := klineActionIconButton(c, lucide.Trash, "删除标注", false).Disabled(state.replaySelecting)
		if remove.Clicked() {
			resetAnnotationClickHistory(state)
			deleteSelectedAnnotation(state)
		}

		ui.Spacer(c)
		components.Switch(c, &state.showVolume, "成交量")
	})
}

func klineToolbarDivider(c *ui.Context) {
	ui.Box(c).Size(1, 20).Background(c.Theme().Border)
}

func klineActionIconButton(c *ui.Context, icon func() *ui.SVG, label string, selected bool) ui.Element {
	button := components.Button(c, "", components.ButtonGhost).
		Size(28, 28).Padding(0).Radius(14).Center().Label(label).Selected(selected)
	if selected {
		button.Background(c.Theme().Text.Alpha(.12)).TextColor(c.Theme().Text)
	}
	button.Children(func() {
		components.Icon(c, icon).Size(14, 14)
	})
	components.Tooltip(c, button, label)
	return button
}

func (a *application) klineTextEditor(c *ui.Context) {
	state := &a.chart
	g := geometryForMode(state.plotWidth+klineRightGutter, state.chartHeight, state.showVolume, !a.marketToolbarInTitle)
	low, high, ok := visibleKlinePriceRange(a.candles, state)
	if !ok {
		return
	}
	x, y, ok := annotationPoint(state.textEditingAnchor, g, a.candles, state, low, high)
	if !ok {
		return
	}
	const width, height = float32(180), float32(32)
	state.textEditorX = max(4, min(x, state.chartWidth-width-4))
	state.textEditorY = max(4, min(y-height/2, state.chartHeight-height-4))
	input := ui.TextInput(c, &state.textDraft).
		Absolute().Left(state.textEditorX).Top(state.textEditorY).Size(width, height).
		Placeholder("输入文字").Label("标注文字").AutoFocus().
		Background(c.Theme().Surface).Border(1, c.Theme().Accent).Radius(c.Theme().Radius / 2)
	input.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEscape {
			finishAnnotationTextEdit(state, false)
			state.tool = klineToolSelect
			return true
		}
		return false
	})
	input.OnSubmit(func() {
		finishAnnotationTextEdit(state, true)
		state.tool = klineToolSelect
	})
}

func pointerOverKlineToolbar(state *klineState, x, y float32) bool {
	return x >= 12 && x <= state.chartWidth-12 && y >= 4 && y <= 4+klineToolbarHeight
}

func insideKlinePlot(x, y float32, g klineGeometry) bool {
	return x >= g.left && x <= g.left+g.width && y >= g.top && y <= g.bottom
}

func updateKlineHover(state *klineState, candles []Candle, x, y float32, toolbarInTitle bool) bool {
	if state.replaySelecting {
		changed := state.replayPointerX != x || state.hover != -1
		state.replayPointerX = x
		state.hover = -1
		return changed
	}
	g := geometryForMode(state.plotWidth+klineRightGutter, state.chartHeight, state.showVolume, !toolbarInTitle)
	if !insideKlinePlot(x, y, g) || state.Visible <= 0 || g.width <= 0 {
		if state.hover != -1 {
			state.hover = -1
			return true
		}
		return false
	}
	fraction := max(0, min(1, (x-g.left)/g.width))
	index := state.Start + min(state.Visible-1, int(fraction*float32(state.Visible)))
	index = max(0, min(index, len(candles)-1))
	if !state.replaySelecting && index >= state.visibleEnd(len(candles)) {
		index = -1
	}
	changed := state.hover != index || state.hoverY != y
	state.hover, state.hoverY = index, y
	return changed
}

func drawKline(p *ui.Painter, r ui.Rect, g klineGeometry, candles []Candle, state *klineState, t *ui.Theme) {
	count := len(candles)
	if count == 0 || state.Visible <= 0 || g.width <= 0 {
		return
	}
	state.clamp(count)
	start, end := state.Start, state.visibleEnd(count)
	if start >= end {
		return
	}

	priceLow, priceHigh, hasPriceRange := visibleKlinePriceRange(candles, state)
	if !hasPriceRange {
		return
	}
	volumeHigh := 0.0
	for _, candle := range candles[start:end] {
		volumeHigh = math.Max(volumeHigh, candle.Volume)
	}

	upColor := ui.Hex("#20a36a")
	downColor := ui.Hex("#e5535d")
	emaColor := ui.Hex("#e4a83a")
	gridColor := t.Border.Alpha(.55)
	mutedColor := t.TextMuted
	fontSize := max(float32(9), t.FontSize-3)

	priceAt := func(value float64) float32 {
		fraction := (priceHigh - value) / (priceHigh - priceLow)
		return r.Y + g.top + float32(fraction)*g.priceHeight
	}
	plotLeft, plotRight := r.X+g.left, r.X+g.left+g.width
	for i := 0; i <= 4; i++ {
		fraction := float64(i) / 4
		value := priceHigh - fraction*(priceHigh-priceLow)
		y := r.Y + g.top + float32(fraction)*g.priceHeight
		p.Line(plotLeft, y, plotRight, y, 1, gridColor)
		p.Text(plotRight+8, y-fontSize*.55, fmt.Sprintf("%.2f", value), fontSize, mutedColor)
	}
	p.Line(plotLeft, r.Y+g.priceBottom, plotRight, r.Y+g.priceBottom, 1, gridColor)

	barWidth := g.width / float32(state.Visible)
	if state.hover >= start && state.hover < end {
		centerX := plotLeft + (float32(state.hover-start)+.5)*barWidth
		p.Fill(ui.Rect{X: centerX - barWidth/2, Y: r.Y + g.top, W: barWidth, H: g.bottom - g.top}, t.Text.Alpha(.045), 0)
	}
	bodyWidth := max(float32(1), min(float32(10), barWidth*.72))
	var emaPath ui.Path
	for index := start; index < end; index++ {
		candle := candles[index]
		local := float32(index - start)
		centerX := plotLeft + (local+.5)*barWidth
		color := upColor
		if candle.Close < candle.Open {
			color = downColor
		}
		p.Line(centerX, priceAt(candle.High), centerX, priceAt(candle.Low), 1, color)
		openY, closeY := priceAt(candle.Open), priceAt(candle.Close)
		bodyY := min(openY, closeY)
		bodyHeight := max(float32(1), absFloat32(closeY-openY))
		p.Fill(ui.Rect{X: centerX - bodyWidth/2, Y: bodyY, W: bodyWidth, H: bodyHeight}, color, 0)
		emaY := priceAt(candle.EMA20)
		if index == start {
			emaPath.MoveTo(centerX, emaY)
		} else {
			emaPath.LineTo(centerX, emaY)
		}
	}
	p.StrokePath(&emaPath, 1.5, emaColor)

	if state.showVolume {
		p.Line(plotLeft, r.Y+g.volumeTop, plotRight, r.Y+g.volumeTop, 1, gridColor)
		p.Text(plotLeft, r.Y+g.volumeTop+4, "成交量", fontSize, mutedColor)
		if volumeHigh > 0 {
			for index := start; index < end; index++ {
				candle := candles[index]
				local := float32(index - start)
				centerX := plotLeft + (local+.5)*barWidth
				barHeight := float32(candle.Volume/volumeHigh) * (g.volumeBottom - g.volumeTop - 8)
				color := upColor
				if candle.Close < candle.Open {
					color = downColor
				}
				p.Fill(ui.Rect{X: centerX - bodyWidth/2, Y: r.Y + g.volumeBottom - barHeight, W: bodyWidth, H: barHeight}, color.Alpha(.48), 0)
			}
		}
		p.Line(plotLeft, r.Y+g.volumeBottom, plotRight, r.Y+g.volumeBottom, 1, gridColor)
	}

	for i := 0; i <= 4; i++ {
		index := start + (end-start-1)*i/4
		label := candles[index].Time.Local().Format("01-02")
		textWidth, _ := p.MeasureText(0, ui.Span{Text: label, Size: fontSize})
		x := plotLeft + (float32(index-start)+.5)*barWidth - textWidth/2
		x = max(r.X+2, min(x, r.X+r.W-textWidth-2))
		p.Text(x, r.Y+g.bottom+5, label, fontSize, mutedColor)
	}

	if state.hover >= start && state.hover < end {
		local := float32(state.hover - start)
		centerX := plotLeft + (local+.5)*barWidth
		p.Line(centerX, r.Y+g.top, centerX, r.Y+g.bottom, 1, t.TextMuted.Alpha(.75))
		drawHoverTimeTag(p, r, centerX, r.Y+g.bottom+2, formatCandleTime(candles[state.hover].Time), fontSize, t)
		if state.hoverY >= g.top && state.hoverY <= g.priceBottom {
			p.Line(plotLeft, r.Y+state.hoverY, plotRight, r.Y+state.hoverY, 1, t.TextMuted.Alpha(.65))
			price := priceForY(state.hoverY, g, priceLow, priceHigh)
			drawHoverPriceTag(p, r, plotRight+2, r.Y+state.hoverY-9, fmt.Sprintf("%.2f", price), fontSize, t)
		}
	}

	drawKlineAnnotations(p, r, g, candles, state, t, priceLow, priceHigh)
}

func drawReplaySelection(p *ui.Painter, r ui.Rect, g klineGeometry, state *klineState, t *ui.Theme) {
	if !state.replaySelecting || state.Visible <= 0 || g.width <= 0 {
		return
	}
	plotLeft, plotRight := r.X+g.left, r.X+g.left+g.width
	markerX := max(plotLeft, min(plotRight, r.X+state.replayPointerX))
	p.Fill(ui.Rect{X: markerX, Y: r.Y + g.top, W: plotRight - markerX, H: g.bottom - g.top}, ui.Hex("#808080").Alpha(.24), 0)
	p.Line(markerX, r.Y+g.top, markerX, r.Y+g.bottom, 2, t.Accent.Alpha(.72))
}

func priceForY(y float32, g klineGeometry, low, high float64) float64 {
	if g.priceHeight <= 0 {
		return low
	}
	fraction := max(0, min(1, (y-g.top)/g.priceHeight))
	return high - float64(fraction)*(high-low)
}

func drawHoverTimeTag(p *ui.Painter, r ui.Rect, centerX, y float32, text string, fontSize float32, t *ui.Theme) {
	textWidth, _ := p.MeasureText(0, ui.Span{Text: text, Size: fontSize})
	width, height := textWidth+12, fontSize+6
	x := max(r.X+2, min(centerX-width/2, r.X+r.W-width-2))
	y = max(r.Y+2, min(y, r.Y+r.H-height-2))
	p.Fill(ui.Rect{X: x, Y: y, W: width, H: height}, t.Accent, t.Radius/2)
	p.Text(x+6, y+3, text, fontSize, t.AccentText)
}

func drawHoverPriceTag(p *ui.Painter, r ui.Rect, x, y float32, text string, fontSize float32, t *ui.Theme) {
	textWidth, _ := p.MeasureText(0, ui.Span{Text: text, Size: fontSize})
	width, height := min(textWidth+12, r.X+r.W-x-2), fontSize+6
	if width <= 0 {
		return
	}
	x = max(r.X+2, min(x, r.X+r.W-width-2))
	y = max(r.Y+2, min(y, r.Y+r.H-height-2))
	p.Fill(ui.Rect{X: x, Y: y, W: width, H: height}, t.Accent, t.Radius/2)
	p.Text(x+6, y+3, text, fontSize, t.AccentText)
}

func absFloat32(value float32) float32 {
	return float32(math.Abs(float64(value)))
}

func formatCandleTime(value time.Time) string {
	return value.Local().Format("2006-01-02 15:04")
}
