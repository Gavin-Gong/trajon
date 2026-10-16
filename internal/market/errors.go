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

// SourceAuthorizationError distinguishes missing/expired grants from network
// and quote-entitlement failures without depending on a provider SDK.
type SourceAuthorizationError struct {
	SourceID DataSourceID
	Expired  bool
	Err      error // Optional sanitized provider detail.
}

func (e *SourceAuthorizationError) Error() string {
	message := fmt.Sprintf("%s requires authorization", (Instrument{SourceID: e.SourceID}).SourceLabel())
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	return message
}

func (e *SourceAuthorizationError) Unwrap() error { return e.Err }

type InstrumentSearchError struct {
	SourceID   DataSourceID
	MarketType MarketType
	Err        error
}

func (e *InstrumentSearchError) Error() string {
	return fmt.Sprintf("%s · %s: %v", (Instrument{SourceID: e.SourceID}).SourceLabel(), e.MarketType, e.Err)
}

func (e *InstrumentSearchError) Unwrap() error { return e.Err }
