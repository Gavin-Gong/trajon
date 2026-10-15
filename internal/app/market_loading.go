package app

import (
	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
	appui "trajon/internal/ui"
)

const (
	binanceCandleLimit = market.DefaultCandleLimit
	olderErrorToastID  = "market-older-load-error"
)

func (a *application) displayInstrument() instrument {
	if a.currentInstrument.Symbol != "" {
		return a.currentInstrument
	}
	return instrument{Name: "Demo Asset", Symbol: "DEMO/USDT"}
}

func (a *application) ensureMarketLoaded() {
	if a.marketLoadStarted || a.marketLoader == nil {
		return
	}
	a.marketLoadStarted = true
	interval := a.chart.timeframe
	if interval == "" {
		interval = "1h"
		a.chart.timeframe = interval
	}
	a.loadMarketData(a.currentInstrument, interval, false)
}

func (a *application) selectInstrument(item instrument) {
	if a.chart.replayLocked() {
		return
	}
	a.pendingPreviousTimeframe = ""
	a.stopInstrumentSearch()
	a.pendingHistoryIdentity = item.Identity()
	a.instrumentPickerOpen = false
	a.instrumentPickerQuery = ""
	a.instrumentPickerTab = instrumentTabAll
	a.instrumentPickerCursor = 0
	interval := a.chart.timeframe
	if interval == "" {
		interval = "1h"
		a.chart.timeframe = interval
	}
	a.loadMarketData(item, interval, false)
}

func (a *application) loadMarketData(item instrument, interval string, force bool) {
	if a.marketLoader == nil {
		return
	}
	if interval == "" {
		interval = "1h"
	}
	a.marketLoadStarted = true
	a.marketLoader.Select(item, interval, force)
}

func (a *application) publishMarketSnapshot(snapshot market.Snapshot) {
	if a.window != nil {
		a.window.Update(func() {
			a.applyMarketSnapshot(snapshot)
		})
		return
	}
	if a.marketUpdates != nil {
		a.marketUpdates <- snapshot
	}
}

func (a *application) drainMarketUpdates() {
	for a.marketUpdates != nil {
		select {
		case snapshot := <-a.marketUpdates:
			a.applyMarketSnapshot(snapshot)
		default:
			return
		}
	}
}

func (a *application) applyMarketSnapshot(snapshot market.Snapshot) {
	wasLoading := a.marketLoading
	hadCandles := len(a.candles) > 0
	previousIdentity := a.currentInstrument.Identity()
	previousInterval := a.chart.timeframe
	previousOlderError := a.olderError
	if snapshot.HistoryPrepended > 0 {
		a.chart.Start += snapshot.HistoryPrepended
		if a.chart.replayHasStart {
			a.chart.replayStart += snapshot.HistoryPrepended
			a.chart.replayEnd += snapshot.HistoryPrepended
		}
	}
	if len(snapshot.Candles) > 0 || (!wasLoading && snapshot.Pending == nil && snapshot.Error == nil) {
		a.candles = cloneCandles(snapshot.Candles)
	}
	a.marketLoading = snapshot.Loading
	a.marketError = snapshot.Error
	a.pendingCandleLoad = cloneRequest(snapshot.Pending)
	a.olderLoading = snapshot.OlderLoading
	a.olderError = snapshot.OlderError
	a.olderExhausted = snapshot.OlderExhausted
	a.pendingOlderLoad = cloneRequest(snapshot.OlderPending)
	if snapshot.OlderError != nil && (previousOlderError == nil || previousOlderError.Error() != snapshot.OlderError.Error()) {
		a.pendingOlderErrorToast = true
	}

	committed := snapshot.Pending == nil && !snapshot.Loading && snapshot.Error == nil && snapshot.Instrument.SourceID != ""
	if committed {
		newSeries := !hadCandles || previousIdentity != snapshot.Instrument.Identity() || previousInterval != snapshot.Interval
		a.currentInstrument = snapshot.Instrument
		if a.pendingHistoryIdentity == snapshot.Instrument.Identity() {
			a.rememberInstrument(snapshot.Instrument)
			a.pendingHistoryIdentity = ""
		}
		if snapshot.Interval != "" {
			a.chart.timeframe = snapshot.Interval
		}
		if newSeries {
			showVolume, replaySpeed := a.chart.showVolume, a.chart.replaySpeed
			indicators := a.chart.indicators
			a.chart = newKlineState(len(a.candles))
			a.chart.indicators = indicators
			a.chart.timeframe = snapshot.Interval
			a.chart.showVolume = showVolume
			a.chart.replaySpeed = replaySpeed
		}
		a.pendingPreviousTimeframe = ""
	}
	if snapshot.Error != nil && a.pendingPreviousTimeframe != "" {
		a.chart.timeframe = a.pendingPreviousTimeframe
		a.pendingPreviousTimeframe = ""
	}
	a.chart.clamp(len(a.candles))
}

func cloneRequest(request *market.CandleRequest) *candleRequest {
	if request == nil {
		return nil
	}
	copyRequest := candleRequest(*request)
	return &copyRequest
}

func cloneCandles(candles []Candle) []Candle {
	return market.CloneCandles(candles)
}

func (a *application) retryMarketData() {
	if a.marketLoader != nil {
		a.marketLoader.RetryLatest()
	}
}

func (a *application) loadOlderCandles() {
	if a.marketLoader != nil {
		a.marketLoader.LoadOlder()
	}
}

func (a *application) retryOlderCandles() {
	if a.marketLoader != nil {
		a.marketLoader.RetryOlder()
	}
}

func (a *application) showPendingMarketToasts(c *ui.Context) {
	if !a.pendingOlderErrorToast {
		return
	}
	a.pendingOlderErrorToast = false
	if a.olderError == nil {
		return
	}
	c.AddToast(ui.Toast{
		ID:          olderErrorToastID,
		Title:       a.text().OlderMarketLoadFailed(),
		Description: appui.MarketError(a.text(), a.olderError),
		Type:        "error",
		Action:      a.text().Retry(),
		OnAction:    a.retryOlderCandles,
	})
}

func (a *application) marketStatus(c *ui.Context) {
	a.showPendingMarketToasts(c)
	appui.RenderMarketStatus(c, appui.MarketStatusState{
		Loading:      a.marketLoading,
		Error:        a.marketError,
		Pending:      (*market.CandleRequest)(a.pendingCandleLoad),
		OlderLoading: a.olderLoading,
		OlderError:   a.olderError,
	}, a.text(), a.retryMarketData, a.retryOlderCandles)
}
