package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

type instrumentSearchFunc func(context.Context, string) ([]market.Instrument, error)

func (f instrumentSearchFunc) Search(ctx context.Context, query string) ([]market.Instrument, error) {
	return f(ctx, query)
}

func TestInstrumentSearchDebouncesRetriesPartialResultsAndSupportsKeyboardSelection(t *testing.T) {
	eth := historyETH()
	futures := eth
	futures.MarketType, futures.ContractType = marketFutures, contractUSDM
	candles := demoCandles()
	sources := newMarketDataRegistry(map[dataSourceID]MarketDataSource{
		dataSourceBinance: staticCandleSource{candles: candles},
	})
	app := &application{
		currentInstrument: defaultInstrument(), chart: newKlineState(0),
		instrumentPickerOpen: true, marketUpdates: make(chan market.Snapshot, 16),
	}
	var calls atomic.Int32
	app.instrumentSearch.provider = instrumentSearchFunc(func(ctx context.Context, query string) ([]instrument, error) {
		if query != "eth" {
			t.Errorf("debounced query = %q, want eth", query)
		}
		var err error
		if calls.Add(1) == 1 {
			err = errors.New("OKX offline")
		}
		return []instrument{eth, futures}, err
	})
	app.marketLoader = market.NewLoader(sources, app.publishMarketSnapshot)
	t.Cleanup(app.marketLoader.Close)
	t.Cleanup(app.stopInstrumentSearch)
	focusList := false
	tt := ui.NewTester(func(c *ui.Context) {
		app.drainMarketUpdates()
		app.instrumentPickerDialog(c)
		if focusList {
			app.instrumentPickerList.Focus(c)
		}
	}, 760, 520)
	for _, query := range []string{"e", "et", "eth"} {
		app.instrumentPickerQuery = query
		app.searchInstruments()
	}
	waitInstrumentUI(t, tt, func() bool { return !app.instrumentSearch.loading })
	if calls.Load() != 1 || !tt.HasText("部分市场搜索失败") || len(app.filteredInstruments()) != 2 {
		t.Fatal("debounce or partial search results were not preserved")
	}
	if err := tt.Click("现货"); err != nil {
		t.Fatal(err)
	}
	if items := app.filteredInstruments(); len(items) != 1 || items[0].MarketType != marketSpot {
		t.Fatalf("remote spot filter = %+v", items)
	}
	if err := tt.Click("合约"); err != nil {
		t.Fatal(err)
	}
	if items := app.filteredInstruments(); len(items) != 1 || items[0].MarketType != marketFutures {
		t.Fatalf("remote futures filter = %+v", items)
	}
	if err := tt.Click("重试"); err != nil {
		t.Fatal(err)
	}
	waitInstrumentUI(t, tt, func() bool { return !app.instrumentSearch.loading })
	if calls.Load() != 2 || app.instrumentSearch.err != nil || tt.HasText("部分市场搜索失败") {
		t.Fatal("retry did not clear the search failure")
	}
	if err := tt.Click("全部"); err != nil {
		t.Fatal(err)
	}
	focusList = true
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	if app.instrumentPickerCursor != 1 {
		t.Fatalf("keyboard cursor = %d, want 1", app.instrumentPickerCursor)
	}
	tt.Key(0, ui.KeyEnter)
	focusList = false
	waitInstrumentUI(t, tt, func() bool { return app.currentInstrument.Identity() == futures.Identity() })
	if app.instrumentPickerOpen || len(app.instrumentHistory) != 1 {
		t.Fatal("keyboard submission did not select and remember remote instrument")
	}
}

func TestInstrumentSearchIgnoresLateUncancellableProviderResponse(t *testing.T) {
	type request struct {
		query    string
		response chan []instrument
		returned chan struct{}
	}
	requests := make(chan request, 2)
	app := &application{instrumentPickerOpen: true}
	app.instrumentSearch.provider = instrumentSearchFunc(func(ctx context.Context, query string) ([]instrument, error) {
		req := request{query: query, response: make(chan []instrument, 1), returned: make(chan struct{})}
		requests <- req
		// Deliberately ignore cancellation to simulate a late transport.
		items := <-req.response
		close(req.returned)
		return items, nil
	})
	t.Cleanup(app.stopInstrumentSearch)
	tt := ui.NewTester(func(c *ui.Context) { app.instrumentPickerDialog(c) }, 760, 520)
	app.instrumentPickerQuery = "btc"
	app.searchInstruments()
	waitInstrumentUI(t, tt, func() bool { return len(requests) == 1 })
	old := <-requests
	t.Cleanup(func() {
		select {
		case old.response <- nil:
		default:
		}
	})
	app.instrumentPickerQuery = "eth"
	app.searchInstruments()
	waitInstrumentUI(t, tt, func() bool { return len(requests) == 1 })
	current := <-requests
	t.Cleanup(func() {
		select {
		case current.response <- nil:
		default:
		}
	})
	if old.query != "btc" || current.query != "eth" {
		t.Fatalf("request order = %q, %q", old.query, current.query)
	}
	current.response <- []instrument{historyETH()}
	waitInstrumentUI(t, tt, func() bool { return !app.instrumentSearch.loading })
	old.response <- []instrument{defaultInstrument()}
	<-old.returned
	tt.Frame()
	if len(app.instrumentSearch.items) != 1 || app.instrumentSearch.items[0].Symbol != "ETH/USDT" {
		t.Fatal("late canceled response replaced the current search")
	}
}
