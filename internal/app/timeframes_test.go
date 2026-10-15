package app

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"trajon/internal/locales"
	"trajon/internal/market"
)

func TestTimeframeStarsOnlyChangeQuickChoices(t *testing.T) {
	app, tt := newKlineToolbarTester()
	clickReplaySetting(t, tt, app.text().Timeframes())
	clickReplaySetting(t, tt, app.text().FavoriteTimeframe("3m"))
	if app.chart.hover != -1 {
		t.Fatal("pointer over the timeframe list leaked into chart hover")
	}
	if app.chart.timeframe != "1h" || !app.timeframePickerOpen {
		t.Fatal("starring switched the chart or closed the list")
	}
	if want := []string{"3m", "15m", "1h", "4h", "1D"}; !reflect.DeepEqual(app.favoriteTimeframes(), want) {
		t.Fatalf("favorites = %v, want %v", app.favoriteTimeframes(), want)
	}
	if _, ok := tt.Find(app.text().UnfavoriteTimeframe("3m")); !ok {
		t.Fatal("star state did not update")
	}
	clickReplaySetting(t, tt, app.text().SelectTimeframe("3m"))
	if app.chart.timeframe != "3m" || app.timeframePickerOpen {
		t.Fatal("selecting a row did not switch the chart and dismiss the list")
	}
	if _, ok := tt.Find("3m"); !ok {
		t.Fatal("starred interval is missing from the quick choices")
	}
	clickReplaySetting(t, tt, "4h")
	clickReplaySetting(t, tt, app.text().Timeframes())
	app.timeframePickerList.ScrollTo(0, ui.Start)
	tt.Frame()
	clickReplaySetting(t, tt, app.text().UnfavoriteTimeframe("3m"))
	if !app.timeframePickerOpen || app.chart.timeframe != "4h" {
		t.Fatal("unstarring closed the list or switched the chart")
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := tt.Find("3m"); ok {
		t.Fatalf("unstarred interval remained in the quick choices: open=%v favorites=%v texts=%v", app.timeframePickerOpen, app.favoriteTimeframes(), tt.Texts())
	}
}

func TestTimeframeListKeyboardDismissalAndReplayLock(t *testing.T) {
	app, tt := newKlineToolbarTester()
	for range 3 {
		clickReplaySetting(t, tt, app.text().Timeframes())
		tt.Key(0, ui.KeyEscape)
		tt.Frame()
		if app.timeframePickerOpen || !tt.Focused(app.text().Timeframes()) {
			t.Fatalf("Escape did not close the list and restore trigger focus: open=%v focused=%v texts=%v", app.timeframePickerOpen, tt.Focused(app.text().Timeframes()), tt.Texts())
		}
	}
	clickReplaySetting(t, tt, app.text().Timeframes())
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if app.chart.timeframe != "2h" || app.timeframePickerOpen {
		t.Fatalf("keyboard selection left period=%q, open=%v", app.chart.timeframe, app.timeframePickerOpen)
	}
	period, periodOK := tt.Find("2h")
	trigger, triggerOK := tt.Find(app.text().Timeframes())
	if !periodOK || !triggerOK || period.X < trigger.X || period.X+period.W > trigger.X+trigger.W {
		t.Fatal("unstarred current interval is missing from the combined dropdown button")
	}
	if !reflect.DeepEqual(app.favoriteTimeframes(), defaultTimeframeFavorites()) {
		t.Fatal("selecting an interval implicitly starred it")
	}
	clickReplaySetting(t, tt, app.text().Timeframes())
	g := currentKlineGeometry(app)
	tt.ClickAt(g.left+g.width*.8, g.top+g.priceHeight*.7)
	if app.timeframePickerOpen || !tt.Focused(app.text().CandleChart(app.displayInstrument().Symbol)) {
		t.Fatal("outside press did not dismiss the list while preserving chart focus")
	}
	app.chart.selectReplayStart(12, len(app.candles))
	tt.Frame()
	clickReplaySetting(t, tt, app.text().Timeframes())
	app.selectTimeframe("1W")
	if app.timeframePickerOpen || app.chart.timeframe != "2h" {
		t.Fatal("replay lock did not prevent timeframe selection")
	}
	app.chart.replayComplete = true
	tt.Frame()
	clickReplaySetting(t, tt, app.text().Timeframes())
	if !app.timeframePickerOpen {
		t.Fatal("completed replay did not unlock the dropdown")
	}
}

func TestTimeframeUnstarredCurrentUsesCombinedDropdown(t *testing.T) {
	for _, language := range []locales.Language{locales.English, locales.Chinese} {
		t.Run(string(language), func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			app.applyLocale(string(language))
			tt.Frame()
			original, _ := tt.Find(app.text().Timeframes())
			clickReplaySetting(t, tt, app.text().Timeframes())
			clickReplaySetting(t, tt, app.text().SelectTimeframe("2h"))
			trigger, _ := tt.Find(app.text().Timeframes())
			period, shown := tt.Find("2h")
			if !shown || trigger.W <= original.W || period.X < trigger.X || period.X+period.W >= trigger.X+trigger.W {
				t.Fatalf("unstarred current interval is not inside the wider trigger: trigger=%+v period=%+v", trigger, period)
			}
			if !tt.Focused(app.text().Timeframes()) || !reflect.DeepEqual(app.favoriteTimeframes(), defaultTimeframeFavorites()) {
				t.Fatal("selection lost trigger focus or implicitly starred the interval")
			}
			// Both halves activate the same button, and its focus identity survives
			// favorites moving the label between that button and the quick choices.
			clickReplaySetting(t, tt, "2h")
			if !app.timeframePickerOpen {
				t.Fatal("clicking the period did not open the dropdown")
			}
			clickReplaySetting(t, tt, app.text().FavoriteTimeframe("2h"))
			starredTrigger, _ := tt.Find(app.text().Timeframes())
			quick, _ := tt.Find("2h")
			if starredTrigger.W != original.W || quick.X+quick.W > starredTrigger.X {
				t.Fatal("starring the current interval did not restore the icon-only trigger")
			}
			clickReplaySetting(t, tt, app.text().UnfavoriteTimeframe("2h"))
			if trigger, _ = tt.Find(app.text().Timeframes()); trigger.W <= original.W || !app.timeframePickerOpen {
				t.Fatal("unstarring the current interval did not restore the combined trigger while keeping the list open")
			}
			tt.Key(0, ui.KeyEscape)
			if !tt.Focused(app.text().Timeframes()) {
				t.Fatal("trigger resizing lost dismissal focus")
			}
			trigger, _ = tt.Find(app.text().Timeframes())
			tt.ClickAt(trigger.X+trigger.W-8, trigger.Y+trigger.H/2)
			if !app.timeframePickerOpen {
				t.Fatal("clicking the arrow did not open the combined dropdown")
			}
			tt.Key(0, ui.KeyEscape)
			clickReplaySetting(t, tt, "1h")
			if trigger, _ = tt.Find(app.text().Timeframes()); trigger.W != original.W {
				t.Fatal("choosing a starred interval left the combined trigger visible")
			}
			if _, shown := tt.Find("2h"); shown {
				t.Fatal("the previous unstarred interval remained in the toolbar")
			}
		})
	}
}

func TestTimeframeCompactListUsesShadingWithoutCheckmark(t *testing.T) {
	app, tt := newKlineToolbarTester()
	clickReplaySetting(t, tt, app.text().Timeframes())
	choice, _ := tt.Find(app.text().SelectTimeframe("1h"))
	star, _ := tt.Find(app.text().UnfavoriteTimeframe("1h"))
	trigger, _ := tt.Find(app.text().Timeframes())
	if span := star.X + star.W - choice.X; span > 116 || star.X-(choice.X+choice.W) != 2 {
		t.Fatalf("list is not compact: choice=%+v star=%+v", choice, star)
	}
	if inset := trigger.X + 132 - (star.X + star.W); inset < 10 {
		t.Fatalf("star button is not inset from the panel edge: %v", inset)
	}
	img := tt.Image()
	selected := img.RGBAAt(int(choice.X+choice.W-16), int(choice.Y+choice.H/2))
	other, _ := tt.Find(app.text().SelectTimeframe("2h"))
	unselected := img.RGBAAt(int(other.X+other.W-16), int(other.Y+other.H/2))
	if selected == unselected {
		t.Fatal("the selected interval has no shaded background")
	}
	// The former checkmark area contains only the uniform selection shading.
	for x := int(choice.X + choice.W - 22); x < int(choice.X+choice.W-7); x++ {
		for y := int(choice.Y+choice.H/2) - 7; y < int(choice.Y+choice.H/2)+7; y++ {
			if got := img.RGBAAt(x, y); got != selected {
				t.Fatalf("selection contains a mark instead of just shading at (%d,%d): %v, want %v", x, y, got, selected)
			}
		}
	}
}

func TestTimeframeSelectionHasOneBackgroundAcrossTextAndStar(t *testing.T) {
	for _, period := range []string{"1h", "5m"} {
		t.Run(period, func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			app.selectTimeframe(period)
			tt.Frame()
			clickReplaySetting(t, tt, app.text().Timeframes())
			choice, _ := tt.Find(app.text().SelectTimeframe(period))
			starLabel := app.text().FavoriteTimeframe(period)
			if period == "1h" {
				starLabel = app.text().UnfavoriteTimeframe(period)
			}
			star, _ := tt.Find(starLabel)
			y := choice.Y + choice.H/2
			expected := tt.Image().RGBAAt(int(choice.X+choice.W+1), int(y))
			for _, stage := range []string{"idle", "text hover", "star hover", "keyboard cursor moved"} {
				switch stage {
				case "text hover":
					tt.Move(choice.X+choice.W/2, y)
				case "star hover":
					tt.Move(star.X+star.W/2, y)
				case "keyboard cursor moved":
					tt.Move(10, 450)
					tt.Key(0, ui.KeyDown)
				}
				img := tt.Image()
				background := img.RGBAAt(int(choice.X+choice.W+1), int(y))
				if background != expected {
					t.Fatalf("%s: current interval lost its single row background: %v, want %v", stage, background, expected)
				}
				for _, x := range []float32{choice.X + choice.W - 16, star.X + 4} {
					if got := img.RGBAAt(int(x), int(y)); got != background {
						t.Fatalf("%s: selection has nested backgrounds at x=%v: %v, want whole-row background %v", stage, x, got, background)
					}
				}
			}
		})
	}
}

func TestTimeframeFavoritesSurviveEmptySelectionAndSeriesChanges(t *testing.T) {
	app, tt := newKlineToolbarTester()
	for _, period := range defaultTimeframeFavorites() {
		app.toggleTimeframeFavorite(period)
	}
	tt.Frame()
	if app.timeframeFavorites == nil || len(app.favoriteTimeframes()) != 0 {
		t.Fatal("removing all stars restored defaults")
	}
	clickReplaySetting(t, tt, app.text().Timeframes())
	app.timeframePickerList.ScrollToEnd()
	tt.Frame()
	clickReplaySetting(t, tt, app.text().FavoriteTimeframe("1W"))
	clickReplaySetting(t, tt, app.text().SelectTimeframe("1W"))
	app.applyMarketSnapshot(market.Snapshot{Instrument: defaultInstrument(), Interval: "1W", Candles: demoCandles()})
	app.chart.selectReplayStart(12, len(app.candles))
	app.chart.resetReplay()
	app.applyMarketSnapshot(market.Snapshot{Instrument: historyETH(), Interval: "1M", Candles: demoCandles()})
	if want := []string{"1W"}; !reflect.DeepEqual(app.favoriteTimeframes(), want) {
		t.Fatalf("series/replay changes lost favorites: %v", app.favoriteTimeframes())
	}
}

func TestTimeframeDropdownAcrossTitleHostsAndLocales(t *testing.T) {
	for _, language := range []locales.Language{locales.English, locales.Chinese} {
		for _, titleFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/title-first=%v", language, titleFirst), func(t *testing.T) {
				app, tt := newApplicationShellOrderTester(language, titleFirst)
				for range 2 {
					clickReplaySetting(t, tt, app.text().Replay())
					clickReplaySetting(t, tt, app.text().Timeframes())
					clickReplaySetting(t, tt, app.text().SelectTimeframe("2h"))
					clickReplaySetting(t, tt, "2h")
					clickReplaySetting(t, tt, app.text().FavoriteTimeframe("5m"))
					app.applyLocale(string(map[locales.Language]locales.Language{locales.English: locales.Chinese, locales.Chinese: locales.English}[app.language]))
					tt.Frame()
					clickReplaySetting(t, tt, app.text().UnfavoriteTimeframe("5m"))
					clickReplaySetting(t, tt, app.text().Overview())
					if app.timeframePickerOpen {
						t.Fatal("leaving the market page did not dismiss the list")
					}
				}
			})
		}
	}
}

func TestTimeframeFavoritesFitNarrowToolbar(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.timeframeFavorites = market.CandleIntervals()
	tt.SetSize(520, 520)
	tt.Frame()
	arrow, arrowOK := tt.Find(app.text().Timeframes())
	indicators, indicatorsOK := tt.Find(app.text().Indicators())
	toolbar, _ := tt.Find(app.text().ChartToolbar())
	if !arrowOK || !indicatorsOK || arrow.X+arrow.W > indicators.X || indicators.X+indicators.W > toolbar.X+toolbar.W {
		t.Fatalf("narrow toolbar clipped controls: arrow=%+v indicators=%+v toolbar=%+v", arrow, indicators, toolbar)
	}
	clickReplaySetting(t, tt, app.text().Timeframes())
	app.timeframePickerList.ScrollToEnd()
	tt.Frame()
	clickReplaySetting(t, tt, app.text().SelectTimeframe("1M"))
	if app.chart.timeframe != "1M" {
		t.Fatal("last interval was unreachable in the scrollable list")
	}
	app.toggleTimeframeFavorite("1M")
	tt.Frame()
	for _, empty := range []bool{false, true} {
		if empty {
			app.timeframeFavorites = []string{}
			tt.Frame()
		}
		trigger, _ := tt.Find(app.text().Timeframes())
		indicators, _ := tt.Find(app.text().Indicators())
		if trigger.W != 58 || trigger.X+trigger.W > indicators.X || indicators.X+indicators.W > toolbar.X+toolbar.W {
			t.Fatalf("combined trigger clipped narrow toolbar controls (empty=%v): trigger=%+v indicators=%+v", empty, trigger, indicators)
		}
	}
}

func TestTimeframeFailedSaveRetainsChoicesAndRejectsStaleResults(t *testing.T) {
	app, tt := newKlineToolbarTester()
	previous := app.favoriteTimeframes()
	app.toggleTimeframeFavorite("5m")
	app.applyTimeframeSaveResult(previous, errors.New("old failure"))
	if app.timeframeSaveError != nil {
		t.Fatal("stale save failure replaced current state")
	}
	app.applyTimeframeSaveResult(app.favoriteTimeframes(), errors.New("disk unavailable"))
	tt.Frame()
	clickReplaySetting(t, tt, app.text().Timeframes())
	if !tt.HasText(app.text().TimeframeSaveFailed()) || !tt.HasText(app.text().Retry()) {
		t.Fatalf("failed save did not offer feedback and retry: error=%v texts=%v", app.timeframeSaveError, tt.Texts())
	}
	if !reflect.DeepEqual(app.favoriteTimeframes(), []string{"5m", "15m", "1h", "4h", "1D"}) {
		t.Fatal("save failure discarded the current choices")
	}
}

// Opt-in snapshots stay outside the repository and are useful for visual review.
func TestTimeframeVisualSnapshots(t *testing.T) {
	dir := os.Getenv("TRAJON_UI_SNAPSHOTS")
	if dir == "" {
		t.Skip("set TRAJON_UI_SNAPSHOTS to capture timeframe states")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	app, tt := newKlineToolbarTester()
	capture := func(name string) {
		t.Helper()
		file, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if err := png.Encode(file, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
	capture("toolbar")
	clickReplaySetting(t, tt, app.text().Timeframes())
	capture("list")
	clickReplaySetting(t, tt, app.text().FavoriteTimeframe("3m"))
	capture("starred")
	clickReplaySetting(t, tt, app.text().SelectTimeframe("5m"))
	capture("unstarred-toolbar")
	clickReplaySetting(t, tt, "5m")
	capture("unstarred-list")
	app.timeframeFavorites = market.CandleIntervals()
	tt.SetSize(520, 520)
	tt.Frame()
	capture("narrow")
	app.toggleTimeframeFavorite("5m")
	tt.Frame()
	capture("narrow-unstarred")
}
