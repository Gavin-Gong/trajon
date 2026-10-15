package app

import (
	"math"
	"testing"
	"time"

	"trajon/internal/locales"
	"trajon/internal/market"
)

func TestReplayPositionLineAndCloseUseWholePositionAtLatestClose(t *testing.T) {
	for _, side := range []market.ReplaySide{market.ReplayBuy, market.ReplaySell} {
		t.Run(replayPositionCaption(map[market.ReplaySide]float64{market.ReplayBuy: 2, market.ReplaySell: -2}[side], locales.In(string(locales.Chinese))), func(t *testing.T) {
			app, tt := newKlineToolbarTester()
			app.chart.Start = 100
			app.chart.selectReplayStart(120, len(app.candles))
			app.candles[120].Close = 100
			app.candles[121].Close = 110
			app.chart.replayOrderQuantity = 2
			if _, err := app.placeReplayTrade(side); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			caption := app.replayPositionCaption(app.chart.replayAccount.Quantity)
			if _, ok := tt.Find(caption); !ok {
				t.Fatalf("position tag %q missing", caption)
			}
			line, ok := tt.Find("仓位线")
			closeButton, closeOK := tt.Find("平仓")
			g := currentKlineGeometry(app)
			low, high, _ := visibleKlinePriceRange(app.candles, &app.chart)
			wantY := g.top + float32((high-100)/(high-low))*g.priceHeight
			if !ok || !closeOK || math.Abs(float64(line.Y+line.H/2-wantY)) > .01 || closeButton.X < line.X || closeButton.X+closeButton.W > line.X+line.W {
				t.Fatalf("position line or close button misplaced: line=%+v close=%+v wantY=%v", line, closeButton, wantY)
			}
			clickReplaySetting(t, tt, "播放")
			app.chart.replayNextStep = time.Now().Add(time.Hour)
			nextStep := app.chart.replayNextStep
			app.advanceReplayStep()
			app.chart.replayOrderQuantity = 99 // Close must ignore the order input.
			tt.Frame()
			clickReplaySetting(t, tt, "平仓")
			account := app.chart.replayAccount
			trade := account.Trades[1]
			if account.Quantity != 0 || account.AveragePrice != 0 || trade.Quantity != 2 || trade.Price != 110 || !trade.Time.Equal(app.candles[121].Time) {
				t.Fatalf("close did not use the entire position and latest candle: %+v", account)
			}
			if !app.chart.actionPlaying || !app.chart.replayNextStep.Equal(nextStep) || app.chart.replayOrderQuantity != 99 {
				t.Fatal("closing changed playback or the order input")
			}
			for range 3 {
				tt.Frame()
			}
			if _, ok := tt.Find("仓位线"); ok {
				t.Fatal("flat account still has a position line")
			}
		})
	}
}

func TestReplayPositionLineTracksAverageAndDirection(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.Start = 100
	app.chart.selectReplayStart(120, len(app.candles))
	for _, tc := range []struct {
		side                  market.ReplaySide
		quantity, price, held float64
		average               float64
	}{
		{market.ReplayBuy, 2, 100, 2, 100},
		{market.ReplayBuy, 2, 120, 4, 110},
		{market.ReplaySell, 1, 125, 3, 110},
		{market.ReplaySell, 5, 130, -2, 130},
	} {
		app.candles[120].Close = tc.price
		app.chart.replayOrderQuantity = tc.quantity
		if _, err := app.placeReplayTrade(tc.side); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		line, ok := tt.Find("仓位线")
		if _, captionOK := tt.Find(app.replayPositionCaption(tc.held)); !captionOK || !ok {
			t.Fatalf("position line/tag missing for quantity %v", tc.held)
		}
		low, high, _ := visibleKlinePriceRange(app.candles, &app.chart)
		g := currentKlineGeometry(app)
		price := priceForY(line.Y+line.H/2, g, low, high)
		if math.Abs(price-tc.average) > .001 {
			t.Fatalf("position line price=%v, want average %v", price, tc.average)
		}
	}
}

func TestReplayPositionLineRemainsVisibleAwayFromCurrentCandles(t *testing.T) {
	app, tt := newKlineToolbarTester()
	app.chart.selectReplayStart(120, len(app.candles))
	app.chart.replayOrderQuantity = 1
	if _, err := app.placeReplayTrade(market.ReplayBuy); err != nil {
		t.Fatal(err)
	}
	app.chart.Start = 500 // The viewport is past the replay cutoff.
	tt.Frame()
	line, ok := tt.Find("仓位线")
	g := currentKlineGeometry(app)
	if !ok || line.Y < g.top || line.Y+line.H > g.priceBottom {
		t.Fatalf("position line disappeared with no visible candles: %+v", line)
	}
}
