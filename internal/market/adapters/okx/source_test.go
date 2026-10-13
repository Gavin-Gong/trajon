package okx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"trajon/internal/market"
)

func TestSourcePaginatesAndNormalizesCandles(t *testing.T) {
	var afterValues []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/market/candles" {
			t.Errorf("path = %q, want /api/v5/market/candles", r.URL.Path)
		}
		if got := r.URL.Query().Get("instId"); got != "BTC-USDT" {
			t.Errorf("instId = %q, want BTC-USDT", got)
		}
		if got := r.URL.Query().Get("bar"); got != "1H" {
			t.Errorf("bar = %q, want 1H", got)
		}
		if got := r.URL.Query().Get("limit"); got != "4" && got != "2" {
			t.Errorf("limit = %q, want a page limit", got)
		}
		afterValues = append(afterValues, r.URL.Query().Get("after"))
		if r.URL.Query().Get("after") == "" {
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[
				["4000","104","105","103","104.5","14","0","0","1"],
				["3000","103","104","102","103.5","13","0","0","1"]
			]}`))
			return
		}
		if r.URL.Query().Get("after") != "3000" {
			t.Errorf("after = %q, want 3000", r.URL.Query().Get("after"))
		}
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[
			["2000","102","103","101","102.5","12","0","0","1"],
			["1000","101","102","100","101.5","11","0","0","1"]
		]}`))
	}))
	defer server.Close()

	source := New(server.Client())
	source.baseURL = server.URL
	item := market.Instrument{ProviderSymbol: "BTC-USDT", SourceID: market.DataSourceOKX, MarketType: market.MarketSpot}
	candles, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: item, Interval: "1h", Limit: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 4 {
		t.Fatalf("got %d candles, want 4", len(candles))
	}
	for i, candle := range candles {
		want := time.UnixMilli(int64((i + 1) * 1000)).UTC()
		if !candle.Time.Equal(want) {
			t.Fatalf("candle %d time = %s, want %s", i, candle.Time, want)
		}
	}
	if candles[0].Open != 101 || candles[3].Close != 104.5 || candles[0].Volume != 11 {
		t.Fatalf("decoded OKX candles = %+v", candles)
	}
	if len(afterValues) != 2 || afterValues[0] != "" || afterValues[1] != "3000" {
		t.Fatalf("pagination cursors = %v, want [empty 3000]", afterValues)
	}
}

func TestSourceUsesBeforeCursorForOlderCandles(t *testing.T) {
	before := time.UnixMilli(3000).UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("after"); got != "3000" {
			t.Errorf("after = %q, want 3000", got)
		}
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[["2000","102","103","101","102.5","12","0","0","1"]]}`))
	}))
	defer server.Close()

	source := New(server.Client())
	source.baseURL = server.URL
	item := market.Instrument{ProviderSymbol: "BTC-USDT", SourceID: market.DataSourceOKX, MarketType: market.MarketSpot}
	candles, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: item, Interval: "1h", Limit: 1, Before: before})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 || !candles[0].Time.Equal(time.UnixMilli(2000).UTC()) {
		t.Fatalf("older OKX candles = %+v", candles)
	}
}

func TestSourceUsesSwapInstrumentID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("instId"); got != "ETH-USDT-SWAP" {
			t.Errorf("instId = %q, want ETH-USDT-SWAP", got)
		}
		if got := r.URL.Query().Get("bar"); got != "1Dutc" {
			t.Errorf("bar = %q, want 1Dutc", got)
		}
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[["1704067200000","100","101","99","100.5","20","0","0","1"]]}`))
	}))
	defer server.Close()

	source := New(server.Client())
	source.baseURL = server.URL
	item := market.Instrument{ProviderSymbol: "ETH-USDT-SWAP", SourceID: market.DataSourceOKX, MarketType: market.MarketFutures, ContractType: market.ContractOKXSwap}
	candles, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: item, Interval: "1D", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 || candles[0].Close != 100.5 {
		t.Fatalf("swap candles = %+v", candles)
	}
}

func TestSourceAllowsBaseURLOverride(t *testing.T) {
	t.Setenv("OKX_BASE_URL", "https://proxy.example.test/")
	source := New(http.DefaultClient)
	if source.baseURL != "https://proxy.example.test" {
		t.Fatalf("base URL = %q, want trimmed override", source.baseURL)
	}
}

func TestSourceRejectsUnsupportedContract(t *testing.T) {
	source := New(http.DefaultClient)
	item := market.Instrument{ProviderSymbol: "BTC-USDT", SourceID: market.DataSourceOKX, MarketType: market.MarketFutures, ContractType: market.ContractUSDM}
	_, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: item, Interval: "1h", Limit: 1})
	if err == nil {
		t.Fatal("unsupported OKX contract unexpectedly succeeded")
	}
}
