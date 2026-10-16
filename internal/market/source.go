package market

import (
	"context"
	"sync"
)

type CandleSource interface {
	LoadCandles(ctx context.Context, request CandleRequest) ([]Candle, error)
}

// InstrumentSource optionally exposes a source's currently tradable catalog.
type InstrumentSource interface {
	LoadInstruments(ctx context.Context, marketType MarketType) ([]Instrument, error)
}

// InstrumentQuerySource is for providers that resolve a query rather than
// publishing an entire tradable catalog. It owns matching and code validation.
type InstrumentQuerySource interface {
	SearchInstruments(context.Context, string) ([]Instrument, error)
}

// InstrumentSuggestionsSource supplies a small, immutable offline starter list.
type InstrumentSuggestionsSource interface {
	SuggestedInstruments() []Instrument
}

type Registry struct {
	mu      sync.RWMutex
	sources map[DataSourceID]CandleSource
}

func NewRegistry(sources map[DataSourceID]CandleSource) *Registry {
	copySources := make(map[DataSourceID]CandleSource, len(sources))
	for id, source := range sources {
		copySources[id] = source
	}
	return &Registry{sources: copySources}
}

func (r *Registry) Source(id DataSourceID) (CandleSource, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	source, ok := r.sources[id]
	return source, ok
}
