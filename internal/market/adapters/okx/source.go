package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"trajon/internal/market"
)

const pageLimit = 300

type Source struct {
	client  *http.Client
	baseURL string
}

type candleResponse struct {
	Code string              `json:"code"`
	Msg  string              `json:"msg"`
	Data [][]json.RawMessage `json:"data"`
}

func New(client *http.Client) *Source {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	baseURL := os.Getenv("OKX_BASE_URL")
	if baseURL == "" {
		baseURL = "https://www.okx.com"
	}
	return &Source{client: client, baseURL: strings.TrimRight(baseURL, "/")}
}

func (s *Source) LoadCandles(ctx context.Context, request market.CandleRequest) ([]market.Candle, error) {
	if request.Instrument.SourceID != market.DataSourceOKX {
		return nil, fmt.Errorf("OKX source cannot load %q instruments", request.Instrument.SourceID)
	}
	if request.Instrument.ProviderSymbol == "" {
		return nil, errors.New("OKX instrument has no provider symbol")
	}
	if request.Limit <= 0 || request.Limit > 1000 {
		return nil, fmt.Errorf("OKX candle limit must be between 1 and 1000, got %d", request.Limit)
	}
	bar, ok := interval(request.Interval)
	if !ok {
		return nil, fmt.Errorf("unsupported OKX interval %q", request.Interval)
	}
	path, err := endpoint(request.Instrument)
	if err != nil {
		return nil, err
	}

	candles := make([]market.Candle, 0, request.Limit)
	seen := make(map[int64]struct{}, request.Limit)
	cursor := ""
	for page := 0; len(candles) < request.Limit && page < 20; page++ {
		pageSize := min(pageLimit, request.Limit-len(candles))
		params := url.Values{}
		params.Set("instId", request.Instrument.ProviderSymbol)
		params.Set("bar", bar)
		params.Set("limit", strconv.Itoa(pageSize))
		if cursor == "" && !request.Before.IsZero() {
			cursor = strconv.FormatInt(request.Before.UnixMilli(), 10)
		}
		if cursor != "" {
			params.Set("after", cursor)
		}
		endpointURL := strings.TrimRight(s.baseURL, "/") + path + "?" + params.Encode()
		rows, err := s.fetchPage(ctx, endpointURL)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}

		oldest := int64(0)
		added := 0
		for index, row := range rows {
			candle, err := decodeCandle(row)
			if err != nil {
				return nil, fmt.Errorf("decode OKX candle %d: %w", index, err)
			}
			timestamp := candle.Time.UnixMilli()
			if oldest == 0 || timestamp < oldest {
				oldest = timestamp
			}
			if _, exists := seen[timestamp]; exists {
				continue
			}
			seen[timestamp] = struct{}{}
			candles = append(candles, candle)
			added++
		}
		if added == 0 || oldest == 0 {
			break
		}
		cursor = strconv.FormatInt(oldest, 10)
	}
	if len(candles) == 0 {
		return nil, errors.New("OKX returned no candles")
	}

	sort.Slice(candles, func(i, j int) bool { return candles[i].Time.Before(candles[j].Time) })
	if len(candles) > request.Limit {
		candles = candles[len(candles)-request.Limit:]
	}
	market.AddEMA20(candles)
	return candles, nil
}

func (s *Source) fetchPage(ctx context.Context, endpointURL string) ([][]json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create OKX request: %w", err)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request OKX candles: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return nil, fmt.Errorf("OKX candles returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var payload candleResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode OKX candles: %w", err)
	}
	if payload.Code != "0" {
		return nil, fmt.Errorf("OKX candles returned code %s: %s", payload.Code, payload.Msg)
	}
	return payload.Data, nil
}

func endpoint(item market.Instrument) (string, error) {
	switch item.MarketType {
	case market.MarketSpot:
		if item.ContractType != market.ContractNone {
			return "", fmt.Errorf("OKX spot instrument has contract type %q", item.ContractType)
		}
	case market.MarketFutures:
		if item.ContractType != market.ContractOKXSwap {
			return "", fmt.Errorf("unsupported OKX futures contract %q", item.ContractType)
		}
	default:
		return "", fmt.Errorf("unsupported OKX market type %q", item.MarketType)
	}
	return "/api/v5/market/candles", nil
}

func interval(value string) (string, bool) {
	switch value {
	case "15m":
		return "15m", true
	case "1h":
		return "1H", true
	case "4h":
		return "4H", true
	case "1D":
		return "1Dutc", true
	default:
		return "", false
	}
}

func decodeCandle(row []json.RawMessage) (market.Candle, error) {
	if len(row) < 6 {
		return market.Candle{}, fmt.Errorf("expected at least 6 fields, got %d", len(row))
	}
	parseString := func(index int, label string) (string, error) {
		var value string
		if err := json.Unmarshal(row[index], &value); err != nil {
			return "", fmt.Errorf("%s: %w", label, err)
		}
		return value, nil
	}
	parseNumber := func(index int, label string) (float64, error) {
		value, err := parseString(index, label)
		if err != nil {
			return 0, err
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("%s %q: %w", label, value, err)
		}
		return parsed, nil
	}

	timestamp, err := parseString(0, "timestamp")
	if err != nil {
		return market.Candle{}, err
	}
	milliseconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return market.Candle{}, fmt.Errorf("timestamp %q: %w", timestamp, err)
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
		Time:   time.UnixMilli(milliseconds).UTC(),
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: volume,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
