package okx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"trajon/internal/market"
)

func TestLoadInstrumentsTagsLiveSpotAndLinearSwaps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/public/instruments" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		switch r.URL.Query().Get("instType") {
		case "SPOT":
			fmt.Fprint(w, `{"code":"0","data":[
				{"instId":"ETH-USDT","instType":"SPOT","state":"live","baseCcy":"ETH","quoteCcy":"USDT"},
				{"instId":"ETH-BTC","instType":"SPOT","state":"live","baseCcy":"ETH","quoteCcy":"BTC"},
				{"instId":"OLD-USDT","instType":"SPOT","state":"suspend","baseCcy":"OLD","quoteCcy":"USDT"},
				{"instId":"BAD-USDT","instType":"SPOT","state":"live","baseCcy":"BAD"}
			]}`)
		case "SWAP":
			// Swap baseCcy/quoteCcy are empty; use the instrument family.
			fmt.Fprint(w, `{"code":"0","data":[
				{"instId":"ETH-USDT-SWAP","instType":"SWAP","state":"live","instFamily":"ETH-USDT","ctType":"linear"},
				{"instId":"ETH-USDC-SWAP","instType":"SWAP","state":"live","instFamily":"ETH-USDC","ctType":"linear"},
				{"instId":"ETH-USD-SWAP","instType":"SWAP","state":"live","instFamily":"ETH-USD","ctType":"inverse"},
				{"instId":"OLD-USDT-SWAP","instType":"SWAP","state":"preopen","instFamily":"OLD-USDT","ctType":"linear"},
				{"instId":"ETH-USDT-260925","instType":"FUTURES","state":"live","instFamily":"ETH-USDT","ctType":"linear"}
			]}`)
		default:
			t.Errorf("unexpected query %s", r.URL.RawQuery)
		}
	}))
	defer server.Close()
	source := New(server.Client())
	source.baseURL = server.URL
	seen := map[string]bool{}
	for _, marketType := range []market.MarketType{market.MarketSpot, market.MarketFutures} {
		items, err := source.LoadInstruments(context.Background(), marketType)
		if err != nil || len(items) != 2 {
			t.Fatalf("%s catalog = %+v, %v", marketType, items, err)
		}
		for _, item := range items {
			if item.Name != "ETH" || item.SourceID != market.DataSourceOKX || item.MarketType != marketType || seen[item.Identity()] {
				t.Fatalf("unexpected instrument %+v", item)
			}
			if marketType == market.MarketFutures && item.ContractType != market.ContractOKXSwap {
				t.Fatalf("incorrect contract type %+v", item)
			}
			seen[item.Identity()] = true
		}
		if items[0].Symbol != "ETH/USDT" {
			t.Fatalf("symbol conversion = %+v", items[0])
		}
	}
}

func TestLoadInstrumentsRejectsHTTPErrorsAPIErrorsAndBadJSON(t *testing.T) {
	for _, body := range []string{"unavailable", "not-json", `{"code":"50000","msg":"error","data":[]}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if body == "unavailable" {
					w.WriteHeader(http.StatusServiceUnavailable)
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			source := New(server.Client())
			source.baseURL = server.URL
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
