package market

import (
	"context"
	"errors"
	"testing"
)

func TestLoaderRetainsSourceAndSymbolInErrors(t *testing.T) {
	item := DefaultInstrument()
	missing := NewLoader(nil, nil)
	t.Cleanup(missing.Close)
	missing.Select(item, "1h", false)
	var sourceError *MissingSourceError
	if !errors.As(missing.Snapshot().Error, &sourceError) || sourceError.SourceID != item.SourceID {
		t.Fatalf("missing source error = %v", missing.Snapshot().Error)
	}
	loader, updates := testLoader(&fakeSource{})
	t.Cleanup(loader.Close)
	item.SourceID = DataSourceBinance
	loader.Select(item, "1h", false)
	snapshot := waitSnapshot(t, updates, func(snapshot Snapshot) bool { return !snapshot.Loading && snapshot.Error != nil })
	var candleError *NoCandlesError
	if !errors.As(snapshot.Error, &candleError) || candleError.Symbol != item.Symbol {
		t.Fatalf("empty candles error = %v", snapshot.Error)
	}
}

func TestInstrumentSearchRetainsMarketAndUnderlyingError(t *testing.T) {
	searcher, source, _ := searchFixture()
	cause := errors.New("offline")
	source.errs[MarketFutures] = cause
	_, err := searcher.Search(context.Background(), "eth")
	var searchError *InstrumentSearchError
	if !errors.Is(err, cause) || !errors.As(err, &searchError) ||
		searchError.SourceID != DataSourceBinance || searchError.MarketType != MarketFutures {
		t.Fatalf("catalog search error = %v", err)
	}
	if _, err := NewInstrumentSearcher(nil).Search(context.Background(), "eth"); !errors.Is(err, ErrNoInstrumentSource) {
		t.Fatalf("no catalog source = %v", err)
	}
}
