package defaults

import (
	"net/http"

	"trajon/internal/market"
	"trajon/internal/market/adapters/binance"
	"trajon/internal/market/adapters/okx"
)

func NewRegistry(client *http.Client) *market.Registry {
	return market.NewRegistry(map[market.DataSourceID]market.CandleSource{
		market.DataSourceBinance: binance.New(client),
		market.DataSourceOKX:     okx.New(client),
	})
}
