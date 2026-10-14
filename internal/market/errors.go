package market

import (
	"errors"
	"fmt"
)

var ErrNoInstrumentSource = errors.New("no instrument search source available")

// These errors retain domain data until the presentation layer chooses a
// language. Error remains useful for logs, without any dependency on locales.
type MissingSourceError struct{ SourceID DataSourceID }

func (e *MissingSourceError) Error() string {
	return fmt.Sprintf("data source %q not found", e.SourceID)
}

type NoCandlesError struct{ Symbol string }

func (e *NoCandlesError) Error() string {
	return fmt.Sprintf("no candles available for %s", e.Symbol)
}

type InstrumentSearchError struct {
	SourceID   DataSourceID
	MarketType MarketType
	Err        error
}

func (e *InstrumentSearchError) Error() string {
	return fmt.Sprintf("%s · %s: %v", (Instrument{SourceID: e.SourceID}).SourceLabel(), e.MarketType, e.Err)
}

func (e *InstrumentSearchError) Unwrap() error { return e.Err }
