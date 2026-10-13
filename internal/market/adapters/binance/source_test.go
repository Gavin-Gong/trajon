package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"trajon/internal/market"
)

func TestSourceLoadsSpotAndFuturesKlines(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path+"?"+r.URL.RawQuery)
		if got := r.URL.Query().Get("symbol"); got != "BTCUSDT" {
			t.Errorf("symbol = %q, want BTCUSDT", got)
		}
		if got := r.URL.Query().Get("interval"); got != "1d" {
			t.Errorf("interval = %q, want 1d", got)
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Errorf("limit = %q, want 2", got)
		}
		_, _ = w.Write([]byte(`[
			[1704067200000,"100.0","105.0","99.0","102.0","1200.5",1704070799999,"0",10,"0","0","0"],
			[1704070800000,"102.0","108.0","101.0","107.0","1400.5",1704074399999,"0",10,"0","0","0"]
		]`))
	}))
	defer server.Close()

	source := New(server.Client())
	source.spotBaseURL = server.URL
	source.futuresBaseURL = server.URL
	spot := market.Instrument{ProviderSymbol: "BTCUSDT", SourceID: market.DataSourceBinance, MarketType: market.MarketSpot}
	candles, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: spot, Interval: "1D", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 2 {
		t.Fatalf("got %d candles, want 2", len(candles))
	}
	if got, want := candles[0].Time, time.UnixMilli(1704067200000).UTC(); !got.Equal(want) {
		t.Fatalf("first candle time = %s, want %s", got, want)
	}
	if candles[0].Close != 102 || candles[1].Volume != 1400.5 {
		t.Fatalf("decoded candles = %+v", candles)
	}
	if candles[0].EMA20 != 102 || candles[1].EMA20 != 102.47619047619048 {
		t.Fatalf("EMA20 = %v and %v, want 102 and 102.47619047619048", candles[0].EMA20, candles[1].EMA20)
	}

	futures := spot
	futures.MarketType = market.MarketFutures
	futures.ContractType = market.ContractUSDM
	if _, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: futures, Interval: "1D", Limit: 2}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 {
		t.Fatalf("got %d requests, want 2", len(requests))
	}
	if got := requests[0]; !containsPathAndQuery(got, "/api/v3/klines") {
		t.Fatalf("spot request = %q, missing spot path", got)
	}
	if got := requests[1]; !containsPathAndQuery(got, "/fapi/v1/klines") {
		t.Fatalf("futures request = %q, missing futures path", got)
	}
}

func TestSourceUsesEndTimeForOlderCandles(t *testing.T) {
	wantBefore := time.UnixMilli(1704067200000).UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantEnd := strconv.FormatInt(wantBefore.Add(-time.Millisecond).UnixMilli(), 10)
		if got := r.URL.Query().Get("endTime"); got != wantEnd {
			t.Errorf("endTime = %q, want %q", got, wantEnd)
		}
		_, _ = w.Write([]byte(`[[1704060000000,"100","101","99","100.5","10",0,"0",0,"0","0","0"]]`))
	}))
	defer server.Close()

	source := New(server.Client())
	source.spotBaseURL = server.URL
	item := market.Instrument{ProviderSymbol: "BTCUSDT", SourceID: market.DataSourceBinance, MarketType: market.MarketSpot}
	if _, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: item, Interval: "1h", Limit: 1, Before: wantBefore}); err != nil {
		t.Fatal(err)
	}
}

func TestSourceRejectsInvalidRequests(t *testing.T) {
	source := New(http.DefaultClient)
	item := market.Instrument{ProviderSymbol: "BTCUSDT", SourceID: market.DataSourceBinance, MarketType: market.MarketSpot}
	for name, request := range map[string]market.CandleRequest{
		"zero limit":   {Instrument: item, Interval: "1h", Limit: 0},
		"large limit":  {Instrument: item, Interval: "1h", Limit: 1001},
		"bad interval": {Instrument: item, Interval: "3h", Limit: 10},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := source.LoadCandles(context.Background(), request); err == nil {
				t.Fatal("invalid request unexpectedly succeeded")
			}
		})
	}
}

func containsPathAndQuery(raw, path string) bool {
	parsed, err := url.Parse("http://example.test" + raw)
	return err == nil && parsed.Path == path && parsed.Query().Get("limit") == strconv.Itoa(2)
}
