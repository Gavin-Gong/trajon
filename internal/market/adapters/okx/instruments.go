package okx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"trajon/internal/market"
)

func (s *Source) LoadInstruments(ctx context.Context, marketType market.MarketType) ([]market.Instrument, error) {
	instType, contractType := "SPOT", market.ContractNone
	switch marketType {
	case market.MarketSpot:
	case market.MarketFutures:
		instType, contractType = "SWAP", market.ContractOKXSwap
	default:
		return nil, fmt.Errorf("unsupported OKX market type %q", marketType)
	}
	params := url.Values{"instType": {instType}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/api/v5/public/instruments?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create OKX instruments request: %w", err)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request OKX instruments: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return nil, fmt.Errorf("OKX instruments returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstID     string `json:"instId"`
			InstType   string `json:"instType"`
			State      string `json:"state"`
			BaseCcy    string `json:"baseCcy"`
			QuoteCcy   string `json:"quoteCcy"`
			InstFamily string `json:"instFamily"`
			CtType     string `json:"ctType"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode OKX instruments: %w", err)
	}
	if payload.Code != "0" {
		return nil, fmt.Errorf("OKX instruments returned code %s: %s", payload.Code, payload.Msg)
	}
	items := make([]market.Instrument, 0, len(payload.Data))
	for _, row := range payload.Data {
		if row.State != "live" || row.InstType != instType || row.InstID == "" {
			continue
		}
		base, quote := row.BaseCcy, row.QuoteCcy
		if marketType == market.MarketFutures {
			// The replay model uses linear P&L, so inverse/coin-margined
			// contracts must not be offered as interchangeable instruments.
			if row.CtType != "linear" {
				continue
			}
			base, quote, _ = strings.Cut(row.InstFamily, "-")
		}
		if base == "" || quote == "" {
			continue
		}
		items = append(items, market.Instrument{
			Name: base, Symbol: base + "/" + quote, ProviderSymbol: row.InstID,
			SourceID: market.DataSourceOKX, MarketType: marketType, ContractType: contractType,
		})
	}
	return items, nil
}
