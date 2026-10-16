package ui

import (
	"errors"
	myui "github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"trajon/internal/locales"
	"trajon/internal/market"
)

type MarketStatusState struct {
	Loading      bool
	Error        error
	Pending      *market.CandleRequest
	OlderLoading bool
	OlderError   error
}

func RenderMarketStatus(c *myui.Context, state MarketStatusState, text locales.Messages, retryLatest, retryOlder func()) {
	if !state.Loading && state.Error == nil && !state.OlderLoading {
		return
	}
	myui.Column(c).Absolute().Left(0).Right(0).Top(76).Bottom(44).
		AlignItems(myui.Center).Justify(myui.Center).Gap(10).Children(func() {
		if state.Loading {
			components.Text(c, text.LoadingMarket()).TextColor(c.Theme().TextMuted).SingleLine()
			return
		}
		if state.Error != nil {
			message := text.MarketLoadFailed()
			if state.Pending != nil {
				message = text.InstrumentLoadFailed(InstrumentTypeLabel(text, state.Pending.Instrument), state.Pending.Instrument.Symbol)
			}
			components.Text(c, message).TextColor(c.Theme().TextMuted).SingleLine()
			components.Text(c, MarketError(text, state.Error)).TextColor(c.Theme().TextMuted).Wrap().MaxWidth(420)
			label := text.Retry()
			var authorization *market.SourceAuthorizationError
			if errors.As(state.Error, &authorization) {
				label = text.ConnectDataSource()
			}
			retry := components.Button(c.Key("market-retry"), label, components.ButtonSecondary)
			if retry.Clicked() && retryLatest != nil {
				retryLatest()
			}
			return
		}
		if state.OlderLoading {
			components.Text(c, text.LoadingOlderMarket()).TextColor(c.Theme().TextMuted).SingleLine()
			return
		}
	})
}
