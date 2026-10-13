package market

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestReplayStatistics(t *testing.T) {
	settings := ReplayTradingSettings{InitialCapital: 10000, Leverage: 1}
	for _, tc := range []struct {
		name                           string
		trades                         []ReplayTrade
		mark                           float64
		closes, wins, losses, even     int
		profit, loss, realized, unreal float64
		winRate, ratio                 float64
		hasWinRate, hasRatio           bool
	}{
		{name: "no trades", mark: 100},
		{name: "open long", trades: []ReplayTrade{replayTrade(ReplayBuy, 2, 100)}, mark: 130, unreal: 60},
		{name: "open short", trades: []ReplayTrade{replayTrade(ReplaySell, 3, 100)}, mark: 80, unreal: 60},
		{
			name: "partial closes additions shorts and reversal",
			trades: []ReplayTrade{
				replayTrade(ReplayBuy, 2, 100), replayTrade(ReplayBuy, 1, 130),
				replayTrade(ReplaySell, 1, 140), replayTrade(ReplaySell, 3, 90),
				replayTrade(ReplaySell, 1, 110), replayTrade(ReplayBuy, .5, 80),
				replayTrade(ReplayBuy, 1.5, 120),
				replayTrade(ReplayBuy, 1, 100), replayTrade(ReplaySell, 1, 100),
			},
			mark: 200, closes: 5, wins: 2, losses: 2, even: 1,
			profit: 40, loss: 70, realized: -30, winRate: .4, ratio: 20.0 / 35,
			hasWinRate: true, hasRatio: true,
		},
		{
			name: "ratio uses averages not totals",
			trades: []ReplayTrade{
				replayTrade(ReplayBuy, 1, 100), replayTrade(ReplaySell, 1, 120),
				replayTrade(ReplayBuy, 1, 100), replayTrade(ReplaySell, 1, 140),
				replayTrade(ReplayBuy, 1, 100), replayTrade(ReplaySell, 1, 90),
			},
			mark: 200, closes: 3, wins: 2, losses: 1, profit: 60, loss: 10, realized: 50,
			winRate: 2.0 / 3, ratio: 3, hasWinRate: true, hasRatio: true,
		},
		{
			name: "fractional partial close leaves unrealized loss",
			trades: []ReplayTrade{
				replayTrade(ReplayBuy, .5, 100), replayTrade(ReplayBuy, .5, 120), replayTrade(ReplaySell, .25, 130),
			},
			mark: 90, closes: 1, wins: 1, profit: 5, realized: 5, unreal: -15,
			winRate: 1, ratio: math.Inf(1), hasWinRate: true, hasRatio: true,
		},
		{
			name:   "only wins",
			trades: []ReplayTrade{replayTrade(ReplayBuy, 1, 100), replayTrade(ReplaySell, 1, 110)},
			mark:   200, closes: 1, wins: 1, profit: 10, realized: 10,
			winRate: 1, ratio: math.Inf(1), hasWinRate: true, hasRatio: true,
		},
		{
			name:   "only losses",
			trades: []ReplayTrade{replayTrade(ReplaySell, 1, 100), replayTrade(ReplayBuy, 1, 110)},
			mark:   200, closes: 1, losses: 1, loss: 10, realized: -10,
			hasWinRate: true, hasRatio: true,
		},
		{
			name:   "only breakeven",
			trades: []ReplayTrade{replayTrade(ReplayBuy, 1, 100), replayTrade(ReplaySell, 1, 100)},
			mark:   200, closes: 1, even: 1, hasWinRate: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var account ReplayAccount
			for _, trade := range tc.trades {
				if err := account.Execute(settings, trade); err != nil {
					t.Fatal(err)
				}
			}
			before := cloneReplayAccount(account)
			stats := account.Statistics(settings.InitialCapital, tc.mark)
			want := ReplayStatistics{
				InitialCapital: settings.InitialCapital,
				Equity:         settings.InitialCapital + tc.realized + tc.unreal,
				RealizedPnL:    tc.realized,
				UnrealizedPnL:  tc.unreal,
				ClosingTrades:  tc.closes,
				Wins:           tc.wins,
				Losses:         tc.losses,
				Breakeven:      tc.even,
				GrossProfit:    tc.profit,
				GrossLoss:      tc.loss,
			}
			if stats.ClosingTrades != want.ClosingTrades || stats.Wins != want.Wins || stats.Losses != want.Losses || stats.Breakeven != want.Breakeven {
				t.Fatalf("statistics = %+v, want %+v", stats, want)
			}
			assertMetric := func(name string, got float64, defined bool, want float64, wantDefined bool) {
				t.Helper()
				if defined != wantDefined || !(got == want || math.Abs(got-want) <= 1e-9) {
					t.Fatalf("%s = %v (%v), want %v (%v)", name, got, defined, want, wantDefined)
				}
			}
			for _, field := range []struct {
				name      string
				got, want float64
			}{
				{"capital", stats.InitialCapital, want.InitialCapital},
				{"equity", stats.Equity, want.Equity},
				{"realized P&L", stats.RealizedPnL, want.RealizedPnL},
				{"unrealized P&L", stats.UnrealizedPnL, want.UnrealizedPnL},
				{"gross profit", stats.GrossProfit, want.GrossProfit},
				{"gross loss", stats.GrossLoss, want.GrossLoss},
			} {
				assertMetric(field.name, field.got, true, field.want, true)
			}
			rate, defined := stats.WinRate()
			assertMetric("win rate", rate, defined, tc.winRate, tc.hasWinRate)
			ratio, defined := stats.PayoffRatio()
			assertMetric("payoff ratio", ratio, defined, tc.ratio, tc.hasRatio)
			ret, defined := stats.ReturnRate()
			assertMetric("return", ret, defined, (tc.realized+tc.unreal)/settings.InitialCapital, true)
			if !reflect.DeepEqual(account, before) {
				t.Fatal("calculating statistics changed the account or its trades")
			}
		})
	}
}

func TestReplayStatisticsIncludeBracketExitsAndExcludeFailedOrders(t *testing.T) {
	settings := ReplayTradingSettings{InitialCapital: 1000, Leverage: 1}
	var account ReplayAccount
	if err := account.Execute(settings, replayTrade(ReplayBuy, 2, 100)); err != nil {
		t.Fatal(err)
	}
	if err := account.Execute(settings, replayTrade(ReplayBuy, 100, 100)); !errors.Is(err, ErrInsufficientReplayCapital) {
		t.Fatalf("expected rejected order, got %v", err)
	}
	if err := account.SetExit(ReplayTakeProfit, 120, time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := account.CheckExits(settings, Candle{Time: time.Unix(200, 0), Open: 125, High: 130, Low: 120, Close: 125}); err != nil {
		t.Fatal(err)
	}
	if err := account.Execute(settings, replayTrade(ReplaySell, 1, 100)); err != nil {
		t.Fatal(err)
	}
	if err := account.SetExit(ReplayStopLoss, 110, time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := account.CheckExits(settings, Candle{Time: time.Unix(300, 0), Open: 120, High: 125, Low: 100, Close: 110}); err != nil {
		t.Fatal(err)
	}
	stats := account.Statistics(settings.InitialCapital, 110)
	if stats.ClosingTrades != 2 || stats.Wins != 1 || stats.Losses != 1 || stats.RealizedPnL != 30 || stats.GrossProfit != 50 || stats.GrossLoss != 20 {
		t.Fatalf("bracket/failed-order statistics = %+v", stats)
	}
}

func TestReplayStatisticsReturnRejectsInvalidCapital(t *testing.T) {
	for _, capital := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, defined := (ReplayStatistics{InitialCapital: capital}).ReturnRate(); defined {
			t.Fatalf("return defined with invalid initial capital %v", capital)
		}
	}
}
