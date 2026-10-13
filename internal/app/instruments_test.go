package app

import (
	"context"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

type staticCandleSource struct {
	candles []Candle
	items   []instrument
}

func (s staticCandleSource) LoadCandles(_ context.Context, _ candleRequest) ([]Candle, error) {
	return cloneCandles(s.candles), nil
}

func (s staticCandleSource) LoadInstruments(_ context.Context, marketType market.MarketType) ([]instrument, error) {
	return market.FilterInstruments(s.items, "", marketType), nil
}

func TestDefaultInstrumentUsesOKXSpot(t *testing.T) {
	item := defaultInstrument()
	if item.SourceID != dataSourceOKX || item.MarketType != marketSpot || item.ContractType != contractNone {
		t.Fatalf("default instrument = %+v, want OKX spot", item)
	}
}

func TestInstrumentCatalogUsesBinanceAndOKXSpotAndFuturesTags(t *testing.T) {
	if len(builtinInstrumentCatalog) != 4 {
		t.Fatalf("catalog size = %d, want 4", len(builtinInstrumentCatalog))
	}
	seen := map[string]bool{}
	for _, item := range builtinInstrumentCatalog {
		if item.Symbol != "BTC/USDT" || item.Name != "Bitcoin" {
			t.Fatalf("builtin catalog contains a non-Bitcoin instrument: %+v", item)
		}
		if (item.SourceID != dataSourceBinance && item.SourceID != dataSourceOKX) || item.ProviderSymbol == "" {
			t.Fatalf("instrument is not fully tagged: %+v", item)
		}
		if item.MarketType == marketFutures && item.ContractType != contractUSDM && item.ContractType != contractOKXSwap {
			t.Fatalf("futures instrument has contract type %+v: %+v", item.ContractType, item)
		}
		if item.MarketType == marketSpot && item.ContractType != contractNone {
			t.Fatalf("spot instrument has futures contract type: %+v", item)
		}
		if seen[item.Identity()] {
			t.Fatalf("duplicate instrument identity: %s", item.Identity())
		}
		seen[item.Identity()] = true
	}
	if len(seen) != 4 {
		t.Fatalf("unique instrument identities = %d, want 4", len(seen))
	}
}

func TestInstrumentFilteringMatchesNameSymbolAndMarketTab(t *testing.T) {
	app := &application{instrumentPickerTab: instrumentTabAll}

	app.instrumentPickerQuery = "btc"
	items := app.filteredInstruments()
	if len(items) != 4 {
		t.Fatalf("symbol filter = %+v, want BTC spot/futures for both sources", items)
	}

	app.instrumentPickerQuery = ""
	app.instrumentPickerTab = instrumentTabSpot
	items = app.filteredInstruments()
	if len(items) != 2 {
		t.Fatalf("spot tab size = %d, want 2", len(items))
	}
	for _, item := range items {
		if item.MarketType != marketSpot {
			t.Fatalf("spot tab returned %+v", item)
		}
	}

	app.instrumentPickerTab = instrumentTabFutures
	items = app.filteredInstruments()
	if len(items) != 2 {
		t.Fatalf("futures tab size = %d, want 2", len(items))
	}
	for _, item := range items {
		if item.MarketType != marketFutures || (item.ContractType != contractUSDM && item.ContractType != contractOKXSwap) {
			t.Fatalf("futures tab returned %+v", item)
		}
	}
}

func TestInstrumentPickerSearchAndSelectionUsesTaggedRows(t *testing.T) {
	current := defaultInstrument()
	candles := demoCandles()
	eth := instrument{Name: "ETH", Symbol: "ETH/USDT", ProviderSymbol: "ETHUSDT", SourceID: dataSourceBinance, MarketType: marketSpot}
	ethFutures := eth
	ethFutures.MarketType, ethFutures.ContractType = marketFutures, contractUSDM
	sources := newMarketDataRegistry(map[dataSourceID]MarketDataSource{
		dataSourceBinance: staticCandleSource{candles: candles, items: []instrument{eth, ethFutures}},
	})
	app := &application{
		activePage:        "market",
		currentInstrument: current,
		candles:           candles,
		chart:             newKlineState(len(candles)),
		marketUpdates:     make(chan market.Snapshot, 16),
		dataSources:       sources,
		marketLoadStarted: true,
	}
	app.marketLoader = market.NewLoader(sources, app.publishMarketSnapshot)
	t.Cleanup(app.marketLoader.Close)
	t.Cleanup(app.stopInstrumentSearch)
	app.chart.timeframe = "4h"
	app.chart.showVolume = false
	app.chart.annotations = []klineAnnotation{{ID: 1, Kind: annotationText}}

	tt := ui.NewTester(func(c *ui.Context) { app.content(c) }, 1000, 700)
	if err := tt.Click("BTC/USDT"); err != nil {
		t.Fatal(err)
	}
	if !app.instrumentPickerOpen {
		t.Fatal("clicking the current symbol did not open the instrument picker")
	}
	if !tt.Focused("搜索标的") {
		t.Fatal("instrument search field did not receive focus when the picker opened")
	}

	if err := tt.Click("搜索标的"); err != nil {
		t.Fatal(err)
	}
	tt.Type("eth")
	waitInstrumentUI(t, tt, func() bool { return !app.instrumentSearch.loading })
	if !tt.HasText("ETH/USDT") || tt.HasText("Bitcoin") {
		t.Fatal("search did not filter the tagged list")
	}
	if !tt.HasText("Binance · 现货") || !tt.HasText("Binance · 合约") {
		t.Fatal("search result rows did not show market tags")
	}

	if err := tt.Click("ETH ETH/USDT Binance · 现货"); err != nil {
		t.Fatal(err)
	}
	if app.instrumentPickerOpen {
		t.Fatal("selecting an instrument did not close the picker")
	}
	deadline := time.Now().Add(time.Second)
	for app.currentInstrument.Identity() != eth.Identity() && time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(time.Millisecond)
	}
	if app.currentInstrument.Identity() != eth.Identity() {
		t.Fatalf("selected instrument = %+v, want %+v", app.currentInstrument, eth)
	}
	if app.chart.timeframe != "4h" || app.chart.showVolume {
		t.Fatal("switching instruments did not preserve timeframe and volume settings")
	}
	if len(app.chart.annotations) != 0 {
		t.Fatal("switching instruments kept annotations from the previous instrument")
	}
}
