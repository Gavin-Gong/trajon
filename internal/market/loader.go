package market

import (
	"context"
	"strings"
	"sync"
)

const DefaultCandleLimit = 1000

type Snapshot struct {
	Instrument Instrument
	Interval   string
	Candles    []Candle

	Loading        bool
	Error          error
	Pending        *CandleRequest
	OlderLoading   bool
	OlderError     error
	OlderExhausted bool
	OlderPending   *CandleRequest

	// HistoryPrepended is non-zero only for a snapshot emitted after a
	// successful older-history merge. It tells a view how far to move its
	// viewport to preserve the previously visible candle.
	HistoryPrepended int
}

type ChangeHandler func(Snapshot)

type Loader struct {
	mu       sync.Mutex
	registry *Registry
	cache    map[string][]Candle
	onChange ChangeHandler
	closed   bool

	instrument Instrument
	interval   string
	candles    []Candle

	loading        bool
	loadError      error
	pending        *CandleRequest
	loadCancel     context.CancelFunc
	loadGeneration uint64

	olderLoading        bool
	olderError          error
	olderExhausted      bool
	olderPending        *CandleRequest
	olderLoadCancel     context.CancelFunc
	olderLoadGeneration uint64
}

func NewLoader(registry *Registry, onChange ChangeHandler) *Loader {
	return &Loader{
		registry: registry,
		cache:    make(map[string][]Candle),
		onChange: onChange,
	}
}

func (l *Loader) Select(instrument Instrument, interval string, force bool) {
	if interval == "" {
		interval = "1h"
	}
	request := CandleRequest{Instrument: instrument, Interval: interval, Limit: DefaultCandleLimit}

	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	l.cancelOlderLocked()
	if l.loadCancel != nil {
		l.loadCancel()
		l.loadCancel = nil
	}
	l.loadGeneration++
	generation := l.loadGeneration
	requestCopy := request
	l.pending = &requestCopy
	l.loadError = nil

	if !force {
		if cached, ok := l.cache[request.CacheKey()]; ok {
			l.commitCurrentLocked(request, cached)
			snapshot := l.snapshotLocked(0)
			l.mu.Unlock()
			l.notify(snapshot)
			return
		}
	}

	var source CandleSource
	var ok bool
	if l.registry != nil {
		source, ok = l.registry.Source(instrument.SourceID)
	}
	if !ok || source == nil {
		l.loading = false
		l.loadError = &MissingSourceError{SourceID: instrument.SourceID}
		snapshot := l.snapshotLocked(0)
		l.mu.Unlock()
		l.notify(snapshot)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	l.loadCancel = cancel
	l.loading = true
	snapshot := l.snapshotLocked(0)
	l.mu.Unlock()
	l.notify(snapshot)

	go func() {
		candles, err := source.LoadCandles(ctx, request)
		l.finishLatest(generation, request, candles, err)
	}()
}

// InvalidateSourceCache drops only reusable results for a changed account.
// Already displayed candles and replay state are intentionally left intact.
func (l *Loader) InvalidateSourceCache(source DataSourceID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	prefix := string(source) + "|"
	for key := range l.cache {
		if strings.HasPrefix(key, prefix) {
			delete(l.cache, key)
		}
	}
}

func (l *Loader) RetryLatest() {
	l.mu.Lock()
	if l.pending == nil {
		l.mu.Unlock()
		return
	}
	request := *l.pending
	l.mu.Unlock()
	l.Select(request.Instrument, request.Interval, true)
}

func (l *Loader) LoadOlder() {
	l.mu.Lock()
	if l.closed || l.loading || l.olderLoading || l.olderExhausted || len(l.candles) == 0 || l.instrument.SourceID == "" {
		l.mu.Unlock()
		return
	}
	before := l.candles[0].Time
	request := CandleRequest{
		Instrument: l.instrument,
		Interval:   l.interval,
		Limit:      DefaultCandleLimit,
		Before:     before,
	}
	var source CandleSource
	var ok bool
	if l.registry != nil {
		source, ok = l.registry.Source(request.Instrument.SourceID)
	}
	if !ok || source == nil {
		l.olderError = &MissingSourceError{SourceID: request.Instrument.SourceID}
		snapshot := l.snapshotLocked(0)
		l.mu.Unlock()
		l.notify(snapshot)
		return
	}
	if l.olderLoadCancel != nil {
		l.olderLoadCancel()
	}
	l.olderLoadGeneration++
	generation := l.olderLoadGeneration
	requestCopy := request
	l.olderPending = &requestCopy
	l.olderError = nil
	l.olderLoading = true
	ctx, cancel := context.WithCancel(context.Background())
	l.olderLoadCancel = cancel
	snapshot := l.snapshotLocked(0)
	l.mu.Unlock()
	l.notify(snapshot)

	go func() {
		candles, err := source.LoadCandles(ctx, request)
		l.finishOlder(generation, request, candles, err)
	}()
}

func (l *Loader) RetryOlder() {
	l.mu.Lock()
	if l.olderPending == nil {
		l.mu.Unlock()
		return
	}
	l.olderError = nil
	l.mu.Unlock()
	l.LoadOlder()
}

func (l *Loader) Snapshot() Snapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.snapshotLocked(0)
}

func (l *Loader) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	l.closed = true
	if l.loadCancel != nil {
		l.loadCancel()
		l.loadCancel = nil
	}
	if l.olderLoadCancel != nil {
		l.olderLoadCancel()
		l.olderLoadCancel = nil
	}
}

func (l *Loader) finishLatest(generation uint64, request CandleRequest, candles []Candle, err error) {
	l.mu.Lock()
	if l.closed || generation != l.loadGeneration {
		l.mu.Unlock()
		return
	}
	l.loadCancel = nil
	l.loading = false
	if err != nil {
		l.loadError = err
		snapshot := l.snapshotLocked(0)
		l.mu.Unlock()
		l.notify(snapshot)
		return
	}
	if len(candles) == 0 {
		l.loadError = &NoCandlesError{Symbol: request.Instrument.Symbol}
		snapshot := l.snapshotLocked(0)
		l.mu.Unlock()
		l.notify(snapshot)
		return
	}
	cached := CloneCandles(candles)
	l.cache[request.CacheKey()] = cached
	l.commitCurrentLocked(request, cached)
	snapshot := l.snapshotLocked(0)
	l.mu.Unlock()
	l.notify(snapshot)
}

func (l *Loader) finishOlder(generation uint64, request CandleRequest, candles []Candle, err error) {
	l.mu.Lock()
	if l.closed || generation != l.olderLoadGeneration {
		l.mu.Unlock()
		return
	}
	l.olderLoading = false
	l.olderLoadCancel = nil
	if err != nil {
		l.olderError = err
		snapshot := l.snapshotLocked(0)
		l.mu.Unlock()
		l.notify(snapshot)
		return
	}
	if len(candles) == 0 || len(l.candles) == 0 {
		l.olderExhausted = true
		l.olderPending = nil
		snapshot := l.snapshotLocked(0)
		l.mu.Unlock()
		l.notify(snapshot)
		return
	}

	oldest := l.candles[0].Time
	seen := make(map[int64]struct{}, len(candles))
	older := make([]Candle, 0, len(candles))
	for _, candle := range candles {
		timestamp := candle.Time.UnixNano()
		if !candle.Time.Before(oldest) {
			continue
		}
		if _, exists := seen[timestamp]; exists {
			continue
		}
		seen[timestamp] = struct{}{}
		older = append(older, candle)
	}
	if len(older) == 0 {
		l.olderExhausted = true
		l.olderPending = nil
		snapshot := l.snapshotLocked(0)
		l.mu.Unlock()
		l.notify(snapshot)
		return
	}

	merged := make([]Candle, 0, len(older)+len(l.candles))
	merged = append(merged, older...)
	merged = append(merged, l.candles...)
	l.candles = merged
	AddEMA20(l.candles)
	l.olderError = nil
	l.olderPending = nil
	snapshot := l.snapshotLocked(len(older))
	l.mu.Unlock()
	l.notify(snapshot)
}

func (l *Loader) cancelOlderLocked() {
	if l.olderLoadCancel != nil {
		l.olderLoadCancel()
		l.olderLoadCancel = nil
	}
	l.olderLoadGeneration++
	l.olderLoading = false
	l.olderExhausted = false
	l.olderPending = nil
	l.olderError = nil
}

func (l *Loader) commitCurrentLocked(request CandleRequest, candles []Candle) {
	l.instrument = request.Instrument
	l.interval = request.Interval
	l.candles = CloneCandles(candles)
	l.loading = false
	l.loadError = nil
	l.pending = nil
	l.olderLoading = false
	l.olderError = nil
	l.olderExhausted = false
	l.olderPending = nil
}

func (l *Loader) snapshotLocked(historyPrepended int) Snapshot {
	var pending *CandleRequest
	if l.pending != nil {
		copyRequest := *l.pending
		pending = &copyRequest
	}
	var olderPending *CandleRequest
	if l.olderPending != nil {
		copyRequest := *l.olderPending
		olderPending = &copyRequest
	}
	return Snapshot{
		Instrument:       l.instrument,
		Interval:         l.interval,
		Candles:          CloneCandles(l.candles),
		Loading:          l.loading,
		Error:            l.loadError,
		Pending:          pending,
		OlderLoading:     l.olderLoading,
		OlderError:       l.olderError,
		OlderExhausted:   l.olderExhausted,
		OlderPending:     olderPending,
		HistoryPrepended: historyPrepended,
	}
}

func (l *Loader) notify(snapshot Snapshot) {
	if l.onChange != nil {
		l.onChange(snapshot)
	}
}
