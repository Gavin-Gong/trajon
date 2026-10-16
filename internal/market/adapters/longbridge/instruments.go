package longbridge

import (
	"context"
	"errors"
	"strings"
	"time"

	auth "trajon/internal/longbridge"
	"trajon/internal/market"
)

type securityInfo struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name_en"`
	NameCN   string `json:"name_cn"`
	NameHK   string `json:"name_hk"`
	Currency string `json:"currency"`
	Board    string `json:"board"`
}

// The public quote API resolves known security codes; it does not expose a
// whole-market free-text search. These starters cover common names, while exact
// codes are verified against static-info rather than silently fabricated.
func (s *Source) SuggestedInstruments() []market.Instrument {
	rows := []securityInfo{
		{Symbol: "AAPL.US", Name: "Apple", NameCN: "苹果"},
		{Symbol: "TSLA.US", Name: "Tesla", NameCN: "特斯拉"},
		{Symbol: "SPY.US", Name: "SPDR S&P 500 ETF", NameCN: "标普500 ETF"},
		{Symbol: "700.HK", Name: "Tencent", NameCN: "腾讯控股"},
		{Symbol: "9988.HK", Name: "Alibaba", NameCN: "阿里巴巴"},
		{Symbol: "2800.HK", Name: "Tracker Fund of Hong Kong", NameCN: "盈富基金"},
		{Symbol: "600519.SH", Name: "Kweichow Moutai", NameCN: "贵州茅台"},
		{Symbol: "510300.SH", Name: "CSI 300 ETF", NameCN: "沪深300 ETF"},
		{Symbol: "000001.SZ", Name: "Ping An Bank", NameCN: "平安银行"},
		{Symbol: "159919.SZ", Name: "Harvest CSI 300 ETF", NameCN: "嘉实沪深300 ETF"},
	}
	items := make([]market.Instrument, 0, len(rows))
	for _, row := range rows {
		if item, ok := instrument(row); ok {
			items = append(items, item)
		}
	}
	return items
}

func (s *Source) SearchInstruments(ctx context.Context, query string) ([]market.Instrument, error) {
	query = strings.TrimSpace(query)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	candidates := querySymbols(query)
	for _, item := range market.FilterInstruments(s.SuggestedInstruments(), query, market.MarketStock) {
		candidates = append(candidates, item.ProviderSymbol)
	}
	seen := make(map[string]bool)
	var items []market.Instrument
	var failures []error
	for _, symbol := range candidates {
		if seen[symbol] {
			continue
		}
		seen[symbol] = true
		item, found, err := s.lookup(ctx, symbol)
		if err != nil {
			// An optional, disconnected data source must not turn every crypto
			// search into a partial-failure warning. The Stocks tab supplies a
			// connection entry; explicit suffixed queries still get a useful error.
			if errors.Is(err, auth.ErrNotConnected) && !strings.Contains(query, ".") {
				return nil, nil
			}
			failures = append(failures, sourceError(err))
			continue
		}
		if found {
			items = append(items, item)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return items, errors.Join(failures...)
}

func (s *Source) lookup(ctx context.Context, symbol string) (market.Instrument, bool, error) {
	s.mu.Lock()
	cached, found := s.cache[symbol]
	s.mu.Unlock()
	if found && time.Since(cached.at) < 5*time.Minute {
		return cached.item, true, nil
	}
	var response struct {
		Items []securityInfo `json:"secu_static_info"`
	}
	err := s.request(ctx, "/quote/static-info", map[string]any{"symbol": []string{symbol}}, &response)
	if err != nil {
		var apiError *auth.APIError
		if errors.As(err, &apiError) && apiError.Code == 301600 {
			return market.Instrument{}, false, nil
		}
		return market.Instrument{}, false, err
	}
	for _, row := range response.Items {
		item, ok := instrument(row)
		if !ok || item.ProviderSymbol != symbol {
			continue
		}
		s.mu.Lock()
		if len(s.cache) >= 512 {
			clear(s.cache)
		}
		s.cache[symbol] = cachedInstrument{item: item, at: time.Now()}
		s.mu.Unlock()
		return item, true, nil
	}
	return market.Instrument{}, false, nil
}

func querySymbols(query string) []string {
	query = strings.ToUpper(strings.TrimSpace(query))
	if canonical := canonicalSymbol(query); canonical != "" {
		return []string{canonical}
	}
	if query == "" || strings.ContainsAny(query, "/_ ") {
		return nil
	}
	if canonical := canonicalSymbol(query + ".US"); canonical != "" {
		return []string{canonical}
	}
	if canonical := canonicalSymbol(query + ".HK"); canonical != "" {
		return []string{canonical}
	}
	if len(query) == 6 {
		suffix := ".SH"
		if strings.ContainsRune("0123", rune(query[0])) {
			suffix = ".SZ"
		}
		if canonical := canonicalSymbol(query + suffix); canonical != "" {
			return []string{canonical}
		}
	}
	return nil
}

func instrument(row securityInfo) (market.Instrument, bool) {
	symbol := canonicalSymbol(row.Symbol)
	board := strings.ToLower(row.Board)
	if symbol == "" || strings.Contains(board, "option") || strings.Contains(board, "warrant") || strings.Contains(board, "bull") || strings.Contains(board, "bear") {
		return market.Instrument{}, false
	}
	_, securityMarket, _ := strings.Cut(symbol, ".")
	if strings.HasSuffix(symbol, ".US") {
		securityMarket = "US"
	}
	currency := row.Currency
	if currency == "" {
		currency = "CNY"
		if securityMarket == "US" {
			currency = "USD"
		} else if securityMarket == "HK" {
			currency = "HKD"
		}
	}
	displaySymbol := symbol
	if securityMarket == "HK" {
		code, _, _ := strings.Cut(symbol, ".")
		displaySymbol = strings.Repeat("0", 5-len(code)) + code + ".HK"
	}
	name := row.Name
	if name == "" {
		name = row.NameCN
	}
	if name == "" {
		name = displaySymbol
	}
	nameCN := row.NameCN
	if nameCN == "" {
		nameCN = row.NameHK
	}
	return market.Instrument{Name: name, NameCN: nameCN, Symbol: displaySymbol, ProviderSymbol: symbol,
		Currency: currency, SecurityMarket: securityMarket, SourceID: market.DataSourceLongbridge, MarketType: market.MarketStock}, true
}
