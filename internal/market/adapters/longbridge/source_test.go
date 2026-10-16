package longbridge

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	auth "trajon/internal/longbridge"
	"trajon/internal/market"
)

type quoteAPI func(context.Context, string, any, any) error

func (f quoteAPI) Request(ctx context.Context, path string, input, output any) error {
	return f(ctx, path, input, output)
}

func candleWire(stamp time.Time, session any) map[string]any {
	return map[string]any{"timestamp": stamp.Unix(), "open": "100", "high": "110", "low": "90", "close": "105", "volume": "25", "trade_session": session}
}
func response(output any, data any) error {
	encoded, _ := json.Marshal(data)
	return json.Unmarshal(encoded, output)
}
func stock(symbol string) market.Instrument {
	item, _ := instrument(securityInfo{Symbol: symbol})
	return item
}

func TestHistoryPaginationIsExclusiveSortedAndRegularSessionOnly(t *testing.T) {
	location, _ := time.LoadLocation("America/New_York")
	before := time.Date(2024, 11, 4, 11, 0, 0, 0, location)
	calls := 0
	source := New(quoteAPI(func(_ context.Context, path string, input, output any) error {
		calls++
		request := input.(historyRequest)
		if path != "/quote/history-candlesticks" || request.Symbol != "AAPL.US" || request.Period != 1 || request.AdjustType != 1 || request.TradeSession != 0 || request.QueryType != 1 || request.Offset.Direction != 0 || request.Offset.Count != 1000 {
			t.Fatalf("incorrect history request: %+v", request)
		}
		rows := []map[string]any{}
		if calls == 1 {
			if request.Offset.Date != "20241104" || request.Offset.Minute != "1059" {
				t.Fatalf("initial cursor: %+v", request.Offset)
			}
			rows = []map[string]any{candleWire(before, "NORMAL_TRADE"), candleWire(before.Add(-time.Minute), 0), candleWire(before.Add(-time.Minute), "0"), candleWire(before.Add(-2*time.Minute), 1)}
		} else if calls == 2 {
			if request.Offset.Minute != "1057" {
				t.Fatalf("next cursor: %+v", request.Offset)
			}
			rows = []map[string]any{candleWire(before.Add(-2*time.Minute), 1), candleWire(before.Add(-4*time.Minute), 0), candleWire(before.Add(-3*time.Minute), 0)}
		} else {
			t.Fatal("history pagination did not stop")
		}
		return response(output, map[string]any{"symbol": "AAPL.US", "candlesticks": rows})
	}))
	candles, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: stock("AAPL.US"), Interval: "1m", Limit: 3, Before: before.UTC()})
	if err != nil || len(candles) != 3 || calls != 2 {
		t.Fatalf("bars=%d, calls=%d, err=%v", len(candles), calls, err)
	}
	want := []time.Time{before.Add(-4 * time.Minute).UTC(), before.Add(-3 * time.Minute).UTC(), before.Add(-time.Minute).UTC()}
	for i, candle := range candles {
		if !candle.Time.Equal(want[i]) || candle.Time.Location() != time.UTC || candle.EMA20 != 105 {
			t.Fatalf("bar %d: %+v", i, candle)
		}
	}
}

func TestHistoryOffsetsUseExchangeTimeAcrossDSTAndMidnight(t *testing.T) {
	for _, test := range []struct{ symbol, utc, date, minute string }{
		{"AAPL.US", "2024-03-11T13:30:00Z", "20240311", "0929"},
		{"AAPL.US", "2024-11-04T14:30:00Z", "20241104", "0929"},
		{"AAPL.US", "2024-11-04T05:00:00Z", "20241103", "2359"},
		{"700.HK", "2024-11-04T01:30:00Z", "20241104", "0929"},
		{"600519.SH", "2024-11-04T01:30:00Z", "20241104", "0929"},
	} {
		t.Run(test.symbol+test.utc, func(t *testing.T) {
			source := New(quoteAPI(func(_ context.Context, _ string, input, output any) error {
				cursor := input.(historyRequest).Offset
				if cursor.Date != test.date || cursor.Minute != test.minute {
					t.Fatalf("cursor=%+v", cursor)
				}
				return response(output, map[string]any{"candlesticks": []any{}})
			}))
			before, _ := time.Parse(time.RFC3339, test.utc)
			if candles, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: stock(test.symbol), Interval: "1D", Limit: 10, Before: before}); err != nil || len(candles) != 0 {
				t.Fatalf("bars=%d, err=%v", len(candles), err)
			}
		})
	}
}

func TestNativePeriodsAndUSOvernightFallback(t *testing.T) {
	want := map[string]int{"1m": 1, "3m": 3, "5m": 5, "15m": 15, "30m": 30, "1h": 60, "2h": 120, "4h": 240, "1D": 1000, "1W": 2000, "1M": 3000}
	for _, value := range market.CandleIntervals() {
		if got, ok := interval(value); !ok || got != want[value] {
			t.Errorf("%s: %d, %v", value, got, ok)
		}
	}
	location, _ := time.LoadLocation("America/New_York")
	regular := time.Date(2024, 11, 4, 15, 59, 0, 0, location)
	source := New(quoteAPI(func(_ context.Context, _ string, _ any, output any) error {
		return response(output, map[string]any{"candlesticks": []any{candleWire(regular, 0), candleWire(regular.Add(5*time.Hour), 0)}})
	}))
	candles, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: stock("AAPL.US"), Interval: "1m", Limit: 1})
	if err != nil || len(candles) != 1 || !candles[0].Time.Equal(regular) {
		t.Fatalf("overnight fallback bars=%+v, err=%v", candles, err)
	}
}

func TestInvalidCandlesAndStalledHistoryFailInsteadOfExhausting(t *testing.T) {
	stamp := time.Date(2024, 11, 4, 10, 0, 0, 0, time.UTC)
	for _, corrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "stalled", true: "invalid-price"}[corrupt], func(t *testing.T) {
			source := New(quoteAPI(func(_ context.Context, _ string, _ any, output any) error {
				row := candleWire(stamp, 0)
				if corrupt {
					row["low"] = "106"
				}
				return response(output, map[string]any{"candlesticks": []any{row}})
			}))
			if _, err := source.LoadCandles(context.Background(), market.CandleRequest{Instrument: stock("700.HK"), Interval: "1m", Limit: 2, Before: stamp.Add(time.Minute)}); err == nil {
				t.Fatal("invalid data or non-advancing cursor accepted")
			}
		})
	}
}

func TestSymbolDiscoveryVerifiesCodesAndCachesCanonicalIdentity(t *testing.T) {
	calls := 0
	source := New(quoteAPI(func(_ context.Context, path string, input, output any) error {
		calls++
		if path != "/quote/static-info" {
			t.Fatal(path)
		}
		symbol := input.(map[string]any)["symbol"].([]string)[0]
		return response(output, map[string]any{"secu_static_info": []any{map[string]any{"symbol": symbol, "name_en": "Security", "name_cn": "证券"}}})
	}))
	for _, test := range []struct{ query, symbol, display, currency, region string }{
		{"00700.hk", "700.HK", "00700.HK", "HKD", "HK"},
		{"700", "700.HK", "00700.HK", "HKD", "HK"},
		{"aapl", "AAPL.US", "AAPL.US", "USD", "US"},
		{"600519", "600519.SH", "600519.SH", "CNY", "SH"},
		{"000001", "000001.SZ", "000001.SZ", "CNY", "SZ"},
		{"腾讯", "700.HK", "00700.HK", "HKD", "HK"},
	} {
		items, err := source.SearchInstruments(context.Background(), test.query)
		if err != nil || len(items) != 1 {
			t.Fatalf("%s items=%+v, err=%v", test.query, items, err)
		}
		item := items[0]
		if item.ProviderSymbol != test.symbol || item.Symbol != test.display || item.Currency != test.currency || item.SecurityMarket != test.region || item.NameCN != "证券" {
			t.Fatalf("%s: %+v", test.query, item)
		}
	}
	if calls != 4 {
		t.Fatalf("canonical lookups=%d, want 4", calls)
	}
}

func TestDiscoveryDoesNotInventUnknownCodesOrConflateAuthorizationAndEntitlements(t *testing.T) {
	for _, failure := range []error{&auth.APIError{Code: 301600}, auth.ErrAuthorization, &auth.APIError{Status: 403, Code: 301403}} {
		source := New(quoteAPI(func(context.Context, string, any, any) error { return failure }))
		items, err := source.SearchInstruments(context.Background(), "ZZZZ.US")
		var authorization *market.SourceAuthorizationError
		var apiError *auth.APIError
		switch {
		case errors.Is(failure, auth.ErrAuthorization):
			if !errors.As(err, &authorization) || !authorization.Expired {
				t.Fatalf("authorization error=%v", err)
			}
		case errors.As(failure, &apiError) && apiError.Code == 301600:
			if len(items) != 0 || err != nil {
				t.Fatalf("unknown code fabricated: %+v, %v", items, err)
			}
		default:
			if err == nil || errors.As(err, &authorization) {
				t.Fatalf("entitlement error=%v", err)
			}
		}
	}
	if got := querySymbols("ETH/USDT"); !reflect.DeepEqual(got, []string(nil)) {
		t.Fatalf("crypto pair became a security: %v", got)
	}
}
