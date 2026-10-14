package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"trajon/internal/locales"
)

func localePreferencePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "trajon", "locale.json"), nil
}

func loadLocalePreference(path string) (locales.Preference, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return locales.System, nil
	}
	if err != nil {
		return locales.System, err
	}
	var config struct {
		Language locales.Preference `json:"language"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return locales.System, err
	}
	if !config.Language.Valid() {
		return locales.System, fmt.Errorf("unsupported language preference %q", config.Language)
	}
	return config.Language, nil
}

func saveLocalePreference(path string, preference locales.Preference) error {
	if !preference.Valid() {
		return fmt.Errorf("unsupported language preference %q", preference)
	}
	data, err := json.Marshal(struct {
		Language locales.Preference `json:"language"`
	}{preference})
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".locale-*")
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

// Save and Close belong to the UI owner. The worker owns filesystem I/O and err;
// Close drains the latest preference and joins the worker before returning err.
type localePreferenceStore struct {
	updates chan locales.Preference
	done    chan struct{}
	closed  bool
	err     error
}

func newLocalePreferenceStore(path string, onResult func(locales.Preference, error)) *localePreferenceStore {
	store := &localePreferenceStore{updates: make(chan locales.Preference, 1), done: make(chan struct{})}
	go func() {
		defer close(store.done)
		for preference := range store.updates {
			store.err = saveLocalePreference(path, preference)
			if onResult != nil {
				onResult(preference, store.err)
			}
		}
	}()
	return store
}

// Save coalesces queued choices without blocking the UI on disk writes.
func (s *localePreferenceStore) Save(preference locales.Preference) {
	if s == nil || s.closed || !preference.Valid() {
		return
	}
	select {
	case s.updates <- preference:
		return
	default:
	}
	select {
	case <-s.updates:
	default:
	}
	s.updates <- preference
}

func (s *localePreferenceStore) Close() error {
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
