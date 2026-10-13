package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"trajon/internal/market"
)

func TestLoadInstrumentsTagsTradableSpotAndPerpetuals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/exchangeInfo":
			fmt.Fprint(w, `{"symbols":[
				{"symbol":"ETHUSDT","status":"TRADING","baseAsset":"ETH","quoteAsset":"USDT","isSpotTradingAllowed":true},
				{"symbol":"ETHBTC","status":"TRADING","baseAsset":"ETH","quoteAsset":"BTC","isSpotTradingAllowed":true},
				{"symbol":"OLDUSDT","status":"BREAK","baseAsset":"OLD","quoteAsset":"USDT","isSpotTradingAllowed":true},
				{"symbol":"NOUSDT","status":"TRADING","baseAsset":"NO","quoteAsset":"USDT","isSpotTradingAllowed":false},
				{"symbol":"","status":"TRADING","baseAsset":"BAD","quoteAsset":"USDT","isSpotTradingAllowed":true}
			]}`)
		case "/fapi/v1/exchangeInfo":
			fmt.Fprint(w, `{"symbols":[
				{"symbol":"ETHUSDT","status":"TRADING","baseAsset":"ETH","quoteAsset":"USDT","contractType":"PERPETUAL"},
				{"symbol":"ETHUSDC","status":"TRADING","baseAsset":"ETH","quoteAsset":"USDC","contractType":"PERPETUAL"},
				{"symbol":"ETHUSDT_260925","status":"TRADING","baseAsset":"ETH","quoteAsset":"USDT","contractType":"CURRENT_QUARTER"},
				{"symbol":"OLDUSDT","status":"PENDING_TRADING","baseAsset":"OLD","quoteAsset":"USDT","contractType":"PERPETUAL"}
			]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	source := New(server.Client())
	source.spotBaseURL, source.futuresBaseURL = server.URL, server.URL
	seen := map[string]bool{}
	for _, marketType := range []market.MarketType{market.MarketSpot, market.MarketFutures} {
		items, err := source.LoadInstruments(context.Background(), marketType)
		if err != nil || len(items) != 2 {
			t.Fatalf("%s catalog = %+v, %v", marketType, items, err)
		}
		for _, item := range items {
			if item.Name != "ETH" || item.SourceID != market.DataSourceBinance || item.MarketType != marketType || seen[item.Identity()] {
				t.Fatalf("unexpected instrument %+v", item)
			}
			if marketType == market.MarketFutures && item.ContractType != market.ContractUSDM {
				t.Fatalf("incorrect contract type %+v", item)
			}
			seen[item.Identity()] = true
		}
		if items[0].ProviderSymbol != "ETHUSDT" || items[0].Symbol != "ETH/USDT" {
			t.Fatalf("symbol conversion = %+v", items[0])
		}
	}
}

func TestLoadInstrumentsRejectsErrorsAndCancellation(t *testing.T) {
	for _, body := range []string{"unavailable", "not-json"} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if body == "unavailable" {
					w.WriteHeader(http.StatusServiceUnavailable)
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			source := New(server.Client())
			source.spotBaseURL = server.URL
			if _, err := source.LoadInstruments(context.Background(), market.MarketSpot); err == nil {
				t.Fatal("expected a catalog error")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := source.LoadInstruments(ctx, market.MarketSpot); err == nil {
				t.Fatal("expected canceled request to fail")
			}
			if _, err := source.LoadInstruments(context.Background(), "unsupported"); err == nil {
				t.Fatal("expected unsupported market type to fail")
			}
		})
	}
}
