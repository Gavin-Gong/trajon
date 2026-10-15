package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTimeframeFavoritesLoadAndNormalize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	favorites, err := loadTimeframeFavorites(path)
	if err != nil || favorites != nil {
		t.Fatalf("missing favorites = %v, %v", favorites, err)
	}
	for _, tc := range []struct {
		name, data string
		want       []string
		wantError  bool
	}{
		{"invalid", "{", nil, true},
		{"wrong type", `{ "favorites": [] }`, nil, true},
		{"null defaults", "null", nil, false},
		{"explicitly empty", "[]", []string{}, false},
		{"normalized", `["1M", "1m", "15m", "1m", "6h", "12h", "3D", "unknown", "1m "]`, []string{"1m", "15m", "1M"}, false},
		{"removed intervals only", `["6h", "12h", "3D"]`, []string{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tc.data), 0600); err != nil {
				t.Fatal(err)
			}
			favorites, err := loadTimeframeFavorites(path)
			if (err != nil) != tc.wantError || !reflect.DeepEqual(favorites, tc.want) {
				t.Fatalf("favorites = %#v, %v; want %#v, error=%v", favorites, err, tc.want, tc.wantError)
			}
		})
	}
}

func TestTimeframeFavoritesPersistLatestSnapshotAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trajon", "timeframe-favorites.json")
	store := newTimeframeFavoritesStore(path, nil)
	t.Cleanup(func() { store.Close() })
	app := &application{timeframeStore: store}
	for range 31 {
		app.toggleTimeframeFavorite("5m")
		app.toggleTimeframeFavorite("1M")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadTimeframeFavorites(path)
	if err != nil || !reflect.DeepEqual(reloaded, app.favoriteTimeframes()) {
		t.Fatalf("restarted favorites = %v, %v, want %v", reloaded, err, app.favoriteTimeframes())
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".timeframe-favorites-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary files = %v, %v", files, err)
	}
	// Closing drains the final empty selection, rather than restoring defaults.
	store = newTimeframeFavoritesStore(path, nil)
	store.Save([]string{})
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reloaded, err = loadTimeframeFavorites(path)
	if err != nil || reloaded == nil || len(reloaded) != 0 {
		t.Fatalf("restarted empty favorites = %#v, %v", reloaded, err)
	}
	store.Save([]string{"1m"}) // A late save after shutdown is ignored.
}

func TestTimeframeFavoritesStoreOwnsSnapshotsAndReportsFailures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	store := newTimeframeFavoritesStore(path, nil)
	favorites := []string{"5m"}
	store.Save(favorites)
	favorites[0] = "1M"
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadTimeframeFavorites(path)
	if err != nil || !reflect.DeepEqual(reloaded, []string{"5m"}) {
		t.Fatalf("writer observed shared state: %v, %v", reloaded, err)
	}
	var resultError error
	var resultFavorites []string
	store = newTimeframeFavoritesStore(filepath.Join(path, "child.json"), func(favorites []string, err error) {
		resultFavorites, resultError = favorites, err
	})
	store.Save([]string{"1W"})
	if err := store.Close(); err == nil || resultError == nil || !reflect.DeepEqual(resultFavorites, []string{"1W"}) {
		t.Fatalf("unwritable path did not report failed snapshot: %v, %v, %v", err, resultError, resultFavorites)
	}
}
