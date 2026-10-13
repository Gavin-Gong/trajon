package ui

import (
	myui "github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"trajon/internal/market"
)

type MarketStatusState struct {
	Loading      bool
	Error        error
	Pending      *market.CandleRequest
	OlderLoading bool
	OlderError   error
}

func RenderMarketStatus(c *myui.Context, state MarketStatusState, retryLatest, retryOlder func()) {
	if !state.Loading && state.Error == nil && !state.OlderLoading {
		return
	}
	myui.Column(c).Absolute().Left(0).Right(0).Top(76).Bottom(44).
		AlignItems(myui.Center).Justify(myui.Center).Gap(10).Children(func() {
		if state.Loading {
			components.Text(c, "正在加载行情…").TextColor(c.Theme().TextMuted).SingleLine()
			return
		}
		if state.Error != nil {
			message := "无法加载行情"
			if state.Pending != nil {
				message = "无法加载 " + state.Pending.Instrument.TypeLabel() + " · " + state.Pending.Instrument.Symbol
			}
			components.Text(c, message).TextColor(c.Theme().TextMuted).SingleLine()
			components.Text(c, state.Error.Error()).TextColor(c.Theme().TextMuted).Wrap().MaxWidth(420)
			retry := components.Button(c, "重试", components.ButtonSecondary)
			if retry.Clicked() && retryLatest != nil {
				retryLatest()
			}
			return
		}
		if state.OlderLoading {
			components.Text(c, "正在加载更早行情…").TextColor(c.Theme().TextMuted).SingleLine()
			return
		}
	})
}
