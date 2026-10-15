package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"trajon/internal/market"
)

func TestAllCatalogIntervalsReachProvider(t *testing.T) {
	mappings := map[string]string{
		"1m": "1m", "3m": "3m", "5m": "5m", "15m": "15m", "30m": "30m",
		"1h": "1h", "2h": "2h", "4h": "4h",
		"1D": "1d", "1W": "1w", "1M": "1M",
	}
	for _, period := range market.CandleIntervals() {
		want, exists := mappings[period]
		if !exists {
			t.Fatalf("missing provider mapping for %q", period)
		}
		t.Run(period, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("interval"); got != want {
					t.Errorf("provider interval = %q, want %q", got, want)
				}
				_, _ = w.Write([]byte(`[[1704067200000,"100","101","99","100.5","10"]]`))
			}))
			defer server.Close()
			source := New(server.Client())
			source.spotBaseURL, source.futuresBaseURL = server.URL, server.URL
			item := market.Instrument{SourceID: market.DataSourceBinance, ProviderSymbol: "BTCUSDT"}
			for _, kind := range []market.MarketType{market.MarketSpot, market.MarketFutures} {
				item.MarketType, item.ContractType = kind, market.ContractNone
				if kind == market.MarketFutures {
					item.ContractType = market.ContractUSDM
				}
				candles, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: item, Interval: period, Limit: 1})
				if err != nil || len(candles) != 1 {
					t.Fatalf("%s load = %v, %v", kind, candles, err)
				}
			}
		})
	}
	for _, unsupported := range []string{"", "3h", "1d", "1mo", "1s", "6h", "12h", "3D"} {
		if _, ok := interval(unsupported); ok {
			t.Errorf("unsupported interval %q was accepted", unsupported)
		}
	}
}
