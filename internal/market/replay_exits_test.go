package market

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func bracketedReplayAccount(t *testing.T, side ReplaySide) (ReplayAccount, ReplayTradingSettings, time.Time) {
	t.Helper()
	settings := ReplayTradingSettings{InitialCapital: 1000, Leverage: 1}
	stamp := time.Unix(100, 0)
	var account ReplayAccount
	if err := account.Execute(settings, ReplayTrade{Side: side, Quantity: 2, Price: 100, Time: stamp}); err != nil {
		t.Fatal(err)
	}
	tp, sl := 110.0, 90.0
	if side == ReplaySell {
		tp, sl = sl, tp
	}
	if err := account.SetExit(ReplayTakeProfit, tp, stamp); err != nil {
		t.Fatal(err)
	}
	if err := account.SetExit(ReplayStopLoss, sl, stamp); err != nil {
		t.Fatal(err)
	}
	return account, settings, stamp
}

func TestReplayExitsTouchGapAndStopLossPriority(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		side                   ReplaySide
		open, high, low, close float64
		kind                   ReplayExitKind
		price                  float64
	}{
		{"long take profit", ReplayBuy, 100, 115, 95, 105, ReplayTakeProfit, 110},
		{"long stop loss", ReplayBuy, 100, 105, 85, 95, ReplayStopLoss, 90},
		{"short take profit", ReplaySell, 100, 105, 85, 95, ReplayTakeProfit, 90},
		{"short stop loss", ReplaySell, 100, 115, 95, 105, ReplayStopLoss, 110},
		{"long profit gap", ReplayBuy, 120, 125, 115, 120, ReplayTakeProfit, 120},
		{"long loss gap", ReplayBuy, 80, 85, 75, 80, ReplayStopLoss, 80},
		{"short profit gap", ReplaySell, 80, 85, 75, 80, ReplayTakeProfit, 80},
		{"short loss gap", ReplaySell, 120, 125, 115, 120, ReplayStopLoss, 120},
		{"long both", ReplayBuy, 100, 120, 80, 100, ReplayStopLoss, 90},
		{"short both", ReplaySell, 100, 120, 80, 100, ReplayStopLoss, 110},
		{"long exact touch", ReplayBuy, 100, 110, 95, 100, ReplayTakeProfit, 110},
		{"short exact touch", ReplaySell, 100, 105, 90, 100, ReplayTakeProfit, 90},
		{"no touch", ReplayBuy, 100, 105, 95, 100, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account, settings, stamp := bracketedReplayAccount(t, tc.side)
			candle := Candle{Time: stamp.Add(time.Hour), Open: tc.open, High: tc.high, Low: tc.low, Close: tc.close}
			trade, err := account.CheckExits(settings, candle)
			if err != nil || trade.ExitKind != tc.kind || trade.Price != tc.price {
				t.Fatalf("trade=%+v error=%v, want kind=%v price=%v", trade, err, tc.kind, tc.price)
			}
			if tc.kind == 0 {
				if len(account.Trades) != 1 || math.Abs(account.Quantity) != 2 {
					t.Fatal("untouched brackets changed the position")
				}
				return
			}
			if trade.Quantity != 2 || trade.Side == tc.side || !trade.Time.Equal(candle.Time) || account.Quantity != 0 || account.TakeProfit != (ReplayExit{}) || account.StopLoss != (ReplayExit{}) {
				t.Fatalf("exit did not fully close and clear brackets: %+v", account)
			}
			if len(account.Trades) != 2 || account.Trades[1] != trade {
				t.Fatal("exit was not recorded exactly once")
			}
			if _, err := account.CheckExits(settings, candle); err != nil || len(account.Trades) != 2 {
				t.Fatal("checking an already closed position duplicated the fill")
			}
		})
	}
}

func TestReplayExitsOnlyUseCandlesAfterCreationOrModification(t *testing.T) {
	account, settings, stamp := bracketedReplayAccount(t, ReplayBuy)
	candle := Candle{Time: stamp, Open: 100, High: 120, Low: 80, Close: 100}
	if trade, err := account.CheckExits(settings, candle); err != nil || trade.ExitKind != 0 {
		t.Fatal("new brackets triggered retroactively on the already revealed candle")
	}
	modifiedAt := stamp.Add(time.Hour)
	if err := account.SetExit(ReplayTakeProfit, 115, modifiedAt); err != nil {
		t.Fatal(err)
	}
	candle = Candle{Time: modifiedAt, Open: 100, High: 120, Low: 95, Close: 100}
	if trade, err := account.CheckExits(settings, candle); err != nil || trade.ExitKind != 0 {
		t.Fatal("modified bracket triggered against the current candle")
	}
	candle.Time = modifiedAt.Add(time.Hour)
	if trade, err := account.CheckExits(settings, candle); err != nil || trade.ExitKind != ReplayTakeProfit || trade.Price != 115 {
		t.Fatalf("next candle did not use the modified bracket: %+v %v", trade, err)
	}
}

func TestReplayBracketsFollowRemainingPositionAndClearOnReversal(t *testing.T) {
	account, settings, stamp := bracketedReplayAccount(t, ReplayBuy)
	tp, sl := account.TakeProfit, account.StopLoss
	for _, trade := range []ReplayTrade{replayTrade(ReplayBuy, 1, 100), replayTrade(ReplaySell, 1, 100)} {
		if err := account.Execute(settings, trade); err != nil {
			t.Fatal(err)
		}
		if account.TakeProfit != tp || account.StopLoss != sl {
			t.Fatal("adding/reducing the position changed bracket prices")
		}
	}
	candle := Candle{Time: stamp.Add(time.Hour), Open: 100, High: 115, Low: 95, Close: 105}
	if trade, err := account.CheckExits(settings, candle); err != nil || trade.Quantity != 2 {
		t.Fatalf("exit did not close the remaining quantity: %+v %v", trade, err)
	}
	account, settings, _ = bracketedReplayAccount(t, ReplayBuy)
	if err := account.Execute(settings, replayTrade(ReplaySell, 3, 100)); err != nil {
		t.Fatal(err)
	}
	if account.Quantity != -1 || account.TakeProfit != (ReplayExit{}) || account.StopLoss != (ReplayExit{}) {
		t.Fatal("reversal kept brackets from the previous direction")
	}
}

func TestReplayExitCancellationAndInvalidPrices(t *testing.T) {
	for _, side := range []ReplaySide{ReplayBuy, ReplaySell} {
		account, _, stamp := bracketedReplayAccount(t, side)
		before := cloneReplayAccount(account)
		for _, price := range []float64{0, -1, 100, math.NaN(), math.Inf(1)} {
			if err := account.SetExit(ReplayTakeProfit, price, stamp); !errors.Is(err, ErrInvalidReplayExit) || !reflect.DeepEqual(account, before) {
				t.Fatalf("invalid price %v changed the bracket", price)
			}
		}
		if err := account.SetExit(ReplayTakeProfit, account.StopLoss.Price, stamp); !errors.Is(err, ErrInvalidReplayExit) {
			t.Fatal("take profit accepted a price on the loss side")
		}
		if err := account.SetExit(ReplayStopLoss, account.TakeProfit.Price, stamp); !errors.Is(err, ErrInvalidReplayExit) {
			t.Fatal("stop loss accepted a price on the profit side")
		}
		if err := account.SetExit(0, 120, stamp); !errors.Is(err, ErrInvalidReplayExit) {
			t.Fatal("unknown exit kind was accepted")
		}
		account.CancelExit(ReplayTakeProfit)
		if account.TakeProfit != (ReplayExit{}) || account.StopLoss != before.StopLoss || account.Quantity != before.Quantity {
			t.Fatal("canceling take profit changed stop loss or the position")
		}
		account.CancelExit(ReplayStopLoss)
		if account.StopLoss != (ReplayExit{}) {
			t.Fatal("stop loss was not canceled")
		}
	}
}

func TestReplayExitFailureDoesNotMutatePositionOrBrackets(t *testing.T) {
	account, settings, stamp := bracketedReplayAccount(t, ReplayBuy)
	before := cloneReplayAccount(account)
	candle := Candle{Time: stamp.Add(time.Hour), Open: 100, High: 115, Low: 95, Close: 105}
	settings.InitialCapital = 0
	if _, err := account.CheckExits(settings, candle); !errors.Is(err, ErrInvalidReplaySettings) || !reflect.DeepEqual(account, before) {
		t.Fatal("failed exit mutated the account")
	}
	settings.InitialCapital = 1000
	candle.Low = math.NaN()
	if _, err := account.CheckExits(settings, candle); !errors.Is(err, ErrInvalidReplayTrade) || !reflect.DeepEqual(account, before) {
		t.Fatal("invalid candle mutated the account")
	}
}

func TestReplayCloseClearsBracketsAndRejectsFlatPosition(t *testing.T) {
	account, settings, stamp := bracketedReplayAccount(t, ReplaySell)
	trade, err := account.Close(settings, 80, stamp.Add(time.Hour))
	if err != nil || trade.Quantity != 2 || trade.Side != ReplayBuy || account.Quantity != 0 || account.TakeProfit != (ReplayExit{}) || account.StopLoss != (ReplayExit{}) {
		t.Fatalf("manual close: trade=%+v account=%+v error=%v", trade, account, err)
	}
	if _, err := account.Close(settings, 80, stamp); !errors.Is(err, ErrNoReplayPosition) || len(account.Trades) != 2 {
		t.Fatal("flat position accepted another close")
	}
	if err := account.SetExit(ReplayTakeProfit, 90, stamp); !errors.Is(err, ErrNoReplayPosition) {
		t.Fatal("flat position accepted a bracket")
	}
}
