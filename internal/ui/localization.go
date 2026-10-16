package ui

import (
	"strconv"
	"strings"

	"trajon/internal/locales"
	"trajon/internal/market"
)

func InstrumentTypeLabel(text locales.Messages, item market.Instrument) string {
	sourceLabel := item.SourceLabel()
	if item.SourceID == market.DataSourceLongbridge {
		sourceLabel = text.Longbridge()
	}
	if item.MarketType == "" {
		return sourceLabel
	}
	marketLabel := text.Spot()
	if item.MarketType == market.MarketFutures {
		marketLabel = text.Futures()
	}
	if item.MarketType == market.MarketStock {
		marketLabel = text.Stocks()
		switch item.SecurityMarket {
		case "US":
			marketLabel = text.USStocks()
		case "HK":
			marketLabel = text.HKStocks()
		case "SH", "SZ":
			marketLabel = text.ChinaStocks()
		}
	}
	return sourceLabel + " · " + marketLabel
}

// MarketError localizes our structured errors and preserves provider/system
// details verbatim. It never changes shared errors or market data.
func MarketError(text locales.Messages, err error) string {
	if err == nil {
		return ""
	}
	switch err := err.(type) {
	case *market.MissingSourceError:
		return text.MissingSource(strconv.Quote(string(err.SourceID)))
	case *market.SourceAuthorizationError:
		name := (market.Instrument{SourceID: err.SourceID}).SourceLabel()
		if err.SourceID == market.DataSourceLongbridge {
			name = text.Longbridge()
		}
		message := text.DataSourceAuthorizationRequired(name)
		if err.Expired {
			message = text.DataSourceAuthorizationExpired(name)
		}
		if err.Err != nil {
			message += "\n" + err.Err.Error()
		}
		return message
	case *market.NoCandlesError:
		return text.NoCandles(err.Symbol)
	case *market.InstrumentSearchError:
		item := market.Instrument{SourceID: err.SourceID, MarketType: err.MarketType}
		return InstrumentTypeLabel(text, item) + ": " + MarketError(text, err.Err)
	case interface{ Unwrap() []error }:
		var messages []string
		for _, cause := range err.Unwrap() {
			messages = append(messages, MarketError(text, cause))
		}
		return strings.Join(messages, "\n")
	}
	if err == market.ErrNoInstrumentSource {
		return text.NoSearchSource()
	}
	return err.Error()
}
