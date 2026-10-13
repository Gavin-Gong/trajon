package market

import "strings"

var instrumentCatalog = []Instrument{
	{Name: "Bitcoin", Symbol: "BTC/USDT", ProviderSymbol: "BTCUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Bitcoin", Symbol: "BTC/USDT", ProviderSymbol: "BTCUSDT", SourceID: DataSourceBinance, MarketType: MarketFutures, ContractType: ContractUSDM},
	{Name: "Ethereum", Symbol: "ETH/USDT", ProviderSymbol: "ETHUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Ethereum", Symbol: "ETH/USDT", ProviderSymbol: "ETHUSDT", SourceID: DataSourceBinance, MarketType: MarketFutures, ContractType: ContractUSDM},
	{Name: "Solana", Symbol: "SOL/USDT", ProviderSymbol: "SOLUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Solana", Symbol: "SOL/USDT", ProviderSymbol: "SOLUSDT", SourceID: DataSourceBinance, MarketType: MarketFutures, ContractType: ContractUSDM},
	{Name: "Dogecoin", Symbol: "DOGE/USDT", ProviderSymbol: "DOGEUSDT", SourceID: DataSourceBinance, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Dogecoin", Symbol: "DOGE/USDT", ProviderSymbol: "DOGEUSDT", SourceID: DataSourceBinance, MarketType: MarketFutures, ContractType: ContractUSDM},
	{Name: "Bitcoin", Symbol: "BTC/USDT", ProviderSymbol: "BTC-USDT", SourceID: DataSourceOKX, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Bitcoin", Symbol: "BTC/USDT", ProviderSymbol: "BTC-USDT-SWAP", SourceID: DataSourceOKX, MarketType: MarketFutures, ContractType: ContractOKXSwap},
	{Name: "Ethereum", Symbol: "ETH/USDT", ProviderSymbol: "ETH-USDT", SourceID: DataSourceOKX, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Ethereum", Symbol: "ETH/USDT", ProviderSymbol: "ETH-USDT-SWAP", SourceID: DataSourceOKX, MarketType: MarketFutures, ContractType: ContractOKXSwap},
	{Name: "Solana", Symbol: "SOL/USDT", ProviderSymbol: "SOL-USDT", SourceID: DataSourceOKX, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Solana", Symbol: "SOL/USDT", ProviderSymbol: "SOL-USDT-SWAP", SourceID: DataSourceOKX, MarketType: MarketFutures, ContractType: ContractOKXSwap},
	{Name: "Dogecoin", Symbol: "DOGE/USDT", ProviderSymbol: "DOGE-USDT", SourceID: DataSourceOKX, MarketType: MarketSpot, ContractType: ContractNone},
	{Name: "Dogecoin", Symbol: "DOGE/USDT", ProviderSymbol: "DOGE-USDT-SWAP", SourceID: DataSourceOKX, MarketType: MarketFutures, ContractType: ContractOKXSwap},
}

func InstrumentCatalog() []Instrument {
	items := make([]Instrument, len(instrumentCatalog))
	copy(items, instrumentCatalog)
	return items
}

func DefaultInstrument() Instrument {
	return instrumentCatalog[8]
}

func FilterInstruments(items []Instrument, query string, marketType MarketType) []Instrument {
	query = strings.ToLower(strings.TrimSpace(query))
	filtered := make([]Instrument, 0, len(items))
	for _, item := range items {
		if marketType != "" && item.MarketType != marketType {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(item.Name), query) &&
			!strings.Contains(strings.ToLower(item.Symbol), query) &&
			!strings.Contains(strings.ToLower(item.ProviderSymbol), query) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}
