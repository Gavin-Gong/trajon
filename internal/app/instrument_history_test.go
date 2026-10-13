package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"trajon/internal/market"
)

func historyETH() instrument {
	return instrument{Name: "ETH", Symbol: "ETH/USDT", ProviderSymbol: "ETHUSDT", SourceID: dataSourceBinance, MarketType: marketSpot}
}

func TestInstrumentHistoryDeduplicatesFullIdentityAndMovesLatestFirst(t *testing.T) {
	eth := historyETH()
	futures := eth
	futures.MarketType, futures.ContractType = marketFutures, contractUSDM
	okx := eth
	okx.SourceID, okx.ProviderSymbol = dataSourceOKX, "ETH-USDT"
	app := &application{}
	for _, item := range []instrument{eth, futures, okx, eth} {
		app.rememberInstrument(item)
	}
	if !reflect.DeepEqual(app.instrumentHistory, []instrument{eth, okx, futures}) {
		t.Fatalf("history = %+v", app.instrumentHistory)
	}
	for i := range instrumentHistoryLimit + 3 {
		item := eth
		item.Name, item.Symbol, item.ProviderSymbol = fmt.Sprintf("COIN%d", i), fmt.Sprintf("COIN%d/USDT", i), fmt.Sprintf("COIN%dUSDT", i)
		app.rememberInstrument(item)
	}
	if len(app.instrumentHistory) != instrumentHistoryLimit || app.instrumentHistory[0].Name != "COIN14" {
		t.Fatalf("bounded recent history = %+v", app.instrumentHistory)
	}
}

func TestInstrumentHistoryPersistsLatestSnapshotAndReloadsAfterClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trajon", "instrument-history.json")
	store := newInstrumentHistoryStore(path)
	t.Cleanup(func() { store.Close() })
	app := &application{instrumentHistoryStore: store}
	for i := range 30 {
		item := historyETH()
		item.ProviderSymbol, item.Symbol = fmt.Sprintf("COIN%dUSDT", i), fmt.Sprintf("COIN%d/USDT", i)
		app.rememberInstrument(item)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	items, err := loadInstrumentHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(items, app.instrumentHistory) {
		t.Fatalf("reloaded history = %+v, want %+v", items, app.instrumentHistory)
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".instrument-history-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary history files = %v, %v", files, err)
	}
	// Saving a captured slice must not observe later owner mutations.
	path2 := filepath.Join(t.TempDir(), "history.json")
	store2 := newInstrumentHistoryStore(path2)
	items = []instrument{historyETH()}
	store2.Save(items)
	items[0].Name = "mutated"
	if err := store2.Close(); err != nil {
		t.Fatal(err)
	}
	reloaded, _ := loadInstrumentHistory(path2)
	if len(reloaded) != 1 || reloaded[0].Name != "ETH" {
		t.Fatalf("store observed shared application slice: %+v", reloaded)
	}
}

func TestInstrumentHistoryHandlesMissingInvalidAndUnwritableFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	if items, err := loadInstrumentHistory(path); err != nil || len(items) != 0 {
		t.Fatalf("missing history = %+v, %v", items, err)
	}
	if err := os.WriteFile(path, []byte("invalid JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadInstrumentHistory(path); err == nil {
		t.Fatal("invalid history JSON should report an error")
	}
	if err := os.WriteFile(path, []byte(`[{},{"Name":"bad","Symbol":"ETH/USDT","SourceID":"binance","MarketType":"spot"}]`), 0600); err != nil {
		t.Fatal(err)
	}
	if items, err := loadInstrumentHistory(path); err != nil || len(items) != 0 {
		t.Fatalf("incomplete history entries = %+v, %v", items, err)
	}
	// A regular file cannot be used as the parent directory.
	store := newInstrumentHistoryStore(filepath.Join(path, "child.json"))
	store.Save([]instrument{historyETH()})
	if err := store.Close(); err == nil {
		t.Fatal("unwritable history should report an error")
	}
}

func TestInstrumentHistoryOnlyRecordsSuccessfulSelections(t *testing.T) {
	current, eth := defaultInstrument(), historyETH()
	app := &application{currentInstrument: current, chart: newKlineState(0), pendingHistoryIdentity: eth.Identity()}
	app.applyMarketSnapshot(market.Snapshot{Instrument: current, Error: errors.New("offline")})
	if len(app.instrumentHistory) != 0 {
		t.Fatal("failed selection was recorded in history")
	}
	app.applyMarketSnapshot(market.Snapshot{Instrument: eth, Interval: "1h", Candles: demoCandles()})
	if !reflect.DeepEqual(app.instrumentHistory, []instrument{eth}) || app.pendingHistoryIdentity != "" {
		t.Fatalf("successful selection history = %+v", app.instrumentHistory)
	}
	// Timeframe/history loads do not reorder recent instrument choices.
	app.rememberInstrument(current)
	app.applyMarketSnapshot(market.Snapshot{Instrument: eth, Interval: "4h", Candles: demoCandles()})
	if app.instrumentHistory[0].Identity() != current.Identity() {
		t.Fatal("unrelated candle load reordered history")
	}
	// Explicitly choosing the current instrument still promotes it.
	app.pendingHistoryIdentity = eth.Identity()
	app.applyMarketSnapshot(market.Snapshot{Instrument: eth, Interval: "4h", Candles: demoCandles()})
	if app.instrumentHistory[0].Identity() != eth.Identity() {
		t.Fatal("choosing current instrument did not promote it")
	}
}

func TestInstrumentPickerHistoryChoicesSelectWithoutSearchAndPreserveFocus(t *testing.T) {
	candles, eth := demoCandles(), historyETH()
	path := filepath.Join(t.TempDir(), "history.json")
	if err := saveInstrumentHistory(path, []instrument{eth}); err != nil {
		t.Fatal(err)
	}
	history, err := loadInstrumentHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	sources := newMarketDataRegistry(map[dataSourceID]MarketDataSource{
		dataSourceBinance: staticCandleSource{candles: candles},
	})
	app := &application{
		activePage: "market", currentInstrument: defaultInstrument(), candles: candles,
		chart: newKlineState(len(candles)), marketUpdates: make(chan market.Snapshot, 16),
		dataSources: sources, marketLoadStarted: true, instrumentHistory: history,
	}
	app.marketLoader = market.NewLoader(sources, app.publishMarketSnapshot)
	t.Cleanup(app.marketLoader.Close)
	t.Cleanup(app.stopInstrumentSearch)
	tt := ui.NewTester(func(c *ui.Context) { app.content(c) }, 760, 520)
	if err := tt.Click("BTC/USDT"); err != nil {
		t.Fatal(err)
	}
	searchBounds, searchFound := tt.Find("搜索标的")
	historyBounds, historyFound := tt.Find("历史标的 ETH/USDT · Binance · 现货")
	if !searchFound || !historyFound || historyBounds.Y <= searchBounds.Y {
		t.Fatal("history choice was not placed below search")
	}
	if err := tt.Click("搜索标的"); err != nil {
		t.Fatal(err)
	}
	tt.Type("unknown")
	if err := tt.Click("合约"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("历史标的 ETH/USDT · Binance · 现货"); err != nil {
		t.Fatal(err)
	}
	waitInstrumentUI(t, tt, func() bool { return app.currentInstrument.Identity() == eth.Identity() })
	if app.instrumentPickerOpen || app.instrumentSearch.cancel != nil || app.instrumentPickerQuery != "" || app.instrumentPickerTab != instrumentTabAll {
		t.Fatal("history selection did not close and reset picker")
	}
	if err := tt.Click("ETH/USDT"); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused("搜索标的") || !tt.HasText("历史标的") || len(app.filteredInstruments()) != 4 {
		t.Fatal("reopening did not restore history and defaults")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Frame()
	if app.instrumentPickerOpen {
		t.Fatal("Escape did not close picker with history")
	}
}
