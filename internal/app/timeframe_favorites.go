package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"trajon/internal/market"
)

func timeframeFavoritesPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "trajon", "timeframe-favorites.json"), nil
}

// Normalize in interval order, discard unknown values and duplicates, and keep
// an explicit empty slice distinct from nil (the default favorites).
func normalizeTimeframeFavorites(favorites []string) []string {
	result := make([]string, 0, len(favorites))
	for _, period := range market.CandleIntervals() {
		if slices.Contains(favorites, period) {
			result = append(result, period)
		}
	}
	return result
}

func loadTimeframeFavorites(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var favorites []string
	if err := json.Unmarshal(data, &favorites); err != nil {
		return nil, err
	}
	if favorites == nil {
		return nil, nil
	}
	return normalizeTimeframeFavorites(favorites), nil
}

func saveTimeframeFavorites(path string, favorites []string) error {
	data, err := json.Marshal(normalizeTimeframeFavorites(favorites))
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".timeframe-favorites-*")
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

// Save and Close belong to the UI owner. The writer only sees immutable
// snapshots; Close drains the latest queued snapshot before joining it.
// Result callbacks run on the worker and must publish via Window.Update.
type timeframeFavoritesStore struct {
	updates chan []string
	done    chan struct{}
	closed  bool
	err     error
}

func newTimeframeFavoritesStore(path string, onResult func([]string, error)) *timeframeFavoritesStore {
	store := &timeframeFavoritesStore{updates: make(chan []string, 1), done: make(chan struct{})}
	go func() {
		defer close(store.done)
		for favorites := range store.updates {
			store.err = saveTimeframeFavorites(path, favorites)
			if onResult != nil {
				onResult(favorites, store.err)
			}
		}
	}()
	return store
}

func (s *timeframeFavoritesStore) Save(favorites []string) {
	if s == nil || s.closed {
		return
	}
	favorites = slices.Clone(favorites)
	select {
	case s.updates <- favorites:
		return
	default:
	}
	select {
	case <-s.updates:
	default:
	}
	s.updates <- favorites
}

func (s *timeframeFavoritesStore) Close() error {
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
