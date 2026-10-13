package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"trajon/internal/market"
)

const candleLimit = 1000

type Source struct {
	client         *http.Client
	spotBaseURL    string
	futuresBaseURL string
}

func New(client *http.Client) *Source {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Source{
		client:         client,
		spotBaseURL:    "https://api.binance.com",
		futuresBaseURL: "https://fapi.binance.com",
	}
}

func (s *Source) LoadCandles(ctx context.Context, request market.CandleRequest) ([]market.Candle, error) {
	if request.Instrument.SourceID != market.DataSourceBinance {
		return nil, fmt.Errorf("binance source cannot load %q instruments", request.Instrument.SourceID)
	}
	if request.Instrument.ProviderSymbol == "" {
		return nil, errors.New("binance instrument has no provider symbol")
	}
	if request.Limit <= 0 || request.Limit > candleLimit {
		return nil, fmt.Errorf("binance candle limit must be between 1 and %d, got %d", candleLimit, request.Limit)
	}

	interval, ok := interval(request.Interval)
	if !ok {
		return nil, fmt.Errorf("unsupported Binance interval %q", request.Interval)
	}
	baseURL, path, err := s.endpoint(request.Instrument)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", request.Instrument.ProviderSymbol)
	params.Set("interval", interval)
	params.Set("limit", strconv.Itoa(request.Limit))
	if !request.Before.IsZero() {
		params.Set("endTime", strconv.FormatInt(request.Before.Add(-time.Millisecond).UnixMilli(), 10))
	}
	endpoint := strings.TrimRight(baseURL, "/") + path + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create Binance request: %w", err)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Binance klines: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return nil, fmt.Errorf("Binance klines returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var rows [][]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode Binance klines: %w", err)
	}
	candles := make([]market.Candle, 0, len(rows))
	for index, row := range rows {
		candle, err := decodeCandle(row)
		if err != nil {
			return nil, fmt.Errorf("decode Binance candle %d: %w", index, err)
		}
		candles = append(candles, candle)
	}
	if len(candles) == 0 {
		return nil, errors.New("Binance returned no candles")
	}
	market.AddEMA20(candles)
	return candles, nil
}

func (s *Source) endpoint(item market.Instrument) (string, string, error) {
	switch item.MarketType {
	case market.MarketSpot:
		return s.spotBaseURL, "/api/v3/klines", nil
	case market.MarketFutures:
		if item.ContractType != market.ContractUSDM {
			return "", "", fmt.Errorf("unsupported Binance futures contract %q", item.ContractType)
		}
		return s.futuresBaseURL, "/fapi/v1/klines", nil
	default:
		return "", "", fmt.Errorf("unsupported Binance market type %q", item.MarketType)
	}
}

func interval(value string) (string, bool) {
	switch value {
	case "15m", "1h", "4h":
		return value, true
	case "1D":
		return "1d", true
	default:
		return "", false
	}
}

func decodeCandle(row []json.RawMessage) (market.Candle, error) {
	if len(row) < 6 {
		return market.Candle{}, fmt.Errorf("expected at least 6 fields, got %d", len(row))
	}
	var openMilliseconds int64
	if err := json.Unmarshal(row[0], &openMilliseconds); err != nil {
		return market.Candle{}, fmt.Errorf("open time: %w", err)
	}
	parseNumber := func(index int, label string) (float64, error) {
		var value string
		if err := json.Unmarshal(row[index], &value); err != nil {
			return 0, fmt.Errorf("%s: %w", label, err)
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("%s %q: %w", label, value, err)
		}
		return parsed, nil
	}
	open, err := parseNumber(1, "open")
	if err != nil {
		return market.Candle{}, err
	}
	high, err := parseNumber(2, "high")
	if err != nil {
		return market.Candle{}, err
	}
	low, err := parseNumber(3, "low")
	if err != nil {
		return market.Candle{}, err
	}
	close, err := parseNumber(4, "close")
	if err != nil {
		return market.Candle{}, err
	}
	volume, err := parseNumber(5, "volume")
	if err != nil {
		return market.Candle{}, err
	}
	return market.Candle{
		Time:   time.UnixMilli(openMilliseconds).UTC(),
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: volume,
	}, nil
}
