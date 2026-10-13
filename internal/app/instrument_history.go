package app

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	"trajon/internal/market"
)

const instrumentHistoryLimit = 12

// All methods are called by the UI owner. Only the writer goroutine accesses
// err, and Close waits for it before reading the final write error.
type instrumentHistoryStore struct {
	path    string
	updates chan []instrument
	done    chan struct{}
	closed  bool
	err     error
}

func instrumentHistoryPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "trajon", "instrument-history.json"), nil
}

func loadInstrumentHistory(path string) ([]instrument, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var items []instrument
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return normalizeInstrumentHistory(items), nil
}

func normalizeInstrumentHistory(items []instrument) []instrument {
	history := make([]instrument, 0, instrumentHistoryLimit)
	seen := make(map[string]bool)
	for _, item := range items {
		if item.SourceID == "" || item.ProviderSymbol == "" || item.Symbol == "" ||
			(item.MarketType != market.MarketSpot && item.MarketType != market.MarketFutures) || seen[item.Identity()] {
			continue
		}
		if item.Name == "" {
			item.Name = item.Symbol
		}
		seen[item.Identity()] = true
		history = append(history, item)
		if len(history) == instrumentHistoryLimit {
			break
		}
	}
	return history
}

func newInstrumentHistoryStore(path string) *instrumentHistoryStore {
	store := &instrumentHistoryStore{path: path, updates: make(chan []instrument, 1), done: make(chan struct{})}
	go func() {
		defer close(store.done)
		for items := range store.updates {
			store.err = saveInstrumentHistory(store.path, items)
			if store.err != nil {
				log.Printf("save instrument history: %v", store.err)
			}
		}
	}()
	return store
}

// Save coalesces queued snapshots while a write is in progress. The worker
// never reads application state or blocks the UI on filesystem operations.
func (s *instrumentHistoryStore) Save(items []instrument) {
	if s == nil || s.closed {
		return
	}
	items = append([]instrument(nil), items...)
	select {
	case s.updates <- items:
		return
	default:
	}
	select {
	case <-s.updates:
	default:
	}
	s.updates <- items
}

func (s *instrumentHistoryStore) Close() error {
	if s == nil {
		return nil
	}
	if !s.closed {
		s.closed = true
		close(s.updates)
	}
	<-s.done
	return s.err
}

func saveInstrumentHistory(path string, items []instrument) error {
	data, err := json.Marshal(normalizeInstrumentHistory(items))
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".instrument-history-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func (a *application) rememberInstrument(item instrument) {
	items := make([]instrument, 0, len(a.instrumentHistory)+1)
	items = append(items, item)
	items = append(items, a.instrumentHistory...)
	a.instrumentHistory = normalizeInstrumentHistory(items)
	a.instrumentHistoryStore.Save(a.instrumentHistory)
}
