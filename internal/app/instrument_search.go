package app

import (
	"context"
	"strings"
	"time"

	"trajon/internal/market"
)

const instrumentSearchDelay = 250 * time.Millisecond

type instrumentSearchProvider interface {
	Search(context.Context, string) ([]market.Instrument, error)
}

type instrumentSearchState struct {
	provider   instrumentSearchProvider
	items      []instrument
	loading    bool
	err        error
	generation uint64
	cancel     context.CancelFunc
	updates    chan instrumentSearchResult
}

type instrumentSearchResult struct {
	generation uint64
	query      string
	items      []instrument
	err        error
}

func (a *application) stopInstrumentSearch() {
	if a.instrumentSearch.cancel != nil {
		a.instrumentSearch.cancel()
		a.instrumentSearch.cancel = nil
	}
	a.instrumentSearch.generation++
	a.instrumentSearch.loading = false
}

func (a *application) searchInstruments() {
	a.stopInstrumentSearch()
	a.instrumentSearch.items = nil
	a.instrumentSearch.err = nil
	a.instrumentPickerCursor = 0
	query := strings.TrimSpace(a.instrumentPickerQuery)
	if query == "" || !a.instrumentPickerOpen {
		return
	}
	if a.instrumentSearch.provider == nil {
		a.instrumentSearch.provider = market.NewInstrumentSearcher(a.dataSources)
	}
	if a.instrumentSearch.updates == nil {
		a.instrumentSearch.updates = make(chan instrumentSearchResult, 16)
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.instrumentSearch.cancel = cancel
	a.instrumentSearch.loading = true
	generation := a.instrumentSearch.generation
	provider, window, updates := a.instrumentSearch.provider, a.window, a.instrumentSearch.updates
	go func() {
		timer := time.NewTimer(instrumentSearchDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		items, err := provider.Search(ctx, query)
		if ctx.Err() != nil {
			return
		}
		result := instrumentSearchResult{generation: generation, query: query, items: items, err: err}
		if window != nil {
			window.Update(func() { a.applyInstrumentSearchResult(result) })
			return
		}
		select {
		case updates <- result:
		case <-ctx.Done():
		}
	}()
}

func (a *application) applyInstrumentSearchResult(result instrumentSearchResult) {
	if !a.instrumentPickerOpen || result.generation != a.instrumentSearch.generation ||
		result.query != strings.TrimSpace(a.instrumentPickerQuery) {
		return
	}
	if a.instrumentSearch.cancel != nil {
		a.instrumentSearch.cancel()
		a.instrumentSearch.cancel = nil
	}
	a.instrumentSearch.items = append([]instrument(nil), result.items...)
	a.instrumentSearch.err = result.err
	a.instrumentSearch.loading = false
	a.instrumentPickerCursor = 0
}

func (a *application) drainInstrumentSearchUpdates() {
	for a.instrumentSearch.updates != nil {
		select {
		case result := <-a.instrumentSearch.updates:
			a.applyInstrumentSearchResult(result)
		default:
			return
		}
	}
}
