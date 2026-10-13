package market

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

type catalogSource struct {
	mu    sync.Mutex
	items []Instrument
	calls map[MarketType]int
	errs  map[MarketType]error
}

func (s *catalogSource) LoadCandles(context.Context, CandleRequest) ([]Candle, error) {
	return nil, nil
}

func (s *catalogSource) LoadInstruments(ctx context.Context, marketType MarketType) ([]Instrument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[marketType]++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return FilterInstruments(s.items, "", marketType), s.errs[marketType]
}

func searchFixture() (*InstrumentSearcher, *catalogSource, *catalogSource) {
	binance := &catalogSource{calls: map[MarketType]int{}, errs: map[MarketType]error{}, items: []Instrument{
		{Name: "ETH", Symbol: "ETH/USDT", ProviderSymbol: "ETHUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot},
		{Name: "ETH", Symbol: "ETH/USDT", ProviderSymbol: "ETHUSDT", SourceID: DataSourceBinance, MarketType: MarketFutures, ContractType: ContractUSDM},
	}}
	okx := &catalogSource{calls: map[MarketType]int{}, errs: map[MarketType]error{}, items: []Instrument{
		{Name: "ETH", Symbol: "ETH/USDT", ProviderSymbol: "ETH-USDT", SourceID: DataSourceOKX, MarketType: MarketSpot},
		{Name: "ETH", Symbol: "ETH/USDT", ProviderSymbol: "ETH-USDT-SWAP", SourceID: DataSourceOKX, MarketType: MarketFutures, ContractType: ContractOKXSwap},
	}}
	searcher := NewInstrumentSearcher(NewRegistry(map[DataSourceID]CandleSource{DataSourceBinance: binance, DataSourceOKX: okx}))
	return searcher, binance, okx
}

func TestInstrumentCatalogOnlyIncludesBitcoinAndReturnsCopies(t *testing.T) {
	items := InstrumentCatalog()
	if len(items) != 4 {
		t.Fatalf("catalog size = %d, want 4", len(items))
	}
	seen := map[string]bool{}
	for _, item := range items {
		if item.Symbol != "BTC/USDT" || item.Name != "Bitcoin" || seen[item.Identity()] {
			t.Fatalf("unexpected builtin %+v", item)
		}
		seen[item.Identity()] = true
	}
	if DefaultInstrument().SourceID != DataSourceOKX || DefaultInstrument().MarketType != MarketSpot {
		t.Fatalf("default instrument = %+v", DefaultInstrument())
	}
	items[0].Name = "changed"
	if InstrumentCatalog()[0].Name != "Bitcoin" {
		t.Fatal("catalog returned shared storage")
	}
}

func TestInstrumentSearchMatchesBothSourcesAndCommonSymbolFormats(t *testing.T) {
	searcher, binance, okx := searchFixture()
	var want []Instrument
	for _, query := range []string{"ETH", " eth/usdt ", "ethusdt", "eth-usdt", "ETH_USDT"} {
		items, err := searcher.Search(context.Background(), query)
		if err != nil || len(items) != 4 {
			t.Fatalf("search %q = %+v, %v", query, items, err)
		}
		if want == nil {
			want = items
		} else if !reflect.DeepEqual(items, want) {
			t.Fatalf("unstable results for %q: %+v", query, items)
		}
	}
	want[0].Name = "changed"
	items, _ := searcher.Search(context.Background(), "eth")
	if items[0].Name == "changed" {
		t.Fatal("search returned shared cached storage")
	}
	for _, source := range []*catalogSource{binance, okx} {
		for _, marketType := range []MarketType{MarketSpot, MarketFutures} {
			if source.calls[marketType] != 1 {
				t.Fatalf("catalog downloads = %+v, want one per market", source.calls)
			}
		}
	}
	items, err := searcher.Search(context.Background(), "doesnotexist")
	if err != nil || len(items) != 0 {
		t.Fatalf("unknown instrument search = %+v, %v", items, err)
	}
}

func TestInstrumentSearchKeepsPartialResultsAndRetriesOnlyFailedCatalog(t *testing.T) {
	searcher, binance, _ := searchFixture()
	binance.errs[MarketFutures] = errors.New("offline")
	binance.items = binance.items[:1]
	items, err := searcher.Search(context.Background(), "eth")
	if len(items) != 3 || err == nil {
		t.Fatalf("partial search = %+v, %v", items, err)
	}
	binance.errs[MarketFutures] = nil
	if _, err := searcher.Search(context.Background(), "eth"); err != nil {
		t.Fatal(err)
	}
	if binance.calls[MarketSpot] != 1 || binance.calls[MarketFutures] != 2 {
		t.Fatalf("retry downloads = %+v", binance.calls)
	}
}

func TestInstrumentSearchRefreshesExpiredCatalogsAndHonorsCancellation(t *testing.T) {
	searcher, binance, _ := searchFixture()
	searcher.Search(context.Background(), "eth")
	searcher.mu.Lock()
	for key, cached := range searcher.cache {
		cached.at = time.Now().Add(-instrumentCatalogTTL - time.Second)
		searcher.cache[key] = cached
	}
	searcher.mu.Unlock()
	if _, err := searcher.Search(context.Background(), "eth"); err != nil {
		t.Fatal(err)
	}
	if binance.calls[MarketSpot] != 2 || binance.calls[MarketFutures] != 2 {
		t.Fatalf("expired downloads = %+v", binance.calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := searcher.Search(ctx, "eth"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled cached search error = %v", err)
	}
	if items, err := searcher.Search(context.Background(), "  "); err != nil || len(items) != 4 {
		t.Fatalf("empty search = %+v, %v", items, err)
	}
	if _, err := NewInstrumentSearcher(nil).Search(context.Background(), "eth"); err == nil {
		t.Fatal("search without sources should fail")
	}
}
