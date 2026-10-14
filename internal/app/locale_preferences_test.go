package app

import (
	"os"
	"path/filepath"
	"testing"

	"trajon/internal/locales"
)

func TestLocalePreferenceFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trajon", "locale.json")
	if got, err := loadLocalePreference(path); err != nil || got != locales.System {
		t.Fatalf("missing file = %q, %v", got, err)
	}
	for _, preference := range []locales.Preference{locales.PreferChinese, locales.PreferEnglish, locales.System} {
		if err := saveLocalePreference(path, preference); err != nil {
			t.Fatal(err)
		}
		if got, err := loadLocalePreference(path); err != nil || got != preference {
			t.Fatalf("round trip = %q, %v, want %q", got, err, preference)
		}
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".locale-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary files left behind: %v, %v", files, err)
	}
	if err := saveLocalePreference(path, locales.Preference("fr")); err == nil {
		t.Fatal("invalid preference was saved")
	}
	if got, err := loadLocalePreference(path); err != nil || got != locales.System {
		t.Fatal("invalid save overwrote the valid file")
	}
}

func TestInvalidLocalePreferenceFallsBackToSystem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "locale.json")
	for _, data := range []string{"{", "null", "{}", `{"language":"fr"}`, `{"language":123}`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if got, err := loadLocalePreference(path); err == nil || got != locales.System {
			t.Fatalf("load(%q) = %q, %v, want System and error", data, got, err)
		}
	}
	if got, err := loadLocalePreference(filepath.Dir(path)); err == nil || got != locales.System {
		t.Fatalf("unreadable preference = %q, %v", got, err)
	}
}

func TestLocalePreferenceStoreDrainsLatestChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "locale.json")
	store := newLocalePreferenceStore(path, nil)
	for range 100 {
		store.Save(locales.PreferChinese)
		store.Save(locales.PreferEnglish)
	}
	store.Save(locales.System)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store.Save(locales.PreferChinese)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if got, err := loadLocalePreference(path); err != nil || got != locales.System {
		t.Fatalf("latest persisted choice = %q, %v", got, err)
	}
}

func TestLocalePreferenceStoreReportsFailureAndCanRetry(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "blocked")
	if err := os.WriteFile(parent, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "locale.json")
	results := make(chan error, 2)
	store := newLocalePreferenceStore(path, func(preference locales.Preference, err error) {
		if preference != locales.PreferEnglish {
			panic("unexpected preference")
		}
		results <- err
	})
	t.Cleanup(func() { _ = store.Close() })
	store.Save(locales.PreferEnglish)
	if err := <-results; err == nil {
		t.Fatal("write failure was not reported")
	}
	if err := os.Remove(parent); err != nil {
		t.Fatal(err)
	}
	store.Save(locales.PreferEnglish)
	if err := <-results; err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if got, err := loadLocalePreference(path); err != nil || got != locales.PreferEnglish {
		t.Fatalf("retry persisted = %q, %v", got, err)
	}
}

func TestLocalePreferenceStoreCloseReturnsWriteFailure(t *testing.T) {
	store := newLocalePreferenceStore(t.TempDir(), nil)
	store.Save(locales.PreferChinese)
	if err := store.Close(); err == nil {
		t.Fatal("Close hid the write error")
	}
}
