package market

import (
	"strconv"
	"strings"
	"time"
)

type DataSourceID string

type MarketType string

type ContractType string

const (
	DataSourceBinance DataSourceID = "binance"
	DataSourceOKX     DataSourceID = "okx"

	MarketSpot    MarketType = "spot"
	MarketFutures MarketType = "futures"

	ContractNone    ContractType = ""
	ContractUSDM    ContractType = "usd-m"
	ContractOKXSwap ContractType = "okx-swap"
)

type Instrument struct {
	Name           string
	Symbol         string
	ProviderSymbol string
	SourceID       DataSourceID
	MarketType     MarketType
	ContractType   ContractType
}

// SourceLabel is the provider name, independent of the UI language.
func (i Instrument) SourceLabel() string {
	sourceLabel := string(i.SourceID)
	switch i.SourceID {
	case DataSourceBinance:
		sourceLabel = "Binance"
	case DataSourceOKX:
		sourceLabel = "OKX"
	}
	return sourceLabel
}

func (i Instrument) Identity() string {
	return strings.Join([]string{
		string(i.SourceID),
		string(i.MarketType),
		string(i.ContractType),
		i.ProviderSymbol,
	}, "|")
}

type Candle struct {
	Time            time.Time
	Open, High, Low float64
	Close, Volume   float64
	EMA20           float64
}

type CandleRequest struct {
	Instrument Instrument
	Interval   string
	Limit      int
	Before     time.Time
}

func (r CandleRequest) CacheKey() string {
	return strings.Join([]string{
		string(r.Instrument.SourceID),
		string(r.Instrument.MarketType),
		string(r.Instrument.ContractType),
		r.Instrument.ProviderSymbol,
		r.Interval,
		strconv.Itoa(r.Limit),
		strconv.FormatInt(r.Before.UnixMilli(), 10),
	}, "|")
}

func CloneCandles(candles []Candle) []Candle {
	if candles == nil {
		return nil
	}
	copyCandles := make([]Candle, len(candles))
	copy(copyCandles, candles)
	return copyCandles
}
