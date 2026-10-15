package market

import (
	"reflect"
	"testing"
)

func TestCandleIntervalsAreStableAndIndependentlyOwned(t *testing.T) {
	want := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "1D", "1W", "1M"}
	if got := CandleIntervals(); !reflect.DeepEqual(got, want) {
		t.Fatalf("intervals = %v, want %v", got, want)
	}
	modified := CandleIntervals()
	modified[0] = "changed"
	if !reflect.DeepEqual(CandleIntervals(), want) {
		t.Fatal("caller mutated the shared interval catalog")
	}
}
