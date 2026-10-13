package app

import "trajon/internal/market"

type dataSourceID = market.DataSourceID
type marketType = market.MarketType
type contractType = market.ContractType
type instrument = market.Instrument
type candleRequest = market.CandleRequest
type MarketDataSource = market.CandleSource
type marketDataRegistry = market.Registry

const (
	dataSourceBinance dataSourceID = market.DataSourceBinance
	dataSourceOKX     dataSourceID = market.DataSourceOKX

	marketSpot    marketType = market.MarketSpot
	marketFutures marketType = market.MarketFutures

	contractNone    contractType = market.ContractNone
	contractUSDM    contractType = market.ContractUSDM
	contractOKXSwap contractType = market.ContractOKXSwap
)

func newMarketDataRegistry(sources map[dataSourceID]MarketDataSource) *marketDataRegistry {
	return market.NewRegistry(sources)
}

func addEMA20(candles []Candle) {
	market.AddEMA20(candles)
}
