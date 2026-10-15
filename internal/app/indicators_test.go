package app

import (
	"bytes"
	"context"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"trajon/internal/locales"
	"trajon/internal/market"
)

func TestIndicatorDialogEditsMultipleEMAs(t *testing.T) {
	app, tt := newReplaySettingsTester()
	clickReplaySetting(t, tt, "指标")
	clickReplaySetting(t, tt, "EMA")
	clickReplaySetting(t, tt, "添加 EMA")
	clickReplaySetting(t, tt, "添加 EMA")
	clickReplaySetting(t, tt, "EMA 1 周期")
	replaceReplaySetting(tt, "9")
	clickReplaySetting(t, tt, "EMA 2 周期")
	replaceReplaySetting(tt, "55")
	clickReplaySetting(t, tt, "EMA 2 线宽")
	replaceReplaySetting(tt, "3")
	clickReplaySetting(t, tt, "EMA 3") // Hide only the third line.
	if len(app.chart.indicators.EMALines) != 1 || app.chart.indicators.EMALines[0].Period != 20 {
		t.Fatal("draft edits leaked into saved settings")
	}
	clickReplaySetting(t, tt, "EMA 2 颜色")
	if !tt.HasText("色相") {
		t.Fatal("localized color picker did not open")
	}
	clickReplaySetting(t, tt, "十六进制颜色")
	replaceReplaySetting(tt, "FF00AA")
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeyEscape)
	if !app.indicatorsOpen {
		t.Fatal("closing color picker dismissed the whole dialog")
	}
	clickReplaySetting(t, tt, "保存")
	lines := app.chart.indicators.EMALines
	if app.indicatorsOpen || len(lines) != 3 || lines[0].Period != 9 || lines[1].Period != 55 || lines[1].Width != 3 || lines[1].Color != ui.Hex("#ff00aa") || lines[2].Enabled {
		t.Fatalf("saved EMA settings = %+v, open=%v", lines, app.indicatorsOpen)
	}
	for range 3 {
		tt.Frame()
	}
	if !tt.Focused("指标") {
		t.Fatal("save did not restore toolbar focus")
	}
	clickReplaySetting(t, tt, "指标")
	clickReplaySetting(t, tt, "删除 EMA 1")
	clickReplaySetting(t, tt, "EMA 1 周期") // Former second line retains its identity/draft.
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(ui.Cmd, ui.KeyC)
	if tt.Clipboard() != "55" {
		t.Fatalf("surviving line lost its period: %q", tt.Clipboard())
	}
	clickReplaySetting(t, tt, "保存")
	if len(app.chart.indicators.EMALines) != 2 || app.chart.indicators.EMALines[0].ID != lines[1].ID {
		t.Fatal("removal lost remaining line identity")
	}
}

func TestIndicatorDialogDismissalDiscardsAllEdits(t *testing.T) {
	for _, dismissal := range []string{"cancel", "escape", "backdrop", "close"} {
		t.Run(dismissal, func(t *testing.T) {
			app, tt := newReplaySettingsTester()
			before := app.chart.indicators.clone()
			for range 2 {
				clickReplaySetting(t, tt, "指标")
				clickReplaySetting(t, tt, "成交量")
				clickReplaySetting(t, tt, "显示 成交量")
				clickReplaySetting(t, tt, "EMA")
				clickReplaySetting(t, tt, "添加 EMA")
				clickReplaySetting(t, tt, "EMA 1 周期")
				replaceReplaySetting(tt, "99")
				clickReplaySetting(t, tt, "MACD")
				clickReplaySetting(t, tt, "显示 MACD")
				switch dismissal {
				case "cancel":
					clickReplaySetting(t, tt, "取消")
				case "escape":
					tt.Key(0, ui.KeyEscape)
				case "backdrop":
					tt.ClickAt(12, 80)
				case "close":
					clickReplaySetting(t, tt, "关闭指标设置")
				}
				if app.indicatorsOpen || !app.chart.showVolume || !reflect.DeepEqual(app.chart.indicators, before) {
					t.Fatal("dismissal saved edits")
				}
				for range 3 {
					tt.Frame()
				}
				if !tt.Focused("指标") {
					t.Fatal("dismissal did not restore focus")
				}
			}
		})
	}
}

func TestIndicatorDialogSavesVisibilityAndMACDPeriods(t *testing.T) {
	app, tt := newReplaySettingsTester()
	clickReplaySetting(t, tt, "指标")
	clickReplaySetting(t, tt, "EMA")
	clickReplaySetting(t, tt, "显示 EMA")
	clickReplaySetting(t, tt, "MACD")
	clickReplaySetting(t, tt, "显示 MACD")
	for _, edit := range []struct{ label, value string }{{"快线周期", "5"}, {"慢线周期", "35"}, {"信号周期", "7"}} {
		indicatorEditField(t, tt, edit.label, edit.value)
	}
	clickReplaySetting(t, tt, "保存")
	s := app.chart.indicators
	if s.EMAEnabled || !s.MACDEnabled || s.MACDFast != 5 || s.MACDSlow != 35 || s.MACDSignal != 7 {
		t.Fatalf("saved=%+v", s)
	}
	if !slices.Equal(app.chart.indicatorValues.MACD.DIF, market.ComputeMACD(app.candles, 5, 35, 7).DIF) {
		t.Fatal("MACD did not use saved periods")
	}
	clickReplaySetting(t, tt, "指标")
	if !app.indicatorDraft.MACDEnabled || app.indicatorDraft.MACDFast != 5 {
		t.Fatal("reopening lost settings")
	}
}

func indicatorEditField(t *testing.T, tt *ui.Tester, label, value string) {
	t.Helper()
	clickReplaySetting(t, tt, "MACD "+label)
	replaceReplaySetting(tt, value)
}

func TestIndicatorDialogValidatesPeriods(t *testing.T) {
	app, tt := newReplaySettingsTester()
	clickReplaySetting(t, tt, "指标")
	clickReplaySetting(t, tt, "EMA")
	clickReplaySetting(t, tt, "EMA 1 周期")
	replaceReplaySetting(tt, "9.5")
	if app.indicatorDraft.valid() || !tt.HasText(app.text().IndicatorPeriodsInvalid()) {
		t.Fatal("fractional period was accepted")
	}
	clickReplaySetting(t, tt, "保存")
	if !app.indicatorsOpen {
		t.Fatal("disabled save committed invalid periods")
	}
	clickReplaySetting(t, tt, "EMA 1 周期")
	replaceReplaySetting(tt, "9")
	clickReplaySetting(t, tt, "MACD")
	indicatorEditField(t, tt, "快线周期", "26")
	if app.indicatorDraft.valid() || !tt.HasText(app.text().MACDPeriodsInvalid()) {
		t.Fatal("fast >= slow was accepted")
	}
	clickReplaySetting(t, tt, "取消")
	if app.chart.indicators.EMALines[0].Period != 20 {
		t.Fatal("invalid edit changed settings")
	}
	for _, value := range []float64{0, -1, 10001, math.NaN(), math.Inf(1), 1.5} {
		if validIndicatorPeriod(value) {
			t.Fatalf("accepted invalid period %v", value)
		}
	}
}

func TestIndicatorDialogKeepsFocusedDraftAcrossLanguageChanges(t *testing.T) {
	app, tt := newReplaySettingsTester()
	clickReplaySetting(t, tt, "指标")
	clickReplaySetting(t, tt, "EMA")
	clickReplaySetting(t, tt, "EMA 1 周期")
	replaceReplaySetting(tt, "34")
	app.setLanguagePreference(locales.PreferEnglish)
	for range 3 {
		tt.Frame()
	}
	if !tt.Focused("EMA 1 period") || app.indicatorDraft.EMALines[0].Period != 34 {
		t.Fatal("language change lost focused draft")
	}
	clickReplaySetting(t, tt, "Add EMA")
	clickReplaySetting(t, tt, "Save")
	if app.chart.indicators.EMALines[0].Period != 34 || len(app.chart.indicators.EMALines) != 2 {
		t.Fatal("English dialog did not save")
	}
}

func TestIndicatorsCacheAndMarketChanges(t *testing.T) {
	app, _ := newReplaySettingsTester()
	values, generation := app.chart.indicatorValues, app.indicatorCache.Generation
	app.chart.indicators.EMALines[0].Color = ui.Hex("#ff00aa")
	app.chart.indicators.EMAEnabled = false
	app.ensureIndicators()
	if app.chart.indicatorValues != values || app.indicatorCache.Generation != generation {
		t.Fatal("style edit recomputed values")
	}
	app.chart.indicators.EMALines[0].Period = 55
	app.ensureIndicators()
	if app.chart.indicatorValues == values || !slices.Equal(app.chart.indicatorValues.EMA[55], market.ComputeEMA(app.candles, 55)) {
		t.Fatal("period change did not recompute")
	}
	stale := &indicatorValues{}
	app.applyIndicatorValues(generation, stale)
	if app.chart.indicatorValues == stale {
		t.Fatal("stale worker result was accepted")
	}
	settings := app.chart.indicators.clone()
	app.chart.showVolume = false
	for _, interval := range []string{"4h", "15m"} {
		app.applyMarketSnapshot(market.Snapshot{Instrument: builtinInstrumentCatalog[1], Interval: interval, Candles: cloneCandles(app.candles)})
		app.ensureIndicators()
		if !reflect.DeepEqual(app.chart.indicators, settings) || app.chart.showVolume {
			t.Fatal("market change lost indicator settings")
		}
	}
	older := append([]Candle{{Close: 1}}, app.candles...)
	app.applyMarketSnapshot(market.Snapshot{Instrument: app.currentInstrument, Interval: app.chart.timeframe, Candles: older, HistoryPrepended: 1})
	app.ensureIndicators()
	if !slices.Equal(app.chart.indicatorValues.EMA[55], market.ComputeEMA(older, 55)) {
		t.Fatal("history prepend did not reseed indicators")
	}
	app.closeIndicatorCalculations()
	app.applyIndicatorValues(app.indicatorCache.Generation, stale)
	if app.chart.indicatorValues == stale {
		t.Fatal("closed window accepted indicators")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if calculateIndicators(ctx, app.candles, settings.calculation()) != nil {
		t.Fatal("canceled calculation returned results")
	}
}

func TestIndicatorPaneGeometryAndReplayRanges(t *testing.T) {
	for _, height := range []float32{360, 520, 760} {
		for _, volume := range []bool{false, true} {
			for _, macd := range []bool{false, true} {
				g := geometryForIndicators(760, height, volume, macd, true)
				if g.priceHeight < 48 || g.priceBottom > g.bottom || g.top+g.priceHeight > g.priceBottom {
					t.Fatal("invalid price pane", g)
				}
				if volume && (g.volumeTop <= g.priceBottom || g.volumeBottom > g.bottom) {
					t.Fatal("invalid volume pane", g)
				}
				if macd && (g.macdTop <= g.priceBottom || g.macdBottom != g.bottom || volume && g.volumeBottom >= g.macdTop) {
					t.Fatal("invalid MACD pane", g)
				}
			}
		}
	}
	values := &indicatorValues{MACD: market.MACDSeries{DIF: []float64{0, 1e9}, DEA: []float64{0, 1e9}, Histogram: []float64{0, 1e9}}}
	low, high, ok := macdVisibleRange(values, 0, 1)
	if !ok || low != -1 || high != 1 {
		t.Fatal("future MACD values affected replay range")
	}
	candles := []Candle{{Low: 99, High: 101, EMA20: 100}, {Low: 99, High: 101, EMA20: 100}}
	state := newKlineState(len(candles))
	state.Visible = 2
	state.Start = 0
	state.replayHasStart, state.replayEnd = true, 1
	state.indicators.EMALines[0].Period = 50
	state.indicatorValues = &indicatorValues{EMA: map[int][]float64{50: {100, 1e9}}}
	low, high, ok = visibleKlinePriceRange(candles, &state)
	if !ok || high > 102 || low < 98 {
		t.Fatal("future EMA affected replay range")
	}
	state.indicatorValues.EMA[50][0] = 500
	_, high, _ = visibleKlinePriceRange(candles, &state)
	if high < 500 {
		t.Fatal("visible EMA excluded from price range")
	}
	state.indicators.EMALines[0].Enabled = false
	_, high, _ = visibleKlinePriceRange(candles, &state)
	if high > 102 {
		t.Fatal("hidden EMA affected price range")
	}
}

func TestConfiguredIndicatorsRenderAndModalFits(t *testing.T) {
	app, tt := newReplaySettingsTester()
	app.chart.indicators.EMALines = []emaLineSettings{
		{ID: 1, Enabled: true, Period: 9, Color: ui.Hex("#ff00aa"), Width: 3},
		{ID: 2, Enabled: true, Period: 55, Color: ui.Hex("#0055ff"), Width: 3},
	}
	app.chart.indicators.NextEMAID = 2
	app.chart.indicators.MACDEnabled = true
	tt.Frame()
	g := app.chart.geometry(app.chart.chartWidth, app.chart.chartHeight, true)
	frameImage := tt.Image()
	for _, color := range []ui.Color{ui.Hex("#ff00aa"), ui.Hex("#0055ff")} {
		pixels := 0
		for y := int(g.top + 20); y < int(g.priceBottom); y++ {
			for x := int(g.left); x < int(g.left+g.width); x++ {
				pixel := frameImage.RGBAAt(x, y)
				if pixel.R == color.R && pixel.G == color.G && pixel.B == color.B {
					pixels++
				}
			}
		}
		if pixels < 100 {
			t.Fatalf("EMA color %+v only painted %d price-pane pixels", color, pixels)
		}
	}
	before := bytes.Clone(tt.Image().Pix)
	app.chart.indicators.EMAEnabled = false
	tt.Frame()
	if bytes.Equal(before, tt.Image().Pix) {
		t.Fatal("EMA lines did not change pixels")
	}
	app.chart.indicators.EMAEnabled = true
	clickReplaySetting(t, tt, "指标")
	for _, size := range [][2]int{{900, 600}, {760, 520}, {600, 440}} {
		tt.SetSize(size[0], size[1])
		for _, choice := range []string{"成交量", "EMA", "MACD"} {
			clickReplaySetting(t, tt, choice)
			for _, label := range []string{"关闭指标设置", "取消", "保存"} {
				r, ok := tt.Find(label)
				if !ok || r.X < 0 || r.Y < 0 || r.X+r.W > float32(size[0]) || r.Y+r.H > float32(size[1]) {
					t.Fatalf("%s outside %v: %+v", label, size, r)
				}
			}
		}
	}
}

// Optional rendered evidence uses the same real controls exercised above and
// never writes screenshots into the repository during ordinary test runs.
func TestIndicatorVisualEvidence(t *testing.T) {
	dir := os.Getenv("TRAJON_INDICATOR_SHOTS")
	if dir == "" {
		t.Skip("set TRAJON_INDICATOR_SHOTS to capture rendered UI")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	app, tt := newReplaySettingsTester()
	save := func(name string) {
		t.Helper()
		file, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, tt.Image()); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	save("baseline")
	clickReplaySetting(t, tt, "指标")
	save("volume")
	clickReplaySetting(t, tt, "EMA")
	clickReplaySetting(t, tt, "添加 EMA")
	clickReplaySetting(t, tt, "添加 EMA")
	save("ema")
	clickReplaySetting(t, tt, "MACD")
	clickReplaySetting(t, tt, "显示 MACD")
	save("macd")
	clickReplaySetting(t, tt, "保存")
	save("chart")
	tt.SetSize(760, 520)
	clickReplaySetting(t, tt, "指标")
	clickReplaySetting(t, tt, "EMA")
	save("ema-small")
	app.setLanguagePreference(locales.PreferEnglish)
	tt.Frame()
	save("ema-english")
	tt.SetScale(2)
	save("ema-2x")
	tt.SetScale(1)
	tt.SetDark(true)
	save("ema-dark")
}
