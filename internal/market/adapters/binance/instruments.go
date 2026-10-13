package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"trajon/internal/market"
)

func (s *Source) LoadInstruments(ctx context.Context, marketType market.MarketType) ([]market.Instrument, error) {
	baseURL, path := s.spotBaseURL, "/api/v3/exchangeInfo"
	contractType := market.ContractNone
	switch marketType {
	case market.MarketSpot:
	case market.MarketFutures:
		baseURL, path, contractType = s.futuresBaseURL, "/fapi/v1/exchangeInfo", market.ContractUSDM
	default:
		return nil, fmt.Errorf("unsupported Binance market type %q", marketType)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create Binance instruments request: %w", err)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Binance instruments: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return nil, fmt.Errorf("Binance instruments returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Symbols []struct {
			Symbol               string `json:"symbol"`
			Status               string `json:"status"`
			BaseAsset            string `json:"baseAsset"`
			QuoteAsset           string `json:"quoteAsset"`
			ContractType         string `json:"contractType"`
			IsSpotTradingAllowed bool   `json:"isSpotTradingAllowed"`
		} `json:"symbols"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode Binance instruments: %w", err)
	}
	items := make([]market.Instrument, 0, len(payload.Symbols))
	for _, row := range payload.Symbols {
		if row.Status != "TRADING" || row.Symbol == "" || row.BaseAsset == "" || row.QuoteAsset == "" {
			continue
		}
		if marketType == market.MarketSpot && !row.IsSpotTradingAllowed {
			continue
		}
		if marketType == market.MarketFutures && row.ContractType != "PERPETUAL" {
			continue
		}
		items = append(items, market.Instrument{
			Name: row.BaseAsset, Symbol: row.BaseAsset + "/" + row.QuoteAsset, ProviderSymbol: row.Symbol,
			SourceID: market.DataSourceBinance, MarketType: marketType, ContractType: contractType,
		})
	}
	return items, nil
}
