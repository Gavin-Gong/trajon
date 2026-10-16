package market

import "strings"

var instrumentCatalog = []Instrument{
	{Name: "Bitcoin", Symbol: "BTC/USDT", ProviderSymbol: "BTCUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Bitcoin", Symbol: "BTC/USDT", ProviderSymbol: "BTCUSDT", SourceID: DataSourceBinance, MarketType: MarketFutures, ContractType: ContractUSDM},
	{Name: "Bitcoin", Symbol: "BTC/USDT", ProviderSymbol: "BTC-USDT", SourceID: DataSourceOKX, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Bitcoin", Symbol: "BTC/USDT", ProviderSymbol: "BTC-USDT-SWAP", SourceID: DataSourceOKX, MarketType: MarketFutures, ContractType: ContractOKXSwap},
}

func InstrumentCatalog() []Instrument {
	items := make([]Instrument, len(instrumentCatalog))
	copy(items, instrumentCatalog)
	return items
}

func DefaultInstrument() Instrument {
	return instrumentCatalog[2]
}

func FilterInstruments(items []Instrument, query string, marketType MarketType) []Instrument {
	query = normalizeInstrumentQuery(query)
	filtered := make([]Instrument, 0, len(items))
	for _, item := range items {
		if marketType != "" && item.MarketType != marketType {
			continue
		}
		if query != "" && !strings.Contains(normalizeInstrumentQuery(item.Name), query) &&
			!strings.Contains(normalizeInstrumentQuery(item.NameCN), query) &&
			!strings.Contains(normalizeInstrumentQuery(item.Symbol), query) &&
			!strings.Contains(normalizeInstrumentQuery(item.ProviderSymbol), query) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func normalizeInstrumentQuery(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '-', '_', ' ':
			return -1
		}
		return r
	}, strings.ToLower(strings.TrimSpace(value)))
}
