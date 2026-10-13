package app

import (
	"errors"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func waitInstrumentUI(t *testing.T, tt *ui.Tester, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !ready() && time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(time.Millisecond)
	}
	if !ready() {
		t.Fatal("instrument UI did not reach expected state")
	}
	tt.Frame()
}

func TestInstrumentSearchRejectsStaleClosedAndChangedQueryResults(t *testing.T) {
	item := defaultInstrument()
	app := &application{instrumentPickerOpen: true, instrumentPickerQuery: "btc"}
	app.instrumentSearch.generation = 2
	app.instrumentSearch.loading = true
	result := instrumentSearchResult{generation: 1, query: "btc", items: []instrument{item}}
	app.applyInstrumentSearchResult(result)
	if !app.instrumentSearch.loading || len(app.instrumentSearch.items) != 0 {
		t.Fatal("stale generation changed search state")
	}
	result.generation, result.query = 2, "eth"
	app.applyInstrumentSearchResult(result)
	if !app.instrumentSearch.loading {
		t.Fatal("stale query changed search state")
	}
	result.query = "btc"
	result.err = errors.New("one market offline")
	app.applyInstrumentSearchResult(result)
	if app.instrumentSearch.loading || len(app.instrumentSearch.items) != 1 || app.instrumentSearch.err == nil {
		t.Fatal("current partial results were not applied")
	}
	result.items = nil
	app.instrumentPickerOpen = false
	app.applyInstrumentSearchResult(result)
	if len(app.instrumentSearch.items) != 1 {
		t.Fatal("closed picker accepted results")
	}
	app.instrumentPickerOpen = true
	app.instrumentPickerQuery = ""
	app.searchInstruments()
	if app.instrumentSearch.loading || app.instrumentSearch.err != nil || len(app.instrumentSearch.items) != 0 || len(app.filteredInstruments()) != 4 {
		t.Fatal("clearing search did not restore defaults")
	}
}

func TestInstrumentPickerDismissalCancelsDebouncedSearchAndRestoresFocus(t *testing.T) {
	app := &application{activePage: "market", currentInstrument: defaultInstrument(), chart: newKlineState(0), marketLoadStarted: true}
	tt := ui.NewTester(func(c *ui.Context) { app.content(c) }, 1000, 700)
	if err := tt.Click("BTC/USDT"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("搜索标的"); err != nil {
		t.Fatal(err)
	}
	tt.Type("eth")
	if app.instrumentSearch.cancel == nil || !app.instrumentSearch.loading {
		t.Fatal("typing did not schedule a search")
	}
	// Click close before debounce completes, then allow extra build passes.
	if err := tt.Click("关闭标的选择"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if app.instrumentPickerOpen || app.instrumentSearch.cancel != nil || app.instrumentSearch.loading {
		t.Fatal("closing picker did not cancel search")
	}
	if err := tt.Click("BTC/USDT"); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused("搜索标的") || app.instrumentPickerQuery != "" || len(app.filteredInstruments()) != 4 {
		t.Fatal("reopening did not restore search focus and default list")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if app.instrumentPickerOpen {
		t.Fatal("Escape did not dismiss picker")
	}
}
