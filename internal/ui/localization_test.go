package ui

import (
	"errors"
	"strconv"
	"testing"

	"trajon/internal/locales"
	"trajon/internal/market"
)

func TestMarketLabelsAndErrorsAreLocalizedWithoutChangingDomainData(t *testing.T) {
	item := market.DefaultInstrument()
	identity := item.Identity()
	request := market.CandleRequest{Instrument: item, Interval: "1h", Limit: 100}
	key := request.CacheKey()
	providerErr := errors.New("HTTP 503: provider unavailable")
	searchErr := &market.InstrumentSearchError{SourceID: market.DataSourceOKX, MarketType: market.MarketFutures, Err: providerErr}
	for _, tc := range []struct {
		language                                          locales.Language
		spot, futures, missing, candles, search, noSource string
	}{
		{locales.Chinese, "OKX · 现货", "OKX · 合约", "未找到数据源 \"okx\"", "BTC/USDT 没有可用 K 线", "OKX · 合约: HTTP 503: provider unavailable", "未找到标的搜索数据源"},
		{locales.English, "OKX · Spot", "OKX · Futures", "Data source \"okx\" not found", "No candles available for BTC/USDT", "OKX · Futures: HTTP 503: provider unavailable", "No instrument search source available"},
	} {
		text := locales.In(string(tc.language))
		if got := InstrumentTypeLabel(text, item); got != tc.spot {
			t.Errorf("spot = %q", got)
		}
		future := item
		future.MarketType = market.MarketFutures
		if got := InstrumentTypeLabel(text, future); got != tc.futures {
			t.Errorf("futures = %q", got)
		}
		for _, check := range []struct {
			err  error
			want string
		}{
			{&market.MissingSourceError{SourceID: market.DataSourceOKX}, tc.missing},
			{&market.NoCandlesError{Symbol: item.Symbol}, tc.candles},
			{searchErr, tc.search}, {market.ErrNoInstrumentSource, tc.noSource},
			{errors.Join(searchErr, providerErr), tc.search + "\n" + providerErr.Error()},
			{providerErr, providerErr.Error()}, {nil, ""},
		} {
			if got := MarketError(text, check.err); got != check.want {
				t.Errorf("localized error = %q, want %q", got, check.want)
			}
		}
	}
	// ICU placeholders are values, not patterns; provider braces and quotes
	// must not be parsed as messages or lose the previous %q escaping.
	source := "custom\"\\\n{source}"
	for _, language := range []locales.Language{locales.Chinese, locales.English} {
		text := locales.In(string(language))
		err := &market.MissingSourceError{SourceID: market.DataSourceID(source)}
		if got, want := MarketError(text, err), text.MissingSource(strconv.Quote(source)); got != want {
			t.Errorf("escaped source error = %q, want %q", got, want)
		}
	}
	if !errors.Is(searchErr, providerErr) {
		t.Fatal("search error lost its underlying cause")
	}
	if item.Identity() != identity || request.CacheKey() != key {
		t.Fatal("localization changed instrument or cache identity")
	}
}
