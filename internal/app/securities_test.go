package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

func TestStockSelectionPersistsAndReplaysAcrossAClosedMarketGap(t *testing.T) {
	item := instrument{Name: "Apple", NameCN: "苹果", Symbol: "AAPL.US", ProviderSymbol: "AAPL.US", SourceID: market.DataSourceLongbridge, MarketType: market.MarketStock, Currency: "USD", SecurityMarket: "US"}
	stamps := []string{"2024-11-01T19:59:00Z", "2024-11-04T14:30:00Z", "2024-11-04T14:31:00Z"}
	candles := make([]Candle, len(stamps))
	for i, stamp := range stamps {
		at, _ := time.Parse(time.RFC3339, stamp)
		candles[i] = Candle{Time: at, Open: 100 + float64(i*5), High: 110 + float64(i*5), Low: 90 + float64(i*5), Close: 100 + float64(i*5), Volume: 10}
	}
	registry := newMarketDataRegistry(map[dataSourceID]MarketDataSource{market.DataSourceLongbridge: staticCandleSource{candles: candles}})
	app := &application{currentInstrument: defaultInstrument(), chart: newKlineState(0), dataSources: registry, marketUpdates: make(chan market.Snapshot, 16), replaySettings: defaultReplaySettings()}
	app.chart.timeframe = "1m"
	app.marketLoader = market.NewLoader(registry, app.publishMarketSnapshot)
	t.Cleanup(app.marketLoader.Close)
	app.selectInstrument(item)
	timeout := time.After(3 * time.Second)
	for app.currentInstrument.Identity() != item.Identity() || app.marketLoading {
		select {
		case snapshot := <-app.marketUpdates:
			app.applyMarketSnapshot(snapshot)
		case <-timeout:
			t.Fatal("stock candles did not commit")
		}
	}
	if len(app.candles) != 3 || app.chart.timeframe != "1m" || app.currentInstrument.Currency != "USD" {
		t.Fatal("stock selection changed period, currency, or synthesized gap candles")
	}
	path := filepath.Join(t.TempDir(), "instrument-history.json")
	if err := saveInstrumentHistory(path, app.instrumentHistory); err != nil {
		t.Fatal(err)
	}
	restored, err := loadInstrumentHistory(path)
	if err != nil || len(restored) != 1 || restored[0].Identity() != item.Identity() || restored[0].NameCN != "苹果" || restored[0].Currency != "USD" {
		t.Fatalf("stock history=%+v, err=%v", restored, err)
	}
	app.chart.selectReplayStart(0, len(app.candles))
	if trade, err := app.placeReplayTrade(market.ReplayBuy); err != nil || trade.Price != 100 || !trade.Time.Equal(candles[0].Time) {
		t.Fatalf("stock buy=%+v, err=%v", trade, err)
	}
	app.advanceReplayStep()
	if trade, err := app.placeReplayTrade(market.ReplaySell); err != nil || trade.Price != 105 || !trade.Time.Equal(candles[1].Time) || app.chart.replayEnd != 2 || app.chart.replayAccount.RealizedPnL != 5 {
		t.Fatalf("next-session sell=%+v, err=%v", trade, err)
	}
}

func TestStockAuthorizationDetourPreservesPickerQueryTabAndFocus(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	app.longbridge.client = &testLongbridgeClient{}
	t.Cleanup(app.closeLongbridge)
	t.Cleanup(app.stopInstrumentSearch)
	app.instrumentPickerOpen = true
	app.instrumentPickerTab = instrumentTabStocks
	app.instrumentPickerQuery = "700.HK"
	tt.Frame()
	clickReplaySetting(t, tt, "Connect data source")
	waitLongbridge(t, app, func() bool { return app.longbridge.job == longbridgeIdle })
	tt.Frame()
	if !app.settingsOpen || app.instrumentPickerOpen {
		t.Fatal("connection entry stacked two dialogs")
	}
	tt.Key(0, ui.KeyEscape)
	for range 3 {
		tt.Frame()
	}
	if app.settingsOpen || !app.instrumentPickerOpen || app.instrumentPickerTab != instrumentTabStocks || app.instrumentPickerQuery != "700.HK" || !tt.Focused("Search instruments") {
		t.Fatal("settings detour lost the picker draft or search focus")
	}
}
