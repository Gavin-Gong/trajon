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
// be retried without downloading successful catalogs again. Query-based sources
// resolve their own matches instead of pretending to publish a whole catalog.
type InstrumentSearcher struct {
	registry *Registry
	mu       sync.Mutex
	cache    map[instrumentCatalogKey]cachedInstrumentCatalog
}

func NewInstrumentSearcher(registry *Registry) *InstrumentSearcher {
	return &InstrumentSearcher{registry: registry, cache: make(map[instrumentCatalogKey]cachedInstrumentCatalog)}
}

func (s *InstrumentSearcher) Search(ctx context.Context, query string) ([]Instrument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(query) == "" {
		return s.registry.DefaultInstruments(), nil
	}
	type job struct {
		key     instrumentCatalogKey
		catalog InstrumentSource
		query   InstrumentQuerySource
	}
	sources := s.instrumentSources()
	ids := make([]DataSourceID, 0, len(sources))
	for id := range sources {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var work []job
	for _, id := range ids {
		source := sources[id]
		if query, ok := source.(InstrumentQuerySource); ok {
			work = append(work, job{key: instrumentCatalogKey{source: id}, query: query})
			continue
		}
		if catalog, ok := source.(InstrumentSource); ok {
			for _, kind := range []MarketType{MarketSpot, MarketFutures} {
				work = append(work, job{key: instrumentCatalogKey{id, kind}, catalog: catalog})
			}
		}
	}
	if len(work) == 0 {
		return nil, ErrNoInstrumentSource
	}
	type result struct {
		items []Instrument
		err   error
	}
	results := make([]result, len(work))
	var jobs sync.WaitGroup
	for index, work := range work {
		jobs.Go(func() {
			var items []Instrument
			var err error
			if work.query != nil {
				items, err = work.query.SearchInstruments(ctx, query)
			} else {
				items, err = s.loadCatalog(ctx, work.key, work.catalog)
				items = FilterInstruments(items, query, "")
			}
			if err != nil {
				err = &InstrumentSearchError{SourceID: work.key.source, MarketType: work.key.market, Err: err}
			}
			results[index] = result{items, err}
		})
	}
	jobs.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var items []Instrument
	var errs []error
	seen := make(map[string]bool)
	for _, result := range results {
		for _, item := range result.items {
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

func (s *InstrumentSearcher) instrumentSources() map[DataSourceID]CandleSource {
	sources := make(map[DataSourceID]CandleSource)
	if s.registry == nil {
		return sources
	}
	s.registry.mu.RLock()
	defer s.registry.mu.RUnlock()
	for id, source := range s.registry.sources {
		sources[id] = source
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
