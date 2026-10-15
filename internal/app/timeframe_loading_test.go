package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

type timeframeTestSource func(context.Context, candleRequest) ([]Candle, error)

func (load timeframeTestSource) LoadCandles(ctx context.Context, request candleRequest) ([]Candle, error) {
	return load(ctx, request)
}

func newTimeframeLoadingTester(t *testing.T, source timeframeTestSource) (*application, *ui.Tester) {
	t.Helper()
	item := defaultInstrument()
	registry := newMarketDataRegistry(map[dataSourceID]MarketDataSource{item.SourceID: source})
	app := &application{
		activePage: "market", currentInstrument: item, chart: newKlineState(0),
		marketUpdates: make(chan market.Snapshot, 16), dataSources: registry,
	}
	app.marketLoader = market.NewLoader(registry, app.publishMarketSnapshot)
	t.Cleanup(app.marketLoader.Close)
	tt := ui.NewTester(app.content, 900, 600)
	waitInstrumentUI(t, tt, func() bool { return len(app.candles) > 0 && !app.marketLoading })
	return app, tt
}

func TestTimeframeDropdownLoadsAndCommitsRequestedSeries(t *testing.T) {
	initial := demoCandles()
	changed := cloneCandles(initial[:120])
	changed[len(changed)-1].Close = 1234
	requests := make(chan candleRequest, 8)
	app, tt := newTimeframeLoadingTester(t, func(_ context.Context, request candleRequest) ([]Candle, error) {
		requests <- request
		if request.Interval == "5m" {
			return changed, nil
		}
		return initial, nil
	})
	app.toggleTimeframeFavorite("5m")
	app.chart.annotations = []klineAnnotation{{ID: 1, Kind: annotationText}}
	app.chart.replaySpeed = 5
	tt.Frame()
	clickReplaySetting(t, tt, app.text().Timeframes())
	clickReplaySetting(t, tt, app.text().SelectTimeframe("5m"))
	waitInstrumentUI(t, tt, func() bool { return len(app.candles) == 120 && !app.marketLoading })
	first, second := <-requests, <-requests
	if first.Interval != "1h" || second.Interval != "5m" || second.Instrument.Identity() != app.currentInstrument.Identity() {
		t.Fatalf("requests = %+v, %+v", first, second)
	}
	if app.chart.timeframe != "5m" || app.candles[119].Close != 1234 || len(app.chart.annotations) != 0 || app.pendingPreviousTimeframe != "" {
		t.Fatal("committed timeframe did not replace the chart series and clear old chart state")
	}
	if app.chart.replaySpeed != 5 || !reflect.DeepEqual(app.favoriteTimeframes(), []string{"5m", "15m", "1h", "4h", "1D"}) {
		t.Fatal("committed timeframe discarded independent application preferences")
	}
	clickReplaySetting(t, tt, app.text().Timeframes())
	clickReplaySetting(t, tt, app.text().SelectTimeframe("5m"))
	select {
	case request := <-requests:
		t.Fatalf("selecting the current interval reloaded data: %+v", request)
	default:
	}
}

func TestTimeframeFailedRapidChoicesRestoreCommittedSeriesAndCanRetry(t *testing.T) {
	initial := demoCandles()
	requests := make(chan candleRequest, 8)
	results := make(chan error, 2)
	app, tt := newTimeframeLoadingTester(t, func(ctx context.Context, request candleRequest) ([]Candle, error) {
		requests <- request
		switch request.Interval {
		case "4h":
			<-ctx.Done()
			return nil, ctx.Err()
		case "5m":
			return initial[:100], <-results
		default:
			return initial, nil
		}
	})
	<-requests // Initial committed 1h series.
	app.chart.annotations = []klineAnnotation{{ID: 1, Kind: annotationText}}
	clickReplaySetting(t, tt, "4h")
	clickReplaySetting(t, tt, app.text().Timeframes())
	clickReplaySetting(t, tt, app.text().SelectTimeframe("5m"))
	results <- errors.New("offline")
	waitInstrumentUI(t, tt, func() bool { return app.marketError != nil && !app.marketLoading })
	if app.chart.timeframe != "1h" || len(app.candles) != len(initial) || len(app.chart.annotations) != 1 || app.pendingPreviousTimeframe != "" {
		t.Fatalf("failed choice did not restore committed state: period=%s, candles=%d, annotations=%d", app.chart.timeframe, len(app.candles), len(app.chart.annotations))
	}
	if app.pendingCandleLoad == nil || app.pendingCandleLoad.Interval != "5m" {
		t.Fatal("failure lost the retry request")
	}
	results <- nil
	app.retryMarketData()
	waitInstrumentUI(t, tt, func() bool { return app.marketError == nil && !app.marketLoading && len(app.candles) == 100 })
	if app.chart.timeframe != "5m" || len(app.chart.annotations) != 0 {
		t.Fatal("retry did not commit the requested series")
	}
}
