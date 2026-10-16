package market

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type queryCatalogSource struct {
	catalogSource
	queries []string
	item    Instrument
	failure error
}

func (s *queryCatalogSource) SearchInstruments(ctx context.Context, query string) ([]Instrument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queries = append(s.queries, query)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.Contains(query, "700") {
		return []Instrument{s.item}, nil
	}
	return nil, s.failure
}

func TestQueryBasedDiscoveryKeepsCanonicalAliasesAndIndependentCryptoCatalogs(t *testing.T) {
	searcher, binance, okx := searchFixture()
	stock := &queryCatalogSource{catalogSource: catalogSource{calls: map[MarketType]int{}, errs: map[MarketType]error{}}, item: Instrument{Name: "Tencent", Symbol: "00700.HK", ProviderSymbol: "700.HK", SourceID: DataSourceLongbridge, MarketType: MarketStock}}
	searcher.registry = NewRegistry(map[DataSourceID]CandleSource{DataSourceBinance: binance, DataSourceOKX: okx, DataSourceLongbridge: stock})
	items, err := searcher.Search(context.Background(), "00700.hk")
	if err != nil || len(items) != 1 || items[0].ProviderSymbol != "700.HK" || len(stock.queries) != 1 || len(stock.calls) != 0 {
		t.Fatalf("query result=%+v, err=%v", items, err)
	}
	failure := errors.New("stock source unavailable")
	stock.failure = failure
	items, err = searcher.Search(context.Background(), "eth")
	if len(items) != 4 || !errors.Is(err, failure) {
		t.Fatalf("partial results=%+v, err=%v", items, err)
	}
	for _, source := range []*catalogSource{binance, okx} {
		if source.calls[MarketSpot] != 1 || source.calls[MarketFutures] != 1 {
			t.Fatalf("query source disturbed cached catalogs: %+v", source.calls)
		}
	}
}

func TestAccountChangeInvalidatesOnlyItsReusableCandles(t *testing.T) {
	crypto := Instrument{Symbol: "BTC/USDT", ProviderSymbol: "BTCUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot}
	stock := Instrument{Symbol: "AAPL.US", ProviderSymbol: "AAPL.US", SourceID: DataSourceLongbridge, MarketType: MarketStock, Currency: "USD"}
	stamp := time.Date(2024, 11, 4, 0, 0, 0, 0, time.UTC)
	cryptoSource := &fakeSource{latest: []Candle{{Time: stamp, Close: 100}}}
	stockSource := &fakeSource{latest: []Candle{{Time: stamp, Close: 200}}}
	updates := make(chan Snapshot, 16)
	loader := NewLoader(NewRegistry(map[DataSourceID]CandleSource{DataSourceBinance: cryptoSource, DataSourceLongbridge: stockSource}), func(s Snapshot) { updates <- s })
	t.Cleanup(loader.Close)
	selectItem := func(item Instrument) Snapshot {
		loader.Select(item, "1D", false)
		return waitSnapshot(t, updates, func(s Snapshot) bool {
			return !s.Loading && s.Pending == nil && s.Instrument.Identity() == item.Identity()
		})
	}
	selectItem(crypto)
	selectItem(stock)
	cryptoSource.mu.Lock()
	cryptoSource.latest[0].Close = 101
	cryptoSource.mu.Unlock()
	stockSource.mu.Lock()
	stockSource.latest[0].Close = 201
	stockSource.mu.Unlock()
	loader.InvalidateSourceCache(DataSourceLongbridge)
	if current := loader.Snapshot(); current.Instrument.Identity() != stock.Identity() || current.Candles[0].Close != 200 {
		t.Fatal("account change cleared displayed candles")
	}
	if snapshot := selectItem(crypto); snapshot.Candles[0].Close != 100 {
		t.Fatal("unrelated crypto cache was invalidated")
	}
	if snapshot := selectItem(stock); snapshot.Candles[0].Close != 201 {
		t.Fatal("new account reused the previous grant's cached candles")
	}
}
