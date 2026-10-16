package longbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	auth "trajon/internal/longbridge"
	"trajon/internal/market"
)

// API is the vault-backed OAuth owner's read-only quote transport.
type API interface {
	Request(context.Context, string, any, any) error
}

type Source struct {
	api   API
	mu    sync.Mutex
	cache map[string]cachedInstrument
}

type cachedInstrument struct {
	item market.Instrument
	at   time.Time
}

func New(api API) *Source {
	return &Source{api: api, cache: make(map[string]cachedInstrument)}
}

type historyRequest struct {
	Symbol       string      `json:"symbol"`
	Period       int         `json:"period"`
	AdjustType   int         `json:"adjust_type"`
	QueryType    int         `json:"query_type"`
	TradeSession int         `json:"trade_session"`
	Offset       offsetQuery `json:"offset_request"`
}

type offsetQuery struct {
	Direction int    `json:"direction"`
	Date      string `json:"date,omitempty"`
	Minute    string `json:"minute,omitempty"`
	Count     int    `json:"count"`
}

type candleRow struct {
	Timestamp json.Number     `json:"timestamp"`
	Open      json.Number     `json:"open"`
	High      json.Number     `json:"high"`
	Low       json.Number     `json:"low"`
	Close     json.Number     `json:"close"`
	Volume    json.Number     `json:"volume"`
	Session   json.RawMessage `json:"trade_session"`
}

func (s *Source) LoadCandles(ctx context.Context, request market.CandleRequest) ([]market.Candle, error) {
	item := request.Instrument
	if item.SourceID != market.DataSourceLongbridge || item.MarketType != market.MarketStock || item.ContractType != market.ContractNone {
		return nil, errors.New("Longbridge requires a securities instrument")
	}
	symbol := canonicalSymbol(item.ProviderSymbol)
	period, ok := interval(request.Interval)
	if symbol == "" || !ok || request.Limit < 1 || request.Limit > 1000 {
		return nil, errors.New("invalid Longbridge symbol, interval, or candle limit")
	}
	location, err := symbolLocation(symbol)
	if err != nil {
		return nil, err
	}
	candles := make([]market.Candle, 0, request.Limit)
	seen := make(map[int64]bool)
	cursor := request.Before
	for page := 0; page < 20; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Official history offsets are exchange-local date/minute values, not
		// UTC or the machine's timezone. Going one instant back makes Before
		// exclusive even when the gateway includes its offset bar.
		offset := offsetQuery{Count: 1000}
		if !cursor.IsZero() {
			local := cursor.Add(-time.Nanosecond).In(location)
			offset.Date, offset.Minute = local.Format("20060102"), local.Format("1504")
		}
		query := historyRequest{Symbol: symbol, Period: period, AdjustType: 1, QueryType: 1, TradeSession: 0, Offset: offset}
		var response struct {
			Symbol  string      `json:"symbol"`
			Candles []candleRow `json:"candlesticks"`
		}
		if err := s.request(ctx, "/quote/history-candlesticks", query, &response); err != nil {
			return nil, sourceError(err)
		}
		if response.Symbol != "" && canonicalSymbol(response.Symbol) != symbol {
			return nil, auth.ErrResponse
		}
		if len(response.Candles) == 0 {
			break
		}
		var oldest time.Time
		for _, row := range response.Candles {
			candle, normal, err := decodeCandle(row)
			if err != nil {
				return nil, err
			}
			if oldest.IsZero() || candle.Time.Before(oldest) {
				oldest = candle.Time
			}
			// Some HTTP pages include US extended-session bars even for a
			// regular-session query. For intraday bars, enforce the exchange
			// clock too; calendar candles can legitimately be stamped at midnight.
			if period < 1000 && strings.HasSuffix(symbol, ".US") {
				local := candle.Time.In(location)
				minute := local.Hour()*60 + local.Minute()
				normal = normal && minute >= 9*60+30 && minute < 16*60
			}
			if !normal || !request.Before.IsZero() && !candle.Time.Before(request.Before) || seen[candle.Time.Unix()] {
				continue
			}
			seen[candle.Time.Unix()] = true
			candles = append(candles, candle)
		}
		if len(candles) >= request.Limit {
			break
		}
		if oldest.IsZero() || !cursor.IsZero() && !oldest.Before(cursor) {
			return nil, errors.New("Longbridge history cursor did not advance")
		}
		cursor = oldest
		if page == 19 && len(candles) == 0 {
			return nil, errors.New("Longbridge history pagination limit reached")
		}
	}
	sort.Slice(candles, func(i, j int) bool { return candles[i].Time.Before(candles[j].Time) })
	if len(candles) > request.Limit {
		candles = candles[len(candles)-request.Limit:]
	}
	market.AddEMA20(candles)
	return candles, nil
}

func interval(value string) (int, bool) {
	// Values are the official quote Period enum, including the intraday
	// periods not re-exported as named constants by the Go SDK.
	periods := map[string]int{"1m": 1, "3m": 3, "5m": 5, "15m": 15, "30m": 30,
		"1h": 60, "2h": 120, "4h": 240, "1D": 1000, "1W": 2000, "1M": 3000}
	period, ok := periods[value]
	return period, ok
}

func symbolLocation(symbol string) (*time.Location, error) {
	zone := "Asia/Shanghai"
	if strings.HasSuffix(symbol, ".US") {
		zone = "America/New_York"
	} else if strings.HasSuffix(symbol, ".HK") {
		zone = "Asia/Hong_Kong"
	}
	return time.LoadLocation(zone)
}

func decodeCandle(row candleRow) (market.Candle, bool, error) {
	stamp, err := row.Timestamp.Int64()
	if err != nil || stamp <= 0 {
		return market.Candle{}, false, auth.ErrResponse
	}
	values := []json.Number{row.Open, row.High, row.Low, row.Close, row.Volume}
	parsed := make([]float64, len(values))
	for i, value := range values {
		number, err := value.Float64()
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || i < 4 && number == 0 {
			return market.Candle{}, false, auth.ErrResponse
		}
		parsed[i] = number
	}
	if parsed[1] < max(parsed[0], parsed[3]) || parsed[2] > min(parsed[0], parsed[3]) {
		return market.Candle{}, false, auth.ErrResponse
	}
	normal := true
	if len(row.Session) > 0 && string(row.Session) != "null" {
		var session string
		if json.Unmarshal(row.Session, &session) != nil {
			session = string(row.Session)
		}
		switch session {
		case "0", "NORMAL_TRADE":
		case "1", "2", "3", "PRE_TRADE", "POST_TRADE", "OVERNIGHT_TRADE":
			normal = false
		default:
			return market.Candle{}, false, auth.ErrResponse
		}
	}
	return market.Candle{Time: time.Unix(stamp, 0).UTC(), Open: parsed[0], High: parsed[1], Low: parsed[2], Close: parsed[3], Volume: parsed[4]}, normal, nil
}

func (s *Source) request(ctx context.Context, path string, input, output any) error {
	if s.api == nil {
		return auth.ErrNotConnected
	}
	err := s.api.Request(ctx, path, input, output)
	if errors.Is(err, auth.ErrSessionChanged) {
		// Read-only requests may retry once against the newly committed grant.
		// A disconnect then resolves to ErrNotConnected, not an expired grant.
		return s.api.Request(ctx, path, input, output)
	}
	return err
}

func sourceError(err error) error {
	if errors.Is(err, auth.ErrNotConnected) || errors.Is(err, auth.ErrAuthorization) {
		failure := &market.SourceAuthorizationError{SourceID: market.DataSourceLongbridge, Expired: errors.Is(err, auth.ErrAuthorization)}
		var detail *auth.APIError
		if errors.As(err, &detail) {
			failure.Err = detail
		}
		return failure
	}
	return err
}

func canonicalSymbol(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	code, suffix, ok := strings.Cut(value, ".")
	if !ok {
		return ""
	}
	// US class-share tickers can themselves contain a period.
	if strings.HasSuffix(value, ".US") {
		code = strings.TrimSuffix(value, ".US")
		if len(code) < 1 || len(code) > 15 || code[0] < 'A' || code[0] > 'Z' {
			return ""
		}
		for _, r := range code {
			if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '.' && r != '-' {
				return ""
			}
		}
		return code + ".US"
	}
	if suffix != "HK" && suffix != "SH" && suffix != "SZ" {
		return ""
	}
	if suffix == "HK" && len(code) > 5 || suffix != "HK" && len(code) != 6 {
		return ""
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return ""
		}
	}
	number, err := strconv.Atoi(code)
	if err != nil || number <= 0 {
		return ""
	}
	if suffix == "HK" {
		code = strconv.Itoa(number)
	}
	return fmt.Sprintf("%s.%s", code, suffix)
}
