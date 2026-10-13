package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

var replaySpeeds = [...]float64{.5, 1, 2, 5, 10}

func replayInterval(speed float64) time.Duration {
	for _, supported := range replaySpeeds {
		if speed == supported {
			return time.Duration(float64(replayStepInterval) / speed)
		}
	}
	return replayStepInterval
}

func (s *klineState) setReplaySpeed(speed float64, now time.Time) {
	if speed == s.replaySpeed {
		return
	}
	for _, supported := range replaySpeeds {
		if speed == supported {
			s.replaySpeed = speed
			if s.actionPlaying {
				s.replayNextStep = now.Add(replayInterval(speed))
			}
			return
		}
	}
}

func (a *application) replaySpeedControl(c *ui.Context) {
	state := &a.chart
	speed := state.replaySpeed
	sel := components.Select(c.Key("replay-speed"), &speed)
	sel.Trigger.Size(klineReplaySpeedWidth, 28).Padding(0, 18, 0, 6).Radius(14).FontSize(12).
		Label("播放速度").Tooltip("播放速度").Disabled(state.replayComplete).
		OnChange(func() { state.setReplaySpeed(speed, c.Now()) }).
		Children(func() {
			components.Text(c, fmt.Sprint(state.replaySpeed)+"×").SingleLine()
		})
	sel.Popup(func(panel ui.Element) {
		panel.Width(96)
		for _, option := range replaySpeeds {
			label := fmt.Sprint(option) + "×"
			sel.Item(option).Height(28).Padding(4, 8).Label(label).Disabled(state.replayComplete).Children(func() {
				components.Text(c, label).SingleLine()
			})
		}
	})
}
