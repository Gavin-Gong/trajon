package market

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeSource struct {
	mu       sync.Mutex
	latest   []Candle
	older    []Candle
	err      error
	oldError error
}

func (s *fakeSource) LoadCandles(_ context.Context, request CandleRequest) ([]Candle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !request.Before.IsZero() {
		return CloneCandles(s.older), s.oldError
	}
	return CloneCandles(s.latest), s.err
}

func TestLoaderFailureKeepsPreviousSnapshot(t *testing.T) {
	old := Instrument{Symbol: "BTC/USDT", ProviderSymbol: "BTCUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot}
	newItem := old
	newItem.ProviderSymbol = "ETHUSDT"
	source := &fakeSource{latest: []Candle{{Time: time.UnixMilli(2), Close: 100}}}
	loader, updates := testLoader(source)

	loader.Select(old, "1h", false)
	first := waitSnapshot(t, updates, func(snapshot Snapshot) bool {
		return !snapshot.Loading && snapshot.Error == nil && len(snapshot.Candles) == 1
	})
	if first.Instrument.Identity() != old.Identity() {
		t.Fatalf("first instrument = %+v, want %+v", first.Instrument, old)
	}

	source.mu.Lock()
	source.latest = nil
	source.err = errors.New("request failed")
	source.mu.Unlock()
	loader.Select(newItem, "1h", false)
	failed := waitSnapshot(t, updates, func(snapshot Snapshot) bool {
		return !snapshot.Loading && snapshot.Error != nil
	})
	if failed.Instrument.Identity() != old.Identity() {
		t.Fatalf("failed instrument = %+v, want previous %+v", failed.Instrument, old)
	}
	if len(failed.Candles) != 1 || failed.Candles[0].Close != 100 {
		t.Fatalf("failed load replaced candles: %+v", failed.Candles)
	}
}

func TestLoaderPrependsOlderCandlesAndPreservesHistoryOffset(t *testing.T) {
	item := Instrument{Symbol: "BTC/USDT", ProviderSymbol: "BTCUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot}
	old := make([]Candle, 30)
	for i := range old {
		old[i] = Candle{Time: time.UnixMilli(int64(i+3) * 1000), Close: float64(i + 103)}
	}
	source := &fakeSource{latest: old, older: []Candle{
		{Time: time.UnixMilli(1000), Close: 101},
		{Time: time.UnixMilli(2000), Close: 102},
	}}
	loader, updates := testLoader(source)
	loader.Select(item, "1h", false)
	waitSnapshot(t, updates, func(snapshot Snapshot) bool {
		return !snapshot.Loading && snapshot.Error == nil && len(snapshot.Candles) == len(old)
	})

	loader.LoadOlder()
	merged := waitSnapshot(t, updates, func(snapshot Snapshot) bool { return snapshot.HistoryPrepended == 2 })
	if merged.OlderLoading || merged.OlderError != nil {
		t.Fatal("successful older load left an error/loading state")
	}
	if len(merged.Candles) != 32 || !merged.Candles[0].Time.Equal(time.UnixMilli(1000)) || !merged.Candles[2].Time.Equal(old[0].Time) {
		t.Fatalf("merged candles = %+v", merged.Candles)
	}
	if merged.Candles[0].EMA20 == 0 {
		t.Fatal("EMA20 was not recomputed across merged history")
	}
}

func TestLoaderMarksOlderHistoryExhausted(t *testing.T) {
	item := Instrument{Symbol: "BTC/USDT", ProviderSymbol: "BTCUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot}
	source := &fakeSource{latest: []Candle{{Time: time.UnixMilli(3000), Close: 103}}}
	loader, updates := testLoader(source)
	loader.Select(item, "1h", false)
	waitSnapshot(t, updates, func(snapshot Snapshot) bool {
		return !snapshot.Loading && snapshot.Error == nil && len(snapshot.Candles) == 1
	})

	loader.LoadOlder()
	exhausted := waitSnapshot(t, updates, func(snapshot Snapshot) bool { return snapshot.OlderExhausted })
	if exhausted.OlderLoading {
		t.Fatal("empty older response left loader busy")
	}
}

func testLoader(source CandleSource) (*Loader, <-chan Snapshot) {
	updates := make(chan Snapshot, 16)
	loader := NewLoader(NewRegistry(map[DataSourceID]CandleSource{DataSourceBinance: source}), func(snapshot Snapshot) {
		updates <- snapshot
	})
	return loader, updates
}

func waitSnapshot(t *testing.T, updates <-chan Snapshot, match func(Snapshot) bool) Snapshot {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case snapshot := <-updates:
			if match(snapshot) {
				return snapshot
			}
		case <-deadline:
			t.Fatal("timed out waiting for loader snapshot")
		}
	}
}
