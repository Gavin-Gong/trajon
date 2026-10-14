package market

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

const instrumentCatalogTTL = 5 * time.Minute

type instrumentCatalogKey struct {
	source DataSourceID
	market MarketType
}

type cachedInstrumentCatalog struct {
	items []Instrument
	at    time.Time
}

// InstrumentSearcher caches each market independently, so a failed market can
// be retried without downloading successful catalogs again.
type InstrumentSearcher struct {
	registry *Registry
	mu       sync.Mutex
	cache    map[instrumentCatalogKey]cachedInstrumentCatalog
}

func NewInstrumentSearcher(registry *Registry) *InstrumentSearcher {
	return &InstrumentSearcher{registry: registry, cache: make(map[instrumentCatalogKey]cachedInstrumentCatalog)}
}

func (s *InstrumentSearcher) Search(ctx context.Context, query string) ([]Instrument, error) {
	if strings.TrimSpace(query) == "" {
		return InstrumentCatalog(), nil
	}
	type result struct {
		items []Instrument
		err   error
	}
	sources := s.instrumentSources()
	ids := make([]DataSourceID, 0, len(sources))
	for id := range sources {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) == 0 {
		return nil, ErrNoInstrumentSource
	}

	results := make([]result, len(ids)*2)
	var jobs sync.WaitGroup
	for i, id := range ids {
		for j, marketType := range []MarketType{MarketSpot, MarketFutures} {
			index := i*2 + j
			jobs.Go(func() {
				items, err := s.loadCatalog(ctx, instrumentCatalogKey{id, marketType}, sources[id])
				if err != nil {
					err = &InstrumentSearchError{SourceID: id, MarketType: marketType, Err: err}
				}
				results[index] = result{items, err}
			})
		}
	}
	jobs.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var items []Instrument
	var errs []error
	seen := make(map[string]bool)
	for _, result := range results {
		for _, item := range FilterInstruments(result.items, query, "") {
			if !seen[item.Identity()] {
				seen[item.Identity()] = true
				items = append(items, item)
			}
		}
		if result.err != nil {
			errs = append(errs, result.err)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Symbol != items[j].Symbol {
			return items[i].Symbol < items[j].Symbol
		}
		if items[i].SourceID != items[j].SourceID {
			return items[i].SourceID < items[j].SourceID
		}
		return items[i].MarketType == MarketSpot && items[j].MarketType != MarketSpot
	})
	return items, errors.Join(errs...)
}

func (s *InstrumentSearcher) instrumentSources() map[DataSourceID]InstrumentSource {
	sources := make(map[DataSourceID]InstrumentSource)
	if s.registry == nil {
		return sources
	}
	s.registry.mu.RLock()
	defer s.registry.mu.RUnlock()
	for id, source := range s.registry.sources {
		if source, ok := source.(InstrumentSource); ok {
			sources[id] = source
		}
	}
	return sources
}

func (s *InstrumentSearcher) loadCatalog(ctx context.Context, key instrumentCatalogKey, source InstrumentSource) ([]Instrument, error) {
	s.mu.Lock()
	cached, ok := s.cache[key]
	s.mu.Unlock()
	if ok && time.Since(cached.at) < instrumentCatalogTTL {
		return append([]Instrument(nil), cached.items...), nil
	}
	items, err := source.LoadInstruments(ctx, key.market)
	if err == nil && ctx.Err() == nil {
		s.mu.Lock()
		s.cache[key] = cachedInstrumentCatalog{items: append([]Instrument(nil), items...), at: time.Now()}
		s.mu.Unlock()
	}
	return items, err
}
