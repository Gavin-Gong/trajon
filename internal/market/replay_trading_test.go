package market

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func replayTrade(side ReplaySide, quantity, price float64) ReplayTrade {
	return ReplayTrade{Side: side, Quantity: quantity, Price: price, Time: time.Unix(100, 0)}
}

func TestReplayAccountNetPositionAndRealizedPnL(t *testing.T) {
	settings := ReplayTradingSettings{InitialCapital: 1000, Leverage: 1}
	var account ReplayAccount
	for i, tc := range []struct {
		trade                  ReplayTrade
		quantity, average, pnl float64
	}{
		{replayTrade(ReplayBuy, 2, 100), 2, 100, 0},
		{replayTrade(ReplayBuy, 1, 130), 3, 110, 0},
		{replayTrade(ReplaySell, 1, 140), 2, 110, 30},
		{replayTrade(ReplaySell, 3, 90), -1, 90, -10},
		{replayTrade(ReplaySell, 1, 110), -2, 100, -10},
		{replayTrade(ReplayBuy, .5, 80), -1.5, 100, 0},
		{replayTrade(ReplayBuy, 1.5, 120), 0, 0, -30},
	} {
		if err := account.Execute(settings, tc.trade); err != nil {
			t.Fatal(err)
		}
		if math.Abs(account.Quantity-tc.quantity) > 1e-9 || math.Abs(account.AveragePrice-tc.average) > 1e-9 || math.Abs(account.RealizedPnL-tc.pnl) > 1e-9 {
			t.Fatalf("trade %d produced account %+v, want quantity=%v average=%v pnl=%v", i, account, tc.quantity, tc.average, tc.pnl)
		}
		if len(account.Trades) != i+1 || account.Trades[i] != tc.trade {
			t.Fatal("successful trade was not recorded exactly once")
		}
		wantEquity := settings.InitialCapital + tc.pnl + (tc.trade.Price-tc.average)*tc.quantity
		if got := account.Equity(settings.InitialCapital, tc.trade.Price); math.Abs(got-wantEquity) > 1e-9 {
			t.Fatalf("equity = %v, want %v", got, wantEquity)
		}
	}
}

func TestReplayAccountCapitalAndLeverage(t *testing.T) {
	for _, side := range []ReplaySide{ReplayBuy, ReplaySell} {
		var account ReplayAccount
		settings := ReplayTradingSettings{InitialCapital: 1000, Leverage: 2}
		if err := account.Execute(settings, replayTrade(side, 20, 100)); err != nil {
			t.Fatalf("order at the leverage limit: %v", err)
		}
		before := cloneReplayAccount(account)
		if err := account.Execute(settings, replayTrade(side, .01, 100)); !errors.Is(err, ErrInsufficientReplayCapital) {
			t.Fatalf("excess exposure error = %v", err)
		}
		if !reflect.DeepEqual(account, before) {
			t.Fatal("rejected order mutated the account")
		}
	}
}

func TestReplayAccountUsesUnrealizedAndRealizedPnLForCapital(t *testing.T) {
	settings := ReplayTradingSettings{InitialCapital: 1000, Leverage: 2}
	var account ReplayAccount
	if err := account.Execute(settings, replayTrade(ReplayBuy, 10, 100)); err != nil {
		t.Fatal(err)
	}
	// At 200 the first position has gained 1000, allowing another 10 units.
	if err := account.Execute(settings, replayTrade(ReplayBuy, 10, 200)); err != nil {
		t.Fatalf("unrealized profit did not increase buying power: %v", err)
	}
	if err := account.Execute(settings, replayTrade(ReplaySell, 20, 200)); err != nil {
		t.Fatal(err)
	}
	if account.RealizedPnL != 1000 {
		t.Fatalf("realized profit = %v", account.RealizedPnL)
	}
	if err := account.Execute(settings, replayTrade(ReplaySell, 20, 200)); err != nil {
		t.Fatalf("realized profit did not increase buying power: %v", err)
	}
}

func TestReplayAccountAllowsClosingAfterLosses(t *testing.T) {
	for _, tc := range []struct {
		open, close ReplaySide
		price       float64
	}{
		{ReplayBuy, ReplaySell, 10},
		{ReplaySell, ReplayBuy, 190},
	} {
		settings := ReplayTradingSettings{InitialCapital: 1000, Leverage: 2}
		var account ReplayAccount
		if err := account.Execute(settings, replayTrade(tc.open, 20, 100)); err != nil {
			t.Fatal(err)
		}
		if err := account.Execute(settings, replayTrade(tc.close, 5, tc.price)); err != nil {
			t.Fatalf("reducing a losing position: %v", err)
		}
		if err := account.Execute(settings, replayTrade(tc.close, 15, tc.price)); err != nil {
			t.Fatalf("closing a losing position: %v", err)
		}
		if account.Quantity != 0 || account.AveragePrice != 0 || account.RealizedPnL != -1800 {
			t.Fatalf("loss-making close: %+v", account)
		}
		if err := account.Execute(settings, replayTrade(tc.open, 1, tc.price)); !errors.Is(err, ErrInsufficientReplayCapital) {
			t.Fatalf("opening with negative equity: %v", err)
		}
	}
}

func TestReplayAccountRejectsUnaffordableReversalAtomically(t *testing.T) {
	settings := ReplayTradingSettings{InitialCapital: 100, Leverage: 1}
	var account ReplayAccount
	if err := account.Execute(settings, replayTrade(ReplayBuy, 1, 100)); err != nil {
		t.Fatal(err)
	}
	before := cloneReplayAccount(account)
	if err := account.Execute(settings, replayTrade(ReplaySell, 3, 100)); !errors.Is(err, ErrInsufficientReplayCapital) {
		t.Fatalf("reversal error = %v", err)
	}
	if !reflect.DeepEqual(account, before) {
		t.Fatal("rejected reversal partially closed the long position")
	}
	if err := account.Execute(settings, replayTrade(ReplaySell, 2, 100)); err != nil || account.Quantity != -1 {
		t.Fatalf("affordable reversal: account=%+v error=%v", account, err)
	}
}

func TestReplayAccountRejectsInvalidOrdersAndSettings(t *testing.T) {
	validSettings := ReplayTradingSettings{InitialCapital: 1000, Leverage: 2}
	validTrade := replayTrade(ReplayBuy, 1, 100)
	for _, value := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, field := range []string{"quantity", "price", "capital", "leverage"} {
			settings, trade := validSettings, validTrade
			want := ErrInvalidReplayTrade
			switch field {
			case "quantity":
				trade.Quantity = value
			case "price":
				trade.Price = value
			case "capital":
				settings.InitialCapital = value
				want = ErrInvalidReplaySettings
			case "leverage":
				settings.Leverage = value
				want = ErrInvalidReplaySettings
			}
			var account ReplayAccount
			if err := account.Execute(settings, trade); !errors.Is(err, want) {
				t.Fatalf("%s=%v: error=%v, want %v", field, value, err, want)
			}
			if !reflect.DeepEqual(account, ReplayAccount{}) {
				t.Fatal("invalid order changed the account")
			}
		}
	}
	for _, trade := range []ReplayTrade{
		replayTrade(0, 1, 100), replayTrade(99, 1, 100),
		{Side: ReplayBuy, Quantity: 1, Price: 100},
		replayTrade(ReplayBuy, math.MaxFloat64, math.MaxFloat64),
	} {
		var account ReplayAccount
		if err := account.Execute(validSettings, trade); !errors.Is(err, ErrInvalidReplayTrade) {
			t.Fatalf("invalid trade %+v: %v", trade, err)
		}
		if !reflect.DeepEqual(account, ReplayAccount{}) {
			t.Fatal("invalid order changed the account")
		}
	}
}

func cloneReplayAccount(account ReplayAccount) ReplayAccount {
	account.Trades = append([]ReplayTrade(nil), account.Trades...)
	return account
}
