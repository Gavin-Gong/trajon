package app

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
	"github.com/oneo-me/onevra/source/components"
	"trajon/internal/journal"
)

func (a *application) overviewTrend(c *ui.Context, result *journal.Performance, now time.Time) {
	text, theme := a.text(), c.Theme()
	if len(result.Points) == 0 {
		ui.Column(c).FillWidth().Height(236).Center().Children(func() { components.Text(c, "—").TextColor(theme.TextMuted) })
		return
	}
	start, end := overviewRange(a.overview.period, now)
	if start.IsZero() {
		date := result.Points[0].At.Local()
		start = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	}
	if end.IsZero() || end.After(now) {
		end = now
	}
	if last := result.Points[len(result.Points)-1].At; last.After(end) {
		end = last
	}
	if !end.After(start) {
		end = start.Add(time.Hour)
	}
	lo, hi := 0.0, 0.0
	for _, point := range result.Points {
		lo, hi = min(lo, point.Cumulative), max(hi, point.Cumulative)
	}
	if lo == hi {
		lo, hi = lo-1, hi+1
	}
	y := chart.NewLinear(lo, hi).Nice(4)
	if math.IsInf(y.Hi-y.Lo, 0) || math.IsNaN(y.Hi-y.Lo) {
		return
	}
	y.Format = func(value float64) string {
		if math.Abs(value) >= 1e7 {
			return fmt.Sprintf("%.1e", value)
		}
		if hi-lo < 1 {
			return fmt.Sprintf("%.2g", value)
		}
		if hi-lo < 100 {
			return fmt.Sprintf("%.2f", value)
		}
		return fmt.Sprintf("%.0f", value)
	}
	xs := make([]float32, len(result.Points))
	for i, point := range result.Points {
		xs[i] = float32(point.At.Sub(start).Seconds() / end.Sub(start).Seconds())
	}
	axes := chart.Axes{Left: y.Ticks(4), Label: func(_ chart.Side, tick chart.Tick) {
		components.Text(c, tick.Label).FontSize(10).TextColor(theme.TextMuted).NoWrap()
	}}
	for _, fraction := range []float32{0, .5, 1} {
		at := start.Add(time.Duration(float64(end.Sub(start)) * float64(fraction)))
		axes.Bottom = append(axes.Bottom, chart.Tick{At: fraction, Label: at.Format("01-02")})
	}
	chart.Plot(c.Key("pnl-plot"), axes, func(area ui.Element) {
		area.Label(text.OverviewCumulativePnL())
		chart.Select(c, area, &a.overview.trendIndex, xs, func(i int) string {
			point := result.Points[i]
			return point.At.Local().Format("2006-01-02 15:04") + " · " + text.OverviewMoney(overviewMoney(point.Cumulative))
		})
		selected := a.overview.trendIndex
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tick := range axes.Left {
				shade := theme.Border.Alpha(.5)
				if tick.Value == 0 {
					shade = theme.Border
				}
				p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, shade)
			}
			var path ui.Path
			lastY := chart.Y(r, y.At(0))
			path.MoveTo(r.X, lastY)
			for i, point := range result.Points {
				x, nextY := chart.X(r, xs[i]), chart.Y(r, y.At(point.Cumulative))
				path.LineTo(x, lastY).LineTo(x, nextY)
				lastY = nextY
			}
			path.LineTo(r.X+r.W, lastY)
			p.StrokePath(&path, 2, theme.Accent)
			if selected >= 0 && selected < len(xs) {
				x := chart.X(r, xs[selected])
				p.Line(x, r.Y, x, r.Y+r.H, 1, theme.TextMuted.Alpha(.5))
				p.Fill(ui.Rect{X: x - 3, Y: chart.Y(r, y.At(result.Points[selected].Cumulative)) - 3, W: 6, H: 6}, theme.Accent, 3)
			}
		})
		if selected >= 0 && selected < len(result.Points) {
			point := result.Points[selected]
			components.Tooltip(c, area, point.At.Local().Format("2006-01-02 15:04")+"\n"+text.OverviewCumulativePnL()+": "+
				text.OverviewMoney(overviewMoney(point.Cumulative))+"\n"+text.OverviewTradeCount(point.Count))
		}
	}).FillWidth().Height(236).Gap(8).Padding(4, 12, 0, 0)
}

func (a *application) overviewOutcomeRing(c *ui.Context, result *journal.Performance, colors []ui.Color) {
	var values []float64
	var shades []ui.Color
	for i, count := range []int{result.Wins, result.Losses, result.Breakeven} {
		if count > 0 {
			values = append(values, float64(count))
			shades = append(shades, colors[i])
		}
	}
	pad := 0.0
	if len(values) > 1 {
		pad = .025
	}
	slices := chart.Pie(values, pad)
	track := c.Theme().Border
	ui.Box(c).Size(120, 120).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, outer := r.X+r.W/2, r.Y+r.H/2, min(r.W, r.H)/2
		if result.Count == 0 {
			p.FillPath(chart.Sector(cx, cy, outer-10, outer, chart.Slice{End: 2 * math.Pi}), track)
		}
		for i, slice := range slices {
			p.FillPath(chart.Sector(cx, cy, outer-10, outer, slice), shades[i])
		}
	}).Children(func() {
		ui.Column(c).Fill().Center().Gap(4).Children(func() {
			rate, ok := result.WinRate()
			components.Text(c, replaySummaryPercent(rate, ok)).FontSize(21).Bold().SingleLine()
			components.Text(c, a.text().WinRate()).FontSize(11).TextColor(c.Theme().TextMuted)
		})
	})
}
