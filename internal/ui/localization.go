package ui

import (
	"fmt"
	"strings"

	"trajon/internal/locales"
	"trajon/internal/market"
)

func InstrumentTypeLabel(text *locales.Strings, item market.Instrument) string {
	marketLabel := text.Spot
	if item.MarketType == market.MarketFutures {
		marketLabel = text.Futures
	}
	return item.SourceLabel() + " · " + marketLabel
}

// MarketError localizes our structured errors and preserves provider/system
// details verbatim. It never changes shared errors or market data.
func MarketError(text *locales.Strings, err error) string {
	if err == nil {
		return ""
	}
	switch err := err.(type) {
	case *market.MissingSourceError:
		return fmt.Sprintf(text.MissingSource, err.SourceID)
	case *market.NoCandlesError:
		return fmt.Sprintf(text.NoCandles, err.Symbol)
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
		return text.NoSearchSource
	}
	return err.Error()
}
