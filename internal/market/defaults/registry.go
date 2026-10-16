package defaults

import (
	"net/http"

	"trajon/internal/market"
	"trajon/internal/market/adapters/binance"
	"trajon/internal/market/adapters/longbridge"
	"trajon/internal/market/adapters/okx"
)

func NewRegistry(client *http.Client, longbridgeAPI longbridge.API) *market.Registry {
	return market.NewRegistry(map[market.DataSourceID]market.CandleSource{
		market.DataSourceBinance:    binance.New(client),
		market.DataSourceOKX:        okx.New(client),
		market.DataSourceLongbridge: longbridge.New(longbridgeAPI),
	})
}
